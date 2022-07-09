package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/google/go-jsonnet"
	cp "github.com/otiai10/copy"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/hash"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"io"
	"io/fs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/filters/labels"
	"sigs.k8s.io/kustomize/api/filters/namespace"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sort"
	"strings"
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
		log    *logrus.Logger
	}
)

// NewSvc creates a new Manifest Service which transforms
// deployment config into rendered manifests
func NewSvc(fs afero.Fs, wd string, log *logrus.Logger) *Svc {
	config := &action.Configuration{}
	client := action.NewInstall(config)
	client.DryRun = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	return &Svc{appFs: afero.Afero{Fs: fs}, wd: wd, client: client, log: log}
}

// Verify generates manifests in a temporary directory and
// compares the sha of those files to the sha of the current
// deploy folder contents. If the sha values do not match,
// verify return false. It does not currently handle verifying
// file generated via with => path.
func (s Svc) Verify(deploys cfg.Deploys) (bool, error) {
	var err error
	err = s.doGenerate(deploys)
	if err != nil {
		return false, err
	}
	defer func() {
		if s.tmp != "" {
			err = s.appFs.RemoveAll(s.tmp)
		}
	}()

	// do sha comparisons
	cmp := hash.NewSvc(s.appFs.Fs, s.log)

	tmpHash, err := cmp.SHA256(filepath.Join(s.tmp, cfg.DeployPath))
	if err != nil {
		return false, err
	}

	depHash, err := cmp.SHA256(filepath.Join(s.wd, cfg.DeployPath))
	if err != nil {
		return false, err
	}

	return tmpHash == depHash, nil
}

// Generate generates manifests in a temporary directory and
// copies the content into the deployment directory if the generation
// process completes successfully.
func (s Svc) Generate(deploys cfg.Deploys) error {
	var err error
	err = s.doGenerate(deploys)
	defer func() {
		if err != nil {
			err = s.appFs.RemoveAll(s.tmp)
		}
	}()
	if err != nil {
		return err
	}
	err = s.renameDirectory(s.tmp, s.wd)
	s.log.Debugf("performed rename on %s to %s", s.tmp, s.wd)

	return err
}

// Pull adds a tgz chart to charts from repoUrl with chartRef and version
// addConfig generates a config stub for the chart
func (s Svc) Pull(chartRef string, repoUrl string, version string) error {
	p, err := s.doPull(repoUrl, version)
	if err != nil {
		return err
	}
	out, err := p.Run(chartRef)
	if err != nil {
		return err
	}
	if out != "" {
		s.log.Debugf("helm doPull: %s\n", out)
	}
	s.log.Debugf("saved chart %s-%s.tgz to %s", chartRef, version, p.DestDir)

	return nil
}

func (s Svc) doPull(repoUrl string, version string) (*action.Pull, error) {
	c := action.Configuration{}
	p := action.NewPullWithOpts(action.WithConfig(&c))
	p.DestDir = s.wd + string(os.PathSeparator) + cfg.ChartsPath
	p.Untar = false
	p.RepoURL = repoUrl
	p.Version = version
	p.Settings = &cli.EnvSettings{}
	return p, nil
}

func (s Svc) PullAddConfig(chartRef string, version string) error {
	conf := "chart: " + chartRef + "-" + version + ".tgz"
	path := filepath.Join(s.wd, cfg.ConfPath, chartRef+cfg.Suffix)
	if err := s.appFs.WriteFile(path, []byte(conf), defaultFilePerm); err != nil {
		return err
	}
	s.log.Debugf("added config file for chart %s-%s.tgz", chartRef, version)
	return nil
}

func (s *Svc) doGenerate(deploys cfg.Deploys) error {
	var err error
	s.tmp, err = s.appFs.TempDir("", "simple-ops-")
	if err != nil {
		return err
	}
	for _, deploy := range deploys {
		if err := s.generateDeploy(deploy); err != nil {
			return err
		}
	}

	return nil
}

func (s Svc) generateDeploy(deploy *cfg.Deploy) error {
	var chrt *chart.Chart
	var rel *release.Release
	var err error
	var t []byte
	var rendered bytes.Buffer

	s.log.Debugf("generating deploy %s:%s", deploy.Component, deploy.Environment)

	if deploy.Chart != "" {
		if chrt, err = s.loadChart(deploy); err != nil {
			return err
		}
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
		s.log.Debugf("created namespace manifest for %s", deploy.Id())
	}

	if deploy.Chart != "" {
		s.client.ReleaseName = chrt.Name()
		s.client.Namespace = deploy.Namespace.Name
		s.client.CreateNamespace = false

		// render the helm chart
		rel, err = s.client.Run(chrt, deploy.Values)
		if err != nil {
			return err
		}
		rendered.Write([]byte(rel.Manifest))
		s.log.Debugf("rendered chart %s.%s for %s", chrt.Name(), chrt.Metadata.Version, deploy.Id())
	}

	// with
	if deploy.With != nil {
		if err := s.with(deploy, &rendered); err != nil {
			return err
		}
	}

	// kustomize labels
	t, err = s.kustomizeLabels(deploy.Labels, rendered.Bytes())
	if err != nil {
		return err
	}

	// inject namespace
	if deploy.Namespace.Inject {
		if t, err = s.kustomizeNamespace(deploy, t); err != nil {
			return err
		}
		s.log.Debugf("injected namespace %s", deploy.Namespace.Name)
	}

	// write manifest
	path := s.pathForTmpManifest(deploy)
	if err := s.appFs.WriteFile(path, t, defaultFilePerm); err != nil {
		return err
	}

	// copy preserve paths
	if err := s.copyPreservePaths(deploy); err != nil {
		return err
	}

	// run kustomizations if any
	if err := s.kustomizeDeploy(deploy); err != nil {
		return err
	}
	s.log.Debugf("wrote manifest to %s", path)

	// run jsonnet
	b, err := s.Jsonnets(deploy)
	if err != nil {
		return err
	}
	if len(b) > 0 {

		// write jsonnet
		fh, err := s.appFs.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, defaultFilePerm)
		if err != nil {
			return err
		}
		finfo, err := fh.Stat()
		if err != nil {
			return err
		}
		if finfo.Size() > 0 {
			_, err = fh.Write([]byte("---\n"))
			if err != nil {
				return err
			}
		}
		_, err = fh.Write(b)
		if err != nil {
			return err
		}
	}
	return err
}

func (s Svc) with(deploy *cfg.Deploy, rendered io.Writer) error {
	var t []byte
	var err error

	// ordered with templates
	var orderedFiles []string
	for p := range deploy.With {
		orderedFiles = append(orderedFiles, p)
	}
	sort.Strings(orderedFiles)
	for _, p := range orderedFiles {
		withs, ok := deploy.With[p]
		if !ok {
			return fmt.Errorf("could not find with %s", p)
		}
		var ordered []string
		// iterate in-order such that the generated output
		// is idempotent
		for name := range withs {
			ordered = append(ordered, name)
		}
		sort.Strings(ordered)
		for _, name := range ordered {
			with, ok := withs[name]
			if !ok {
				return fmt.Errorf("could not find with %s", name)
			}
			if with.Path == "" {
				t, err = s.generateWith(p, with, name)
				if err != nil {
					return err
				}
				_, err = rendered.Write([]byte("---\n"))
				if err != nil {
					return err
				}
				_, err = rendered.Write([]byte(fmt.Sprintf("# Source: simple-ops with %s.yml\n", p)))
				if err != nil {
					return err
				}
				_, err = rendered.Write(t)
				if err != nil {
					return err
				}
				s.log.Debugf("generated with %s type %s for %s", name, p, deploy.Id())

			} else {
				if err := s.generateWithToPath(p, with, name); err != nil {
					return err
				}
				s.log.Debugf("generated with %s type %s for %s to path %s", name, p, deploy.Id(), with.Path)
			}
		}
	}
	return nil
}

// generateWith uses file named with/{n}.yml as a template rendered
// using with Values to a byte slice. With Path must be empty
func (s Svc) generateWith(n string, w cfg.With, name string) ([]byte, error) {
	if w.Path != "" {
		return nil, errors.New("unexpected path")
	}
	return s.renderWith(n, w, name)
}

// gnerateWithPath uses file name with/{n}.yml as a template rendered
// using with Values to the non-empty path specified relative to the
// working directory, e.g. apps/n.yaml
func (s Svc) generateWithToPath(n string, w cfg.With, name string) error {
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
	return s.appFs.WriteFile(path, b, defaultFilePerm)
}

// renderWith uses file at /with/n.yml
func (s Svc) renderWith(n string, w cfg.With, name string) ([]byte, error) {
	var c []byte
	var v map[string]interface{}
	var err error
	path := filepath.Join(s.wd, cfg.ResourcesPath, n) + cfg.Suffix
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
	v = cfg.MergeMaps(v, w.Values)
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

func (s Svc) loadChart(deploy *cfg.Deploy) (*chart.Chart, error) {
	var chrt *chart.Chart
	var err error

	// if using memfs under test use LoadArchive with archive file
	// The directory handling code in Helm cannot be persuaded to
	// use the fs abstraction. @todo better
	if _, ok := s.appFs.Fs.(*afero.MemMapFs); ok {
		f, err := s.appFs.Open(s.PathForChart(deploy.Chart))
		if err != nil {
			return nil, err
		}
		defer func() {
			err = f.Close()
		}()
		chrt, err = loader.LoadArchive(f)
	} else {
		chrt, err = loader.Load(s.PathForChart(deploy.Chart))
		if err != nil {
			return nil, err
		}
	}

	// @todo check chart dependencies
	if len(chrt.Dependencies()) != len(chrt.Metadata.Dependencies) {
		return nil, errors.New("dependencies not installed")
	}

	return chrt, err
}

// Namespace create without spec and status for tidier yaml
type Namespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

func (Svc) createNamespaceManifest(deploy *cfg.Deploy) ([]byte, error) {
	ns := Namespace{
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
	yml, err := yaml.Marshal(ns)
	if err != nil {
		return yml, err
	}
	// remove creationTimestamp: null
	return bytes.Replace(yml, []byte("creationTimestamp: null"), []byte(""), 1), nil
}

func (Svc) kustomizeNamespace(deploy *cfg.Deploy, manifest []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBuffer(manifest)}},
		Filters: []kio.Filter{namespace.Filter{Namespace: deploy.Namespace.Name, FsSlice: types.FsSlice{}}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
	}.Execute()
	return buf.Bytes(), err
}

func (Svc) kustomizeLabels(lbls map[string]string, manifest []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	fslice := types.FsSlice{
		{Path: "metadata/labels", CreateIfNotPresent: true},
		{Path: "spec/template/metadata/labels", CreateIfNotPresent: false},
	}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBuffer(manifest)}},
		Filters: []kio.Filter{labels.Filter{Labels: lbls, FsSlice: fslice}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
	}.Execute()
	return buf.Bytes(), err
}

func (s Svc) copyPreservePaths(d *cfg.Deploy) error {
	for _, p := range d.PreservePaths {
		dest := filepath.Join(s.tmp, p)
		src := filepath.Join(s.wd, p)
		if f, _ := s.appFs.Stat(dest); f != nil {
			continue
		}
		if err := cp.Copy(src, dest, cp.Options{AddPermission: defaultFilePerm, PreserveOwner: true}); err != nil {
			return err
		}
	}
	return nil
}

func (s Svc) kustomizeDeploy(d *cfg.Deploy) error {
	kfs := filesys.MakeFsOnDisk()
	opts := krusty.MakeDefaultOptions()
	krust := krusty.MakeKustomizer(opts)
	p := s.pathForTmpComponent(d)
	file := filepath.Join(s.tmp, "kustomization.yaml")
	manifest := filepath.Join(p, "manifest.yaml")
	for _, k := range d.Kustomizations {
		k.Resources = []string{
			manifest,
		}
		// write kustomization
		b, err := yaml.Marshal(k)
		if err != nil {
			return err
		}
		if err := s.appFs.WriteFile(file, b, defaultFilePerm); err != nil {
			return err
		}
		res, err := krust.Run(kfs, s.tmp)
		if err != nil {
			return err
		}
		b, err = res.AsYaml()
		if err != nil {
			return err
		}
		if err := s.appFs.WriteFile(manifest, b, defaultFilePerm); err != nil {
			return err
		}
		// delete kustomization file
		if err := s.appFs.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

func (s Svc) PathForChart(p string) string {
	return s.wd + string(os.PathSeparator) + cfg.ChartsPath + string(os.PathSeparator) + p
}

// os.Rename does not work if the rename crosses file systems
// afero does not move files with a directory change when using memfs
// so here we are
func (s Svc) renameDirectory(from string, to string) error {
	switch s.appFs.Fs.(type) {
	case *afero.OsFs:
		if err := os.RemoveAll(filepath.Join(to, cfg.DeployPath)); err != nil {
			return err
		}
		if err := s.appFs.MkdirAll(to, defaultDirPerm); err != nil {
			return err
		}
		defer func() { _ = os.RemoveAll(from) }()
		onSymLink := func(p string) cp.SymlinkAction {
			return cp.Skip
		}
		return cp.Copy(from, to, cp.Options{AddPermission: defaultFilePerm, PreserveOwner: true, OnSymlink: onSymLink})
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

func (s Svc) ManifestPathForDeploy(d *cfg.Deploy) string {
	return filepath.Join(s.wd, cfg.DeployPath, d.Environment, d.Component, "manifest.yaml")
}

// returns tmp path tmp/deploy/environment/component
func (s Svc) pathForTmpComponent(d *cfg.Deploy) string {
	return pathForTmpDeploy(d, s.tmp) + string(os.PathSeparator) + d.Component
}

// returns tmp path tmp/deploy/environment/
func pathForTmpDeploy(d *cfg.Deploy, tmpDir string) string {
	return tmpDir + string(os.PathSeparator) + cfg.DeployPath + string(os.PathSeparator) + d.Environment
}

// /tmp/dir/deploy/prod/component/manifest.yaml
func (s Svc) pathForTmpManifest(d *cfg.Deploy) string {
	return s.pathForTmpComponent(d) + string(os.PathSeparator) + "manifest.yaml"
}

func (s Svc) Jsonnets(d *cfg.Deploy) ([]byte, error) {
	var res []byte
	var prefix string
	for n, j := range d.Jsonnet {
		r, err := s.Jsonnet(n, j)
		if err != nil {
			return nil, err
		}
		if len(res) > 0 {
			prefix = "---\n"
		} else {
			prefix = ""
		}
		res = append(res, []byte(fmt.Sprintf("%s# Source: simple-ops jsonnet %s\n", prefix, n))...)
		res = append(res, r...)
	}
	return res, nil
}

func (s Svc) Jsonnet(n string, j *cfg.Jsonnet) ([]byte, error) {
	var b []byte
	var err error
	paths := []string{s.wd}
	vm := jsonnet.MakeVM()
	if j.Path != "" {
		paths = append(paths, filepath.Join(s.wd, filepath.Dir(j.Path), "vendor"))
	}
	vm.Importer(&jsonnet.FileImporter{JPaths: paths})
	for k, v := range j.Values {
		vm.ExtVar(k, v)
	}

	if j.Path != "" {
		b, err = s.JsonnetPath(j, vm)
		if err != nil {
			return nil, err
		}
	}

	if j.Inline != "" {
		e, err := s.JsonnetInline(j, vm, n)
		if err != nil {
			return nil, err
		}
		return append(b, e...), nil
	}

	return b, nil
}

func (Svc) JsonToYaml(b []byte) ([]byte, error) {
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return yaml.Marshal(&m)
}
func (s Svc) JsonnetInline(j *cfg.Jsonnet, vm *jsonnet.VM, n string) ([]byte, error) {
	var ss string
	var err error

	ss, err = vm.EvaluateAnonymousSnippet(n, j.Inline)
	if err != nil {
		return nil, err
	}
	return s.JsonToYaml([]byte(ss))
}
func (s Svc) JsonnetPath(j *cfg.Jsonnet, vm *jsonnet.VM) ([]byte, error) {
	var docs map[string]string
	var err error
	r := bytes.Buffer{}
	docs, err = vm.EvaluateFileMulti(j.Path)
	if err != nil {
		return nil, err
	}
	var ordered []string
	for d := range docs {
		ordered = append(ordered, d)
	}
	sort.Strings(ordered)

	for i, d := range ordered {
		if i != 0 {
			r.Write([]byte("---\n"))
			r.Write([]byte(fmt.Sprintf("# Simple-Ops Jsonnet %s\n", d)))
		}
		b, err := s.JsonToYaml([]byte(docs[d]))
		if err != nil {
			return nil, err
		}
		r.Write(b)
	}

	return r.Bytes(), nil
}
