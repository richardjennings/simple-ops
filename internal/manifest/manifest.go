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
	"helm.sh/helm/v3/pkg/cli"
	"io/fs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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
		if err := s.chainDeploy(deploy); err != nil {
			return err
		}
	}

	return nil
}

func (s Svc) chainDeploy(deploy *cfg.Deploy) error {
	var manifest bytes.Buffer
	var crd bytes.Buffer
	var err error

	if err = s.appFs.MkdirAll(s.pathForTmpComponent(deploy), defaultDirPerm); err != nil {
		return err
	}

	// create namespace manifest
	if deploy.Namespace.Create {
		t, err := s.createNamespaceManifest(deploy)
		if err != nil {
			return err
		}
		manifest.Write(t)
		s.log.Debugf("created namespace manifest for %s", deploy.Id())
	}

	// run through chain of actions
	for _, c := range deploy.Chain {
		fn, ok := actions[c]
		if !ok {
			return fmt.Errorf("action %s not found", c)
		}
		err = fn(deploy, &manifest, &crd, s)
		if err != nil {
			return err
		}
	}

	// write tmp
	if manifest.Len() > 0 {
		if err := s.writeTmp(deploy, &manifest); err != nil {
			return err
		}
	}

	// write CRDs
	if crd.Len() > 0 {
		if err := s.appFs.WriteFile(s.pathForTmpCRDs(deploy), crd.Bytes(), defaultFilePerm); err != nil {
			return err
		}
	}

	return nil
}

func (s Svc) writeTmp(deploy *cfg.Deploy, man *bytes.Buffer) error {
	// write manifest
	path := s.pathForTmpManifest(deploy)
	return s.appFs.WriteFile(path, man.Bytes(), defaultFilePerm)
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

func (s Svc) copyKustomizationPaths(d *cfg.Deploy) error {
	for _, p := range d.KustomizationPaths {
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

// /tmp/dir/deploy/prod/component/crds.yaml
func (s Svc) pathForTmpCRDs(d *cfg.Deploy) string {
	return s.pathForTmpComponent(d) + string(os.PathSeparator) + "crds.yaml"
}

func (s Svc) jsonnetDeploy(d *cfg.Deploy, imp jsonnet.Importer) error {
	path := s.pathForTmpManifest(d)

	// run jsonnet
	b, err := s.jsonnets(d, imp)
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

func (s Svc) jsonnets(d *cfg.Deploy, imp jsonnet.Importer) ([]byte, error) {
	var res []byte
	var prefix string
	for n, j := range d.Jsonnet {
		r, err := s.jsonnet(n, j, imp)
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

func (s Svc) jsonnet(n string, j *cfg.Jsonnet, imp jsonnet.Importer) ([]byte, error) {
	var b []byte
	var err error
	paths := []string{s.wd}
	vm := jsonnet.MakeVM()
	// allow injecting mem importer from test
	if imp != nil {
		vm.Importer(imp)
	} else {
		if j.Path != "" {
			paths = append(paths, filepath.Join(s.wd, filepath.Dir(j.Path), "vendor"))
		}
		if j.PathMulti != "" {
			paths = append(paths, filepath.Join(s.wd, filepath.Dir(j.PathMulti), "vendor"))
		}
		vm.Importer(&jsonnet.FileImporter{JPaths: paths})
		for k, v := range j.Values {
			vm.ExtVar(k, v)
		}
	}

	if j.PathMulti != "" {
		if b, err = s.JsonnetPathMulti(j, vm); err != nil {
			return nil, err
		}
	}

	if j.Path != "" {
		e, err := s.JsonnetPath(j, vm)
		if err != nil {
			return nil, err
		}
		b = append(b, e...)
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
	var m interface{}
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
	str, err := vm.EvaluateFile(j.Path)
	if err != nil {
		return nil, err
	}
	return s.JsonToYaml([]byte(str))
}

func (s Svc) JsonnetPathMulti(j *cfg.Jsonnet, vm *jsonnet.VM) ([]byte, error) {

	var docs map[string]string
	var err error
	r := bytes.Buffer{}
	docs, err = vm.EvaluateFileMulti(j.PathMulti)
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
			r.Write([]byte(fmt.Sprintf("# Simple-Ops jsonnet %s\n", d)))
		}
		b, err := s.JsonToYaml([]byte(docs[d]))
		if err != nil {
			return nil, err
		}
		r.Write(b)
	}

	return r.Bytes(), nil
}
