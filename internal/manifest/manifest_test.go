package manifest

import (
	"github.com/google/go-jsonnet"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func setupWithTestChart(t *testing.T, fs afero.Fs) {
	if err := fs.MkdirAll("/test", cfg.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	// set up directories
	if err := fs.MkdirAll("/test/config", cfg.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/deploy", cfg.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/charts", cfg.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/resources", cfg.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	// use tesdata chart
	chrt, err := ioutil.ReadFile("./testdata/test-0.1.0.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "/test/charts/test-0.1.0.tgz", chrt, 0655); err != nil {
		t.Fatal(err)
	}
}

func TestSvc_GenerateVerify(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())

	setupWithTestChart(t, fs)

	withData := "metadata:\n"
	if err := afero.WriteFile(fs, "/test/resources/file.yml", []byte(withData), 0755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "/test/resources/thing.yml", []byte(withData), 0755); err != nil {
		t.Fatal(err)
	}
	deploys := cfg.Deploys{
		&cfg.Deploy{
			Chart: "test-0.1.0.tgz",
			Namespace: cfg.Namespace{
				Name:   "test",
				Inject: true,
				Create: true,
			},
			Values: map[string]interface{}{
				"test": "true",
			},
			With: cfg.Withs{
				"file": {
					"name": cfg.With{
						Values: map[string]interface{}{
							"spec": map[string]interface{}{
								"a": "b",
							},
						},
					},
					"before": cfg.With{
						Values: map[string]interface{}{},
					},
					"path": cfg.With{
						Path: "file.yaml",
					},
				},
				"thing": {
					"aa": cfg.With{},
					"a":  cfg.With{},
				},
			},
			Environment: "env",
			Component:   "test",
			Chain:       []string{"helm", "with", "namespace"},
		},
	}
	if err := m.Generate(deploys); err != nil {
		t.Fatal(err)
	}
	manifest, err := afero.ReadFile(fs, "/test/deploy/env/test/manifest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	expect := `apiVersion: v1
kind: Namespace
metadata:
  labels:
    name: test
  name: test
---
# Source: test/templates/test.yaml
test: true
metadata:
  namespace: test
---
# Source: simple-ops with file.yml
metadata:
  name: before
  namespace: test
---
# Source: simple-ops with file.yml
metadata:
  name: name
  namespace: test
spec:
  a: b
---
# Source: simple-ops with thing.yml
metadata:
  name: a
  namespace: test
---
# Source: simple-ops with thing.yml
metadata:
  name: aa
  namespace: test
`
	assert.Equal(t, string(manifest), expect)

	valid, err := m.Verify(deploys)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, valid, true)

	// check manifest generated via with path
	withPath, err := afero.ReadFile(fs, "/test/file.yaml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "metadata:\n  name: path\n", string(withPath))
}

func TestSvc_chainDeploy(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	m.tmp = "/test"

	setupWithTestChart(t, fs)
	deploy := &cfg.Deploy{
		Chart:       "test-0.1.0.tgz",
		Environment: "env",
		Component:   "test",
		Chain:       []string{"helm"},
	}
	err := m.chainDeploy(deploy)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := afero.ReadFile(fs, filepath.Join(m.tmp, "deploy/env/test/manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	expected := `---
# Source: test/templates/test.yaml
test:
`
	assert.Equal(t, string(actual), expected)
}

func TestSvc_ManifestPathForDeploy(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	deploy := cfg.Deploy{Environment: "testenv", Component: "app"}
	actual := m.ManifestPathForDeploy(&deploy)
	expected := "/test/deploy/testenv/app/manifest.yaml"
	assert.Equal(t, expected, actual)
}

func TestSvc_Pull_Invalid(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	err := m.Pull("a", "b", "c")
	assert.ErrorContains(t, err, "could not find protocol handler for: ")
}

func TestSvc_pull(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	p, err := m.doPull("b", "c")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, p.DestDir, "/test/charts")
	assert.Equal(t, p.Untar, false)
	assert.Equal(t, p.RepoURL, "b")
	assert.Equal(t, p.Version, "c")
}

func TestSvc_pullAddConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	if err := m.PullAddConfig("a", "b"); err != nil {
		t.Error(err)
	}
	b, err := afero.ReadFile(fs, "/test/config/a.yml")
	assert.NilError(t, err)
	assert.Equal(t, string(b), `chart: a-b.tgz`)
}

func TestSvc_JsonnetDeploy_inline(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())
	deploy := cfg.Deploy{
		Environment: "test",
		Component:   "test",
		Jsonnet: map[string]*cfg.Jsonnet{
			"test": {
				Inline: `
local a = 1;
[a]
`,
			},
		},
	}
	err := m.jsonnetDeploy(&deploy, nil)
	assert.NilError(t, err)
	b, err := afero.ReadFile(fs, "/deploy/test/test/manifest.yaml")
	assert.NilError(t, err)
	expected := `# Source: simple-ops jsonnet test
- 1
`
	assert.Equal(t, string(b), expected)
}

func TestSvc_JsonnetDeploy_path(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())

	testjsonnet := `
local a = 1;
[a]
`
	deploy := cfg.Deploy{
		Environment: "test",
		Component:   "test",
		Jsonnet: map[string]*cfg.Jsonnet{
			"test": {
				Path: "jsonnet/test.jsonnet",
			},
		},
	}
	imp := jsonnet.MemoryImporter{}
	c := jsonnet.MakeContents(testjsonnet)
	// path is not absolute for testing
	imp.Data = map[string]jsonnet.Contents{"jsonnet/test.jsonnet": c}
	err := m.jsonnetDeploy(&deploy, &imp)
	assert.NilError(t, err)
	b, err := afero.ReadFile(fs, "/deploy/test/test/manifest.yaml")
	assert.NilError(t, err)
	expected := `# Source: simple-ops jsonnet test
- 1
`
	assert.Equal(t, string(b), expected)
}

func TestSvc_JsonnetDeploy_pathMulti(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test", logrus.New())

	testjsonnet := `
local a = 1;
local b = 2;
{"a": [a]}
{"b": [b]}
`
	deploy := cfg.Deploy{
		Environment: "test",
		Component:   "test",
		Jsonnet: map[string]*cfg.Jsonnet{
			"test": {
				PathMulti: "jsonnet/test.jsonnet",
			},
		},
	}
	imp := jsonnet.MemoryImporter{}
	c := jsonnet.MakeContents(testjsonnet)
	// path is not absolute for testing
	imp.Data = map[string]jsonnet.Contents{"jsonnet/test.jsonnet": c}
	err := m.jsonnetDeploy(&deploy, &imp)
	assert.NilError(t, err)
	b, err := afero.ReadFile(fs, "/deploy/test/test/manifest.yaml")
	assert.NilError(t, err)
	expected := `# Source: simple-ops jsonnet test
- 1
---
# Simple-Ops jsonnet b
- 2
`
	assert.Equal(t, string(b), expected)
}
