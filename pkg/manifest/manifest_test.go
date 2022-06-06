package manifest

import (
	"github.com/richardjennings/simple-ops/pkg/config"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

func TestSvc_GenerateVerify(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := fs.MkdirAll("/test", config.DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	m := NewSvc(fs, "/test")
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
	withData := "metadata:\n"
	if err := afero.WriteFile(fs, "/test/with/file.yml", []byte(withData), 0655); err != nil {
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
					"path": config.With{
						Path: "file.yaml",
					},
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
  name: name
  namespace: test
spec:
  a: b
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
