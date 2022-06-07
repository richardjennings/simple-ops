package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	cp "github.com/otiai10/copy"
	"github.com/richardjennings/simple-ops/pkg/compare"
	"github.com/richardjennings/simple-ops/pkg/config"
	"github.com/spf13/afero"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"io/fs"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"strings"

	"os"
)

const (
	defaultFilePerm = 0655
	defaultDirPerm  = 0755
)

type (
	Svc struct {
		appFs  afero.Afero
		client *action.Install
		wd     string
		tmp    string
		wpaths []string
	}
)

// NewSvc creates a new Manifest Service which transforms
// deployment config into rendered manifests
func NewSvc(fs afero.Fs, wd string) *Svc {
	cfg := &action.Configuration{}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ClientOnly = true
	client.IncludeCRDs = true
	return &Svc{appFs: afero.Afero{Fs: fs}, wd: wd, client: client}
}

// Verify generates manifests in a temporary directory and
// compares the sha of those files to the sha of the current
// deploy folder contents. If the sha values do not match,
// verify return false. It does not currently handle verifying
// file generated via with => path.
func (s Svc) Verify(deploys map[string]config.Deploys) (bool, error) {
	var err error
	err = s.generate(deploys)
	if err != nil {
		return false, err
	}
	defer func() {
		if s.tmp != "" {
			err = s.appFs.RemoveAll(s.tmp)
		}
	}()

	// do sha comparisons
	cmp := compare.NewSvc(s.appFs.Fs)

	tmpHash, err := cmp.SHA256(filepath.Join(s.tmp, config.DeployPath))
	if err != nil {
		return false, err
	}

	depHash, err := cmp.SHA256(filepath.Join(s.wd, config.DeployPath))
	if err != nil {
		return false, err
	}

	return tmpHash == depHash, nil
}

// Generate generates manifests in a temporary directory and
// copies the content into the deployment directory if the generation
// process completes successfully.
func (s Svc) Generate(deploys map[string]config.Deploys) error {
	var err error
	err = s.generate(deploys)
	defer func() {
		if err != nil {
			err = s.appFs.RemoveAll(s.tmp)
		}
	}()
	if err != nil {
		return err
	}
	err = s.renameDirectory(s.tmp, s.wd)
	return err
}

// Pull adds a tgz chart to charts from repoUrl with chartRef and version
// addConfig generates a config stub for the chart
func (s Svc) Pull(chartRef string, repoUrl string, version string, addConfig bool) error {
	c := action.Configuration{}
	p := action.NewPullWithOpts(action.WithConfig(&c))
	p.DestDir = s.wd + string(os.PathSeparator) + config.ChartsPath
	p.Untar = false
	p.RepoURL = repoUrl
	p.Version = version
	p.Settings = &cli.EnvSettings{}
	out, err := p.Run(chartRef)
	if err != nil {
		return err
	}
	_ = out // @todo log output
	if addConfig == true {
		conf := "chart: " + chartRef + "-" + version + ".tgz"
		path := s.wd + string(os.PathSeparator) + config.ConfPath + string(os.PathSeparator) + chartRef + ".yml"
		if err := ioutil.WriteFile(path, []byte(conf), defaultFilePerm); err != nil {
			return err
		}
	}
	return nil
}

func (s *Svc) generate(components map[string]config.Deploys) error {
	var err error
	s.tmp, err = s.appFs.TempDir("", "simple-ops-")
	if err != nil {
		return err
	}
	for _, deploys := range components {
		for _, deploy := range deploys {
			if err := s.generateDeploy(deploy); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Svc) generateDeploy(deploy *config.Deploy) error {
	var chrt *chart.Chart
	var rel *release.Release
	var err error
	var t []byte
	var rendered bytes.Buffer

	if chrt, err = s.loadChart(deploy); err != nil {
		return err
	}

	if err := s.appFs.MkdirAll(s.pathForTmpComponent(deploy), defaultDirPerm); err != nil {
		return err
	}

	// optionally create namespace manifest
	if deploy.Namespace.Create {
		if t, err = s.createNamespaceManifest(deploy); err != nil {
			return err
		}
		rendered.Write(t)
	}
	s.client.ReleaseName = deploy.Component
	s.client.Namespace = deploy.Namespace.Name
	s.client.CreateNamespace = false

	// render the helm chart
	rel, err = s.client.Run(chrt, deploy.Values)
	if err != nil {
		return err
	}
	rendered.Write([]byte(rel.Manifest))

	// with
	if deploy.With != nil {
		for p, withs := range deploy.With {
			for name, with := range withs {
				if with.Path == "" {
					t, err = s.generateWith(p, with, name)
					if err != nil {
						return err
					}
					rendered.Write([]byte("---\n"))
					rendered.Write([]byte(fmt.Sprintf("# Source: simple-ops with %s.yml\n", p)))
					rendered.Write(t)
				} else {
					if err := s.generateWithToPath(p, with, name); err != nil {
						return err
					}
				}
			}
		}
	}

	// inject namespace
	if deploy.Namespace.Inject {
		if t, err = s.injectNamespace(deploy, rendered.Bytes()); err != nil {
			return err
		}
	} else {
		t = rendered.Bytes()
	}

	// write manifest
	return s.appFs.WriteFile(s.pathForTmpManifest(deploy), t, defaultFilePerm)
}

// generateWith uses file named with/{n}.yml as a template rendered
// using with Values to a byte slice. With Path must be empty
func (s Svc) generateWith(n string, w config.With, name string) ([]byte, error) {
	if w.Path != "" {
		return nil, errors.New("unexpected path")
	}
	return s.renderWith(n, w, name)
}

// gnerateWithPath uses file name with/{n}.yml as a template rendered
// using with Values to the non-empty path specified relative to the
// working directory, e.g. apps/n.yaml
func (s Svc) generateWithToPath(n string, w config.With, name string) error {
	if w.Path == "" {
		return errors.New("expected path")
	}
	path, err := s.withPath(filepath.Join(s.tmp, w.Path))
	if err != nil {
		return err
	}

	// render
	b, err := s.renderWith(n, w, name)
	if err != nil {
		return err
	}
	// write bytes b to temporary directory

	dir := filepath.Dir(path)
	if err := s.appFs.MkdirAll(dir, defaultDirPerm); err != nil {
		return err
	}
	if err := s.appFs.WriteFile(path, b, defaultFilePerm); err != nil {
		return err
	}
	return nil
}

// renderWith uses file at /with/n.yml
func (s Svc) renderWith(n string, w config.With, name string) ([]byte, error) {
	var c []byte
	var v map[string]interface{}
	var err error
	path := filepath.Join(s.wd, config.WithPath, n) + config.Suffix
	if c, err = s.appFs.ReadFile(path); err != nil {
		return c, err
	}
	// marshal bytes to map[string]interface{]
	if err := yaml.Unmarshal(c, &v); err != nil {
		return nil, err
	}
	// add name to data
	if w.Values == nil {
		w.Values = make(map[string]interface{})
	}
	if _, ok := w.Values["metadata"]; !ok {
		w.Values["metadata"] = make(map[string]interface{})
	}
	// name overwrites any existing
	w.Values["metadata"].(map[string]interface{})["name"] = name
	// merge values from with into v
	v = config.MergeMaps(v, w.Values)
	// marshal to bytes
	return yaml.Marshal(v)
}

func (s Svc) withPath(path string) (string, error) {
	// within s.wp
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, s.tmp) {
		return "", errors.New("path cannot be outside working directory")
	}
	// error if a duplicate path
	if _, err := s.appFs.Stat(path); err == nil {
		return "", fmt.Errorf("with template path duplicate: %s", strings.TrimPrefix(path, s.tmp))
	}
	return path, nil
}

func (s Svc) loadChart(deploy *config.Deploy) (*chart.Chart, error) {
	var chrt *chart.Chart
	var err error

	// if using memfs under test use LoadArchive with archive file
	// The directory handling code in Helm cannot be persuaded to
	// use the fs abstraction. @todo better
	if _, ok := s.appFs.Fs.(*afero.MemMapFs); ok {
		f, err := s.appFs.Open(s.pathForChart(deploy.Chart))
		if err != nil {
			return nil, err
		}
		defer func() {
			err = f.Close()
		}()
		chrt, err = loader.LoadArchive(f)
	} else {
		chrt, err = loader.Load(s.pathForChart(deploy.Chart))
		if err != nil {
			return nil, err
		}
	}

	// check chart dependencies
	// @todo
	if len(chrt.Dependencies()) != len(chrt.Metadata.Dependencies) {
		return nil, errors.New("dependencies not installed")
	}

	return chrt, err
}

func (s Svc) createNamespaceManifest(deploy *config.Deploy) ([]byte, error) {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deploy.Namespace.Name,
			Labels: map[string]string{
				"name": deploy.Namespace.Name,
			},
		},
	}
	for k, v := range deploy.Namespace.Labels {
		ns.ObjectMeta.Labels[k] = v
	}
	return yaml.Marshal(ns)
}

func (s Svc) injectNamespace(deploy *config.Deploy, manifest []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBuffer(manifest)}},
		Filters: []kio.Filter{namespace.Filter{Namespace: deploy.Namespace.Name, FsSlice: types.FsSlice{}}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
	}.Execute()
	return buf.Bytes(), err
}

func (s Svc) pathForChart(p string) string {
	return s.wd + string(os.PathSeparator) + config.ChartsPath + string(os.PathSeparator) + p
}

// os.Rename does not work if the rename crosses file systems
// afero does not move files with a directory change when using memfs
// so here we are
func (s Svc) renameDirectory(from string, to string) error {
	switch s.appFs.Fs.(type) {
	case *afero.OsFs:
		if err := os.RemoveAll(filepath.Join(to, config.DeployPath)); err != nil {
			return err
		}
		if err := s.appFs.MkdirAll(to, defaultDirPerm); err != nil {
			return err
		}
		return cp.Copy(from, to, cp.Options{AddPermission: defaultFilePerm, PreserveOwner: true})
	case *afero.MemMapFs:
		// move files (not a fan of this)
		// string prefix should be ok because we are inside path already
		if err := s.appFs.Walk(from, func(path string, info fs.FileInfo, err error) error {
			ppath := filepath.Join(to, strings.TrimPrefix(path, from))
			if info.IsDir() {
				if err := s.appFs.MkdirAll(ppath, 0655); err != nil {
					return err
				}
			} else {
				if err := s.appFs.Rename(path, ppath); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		// remove previous
		return s.appFs.Remove(from)
	default:
		return errors.New("unsupported aero type")
	}
}

func (s Svc) pathForTmpComponent(d *config.Deploy) string {
	return pathForTmpDeploy(d, s.tmp) + string(os.PathSeparator) + d.Component
}

func pathForTmpDeploy(d *config.Deploy, tmpDir string) string {
	return tmpDir + string(os.PathSeparator) + config.DeployPath + string(os.PathSeparator) + d.Name
}

// /tmp/dir/deploy/prod/component/manifest.yml
func (s Svc) pathForTmpManifest(d *config.Deploy) string {
	return s.pathForTmpComponent(d) + string(os.PathSeparator) + "manifest.yaml"
}
