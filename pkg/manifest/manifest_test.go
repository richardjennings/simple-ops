package manifest

import (
	"github.com/richardjennings/simple-ops/pkg/config"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func setupWithTestChart(t *testing.T, fs afero.Fs) {
	if err := fs.MkdirAll("/test", config.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	// set up directories
	if err := fs.MkdirAll("/test/config", config.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/deploy", config.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/charts", config.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := fs.MkdirAll("/test/with", config.DefaultConfigDirPerm); err != nil {
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
	m := NewSvc(fs, "/test")

	setupWithTestChart(t, fs)

	withData := "metadata:\n"
	if err := afero.WriteFile(fs, "/test/with/file.yml", []byte(withData), 0755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "/test/with/thing.yml", []byte(withData), 0755); err != nil {
		t.Fatal(err)
	}
	deploys := config.Deploys{
		"env": &config.Deploy{
			Chart: "test-0.1.0.tgz",
			Namespace: config.Namespace{
				Name:   "test",
				Inject: true,
				Create: true,
			},
			Values: map[string]interface{}{
				"test": "true",
			},
			With: config.Withs{
				"file": {
					"name": config.With{
						Values: map[string]interface{}{
							"spec": map[string]interface{}{
								"a": "b",
							},
						},
					},
					"before": config.With{
						Values: map[string]interface{}{},
					},
					"path": config.With{
						Path: "file.yaml",
					},
				},
				"thing": {
					"aa": config.With{},
					"a":  config.With{},
				},
			},
			Name:      "env",
			Component: "test",
		},
	}
	if err := m.Generate(map[string]config.Deploys{"env": deploys}); err != nil {
		t.Fatal(err)
	}
	manifest, err := afero.ReadFile(fs, "/test/deploy/env/test/manifest.yaml")
	if err != nil {
		t.Fatal(err)
	}
	expect := `apiVersion: v1
kind: Namespace
metadata:
  creationTimestamp: null
  labels:
    name: test
  name: test
spec: {}
status: {}
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

	valid, err := m.Verify(map[string]config.Deploys{"env": deploys})
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

func TestSvc_generateDeploy(t *testing.T) {
	fs := afero.NewMemMapFs()
	m := NewSvc(fs, "/test")
	m.tmp = "/test"

	setupWithTestChart(t, fs)
	deploy := &config.Deploy{
		Chart:     "test-0.1.0.tgz",
		Name:      "env",
		Component: "test",
	}
	err := m.generateDeploy(deploy)
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
