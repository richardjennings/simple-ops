package cfg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSvc_Init(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	// should be able to init without force in empty directory
	if err := c.Init(false); err != nil {
		t.Fatal(err)
	}
	c = NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	if err := afero.WriteFile(c.appFs, "/test/file", []byte{}, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	// should Not be able to init without force in Non empty directory
	actual := c.Init(false)
	assert.Equal(t, actual.Error(), fmt.Errorf("path %s not empty", "/test").Error())
	// should be able to init with force in Non empty directory
	actual = c.Init(true)
	assert.NilError(t, actual)
}

func TestSvc_Deploys(t *testing.T) {
	var err error
	var p = ConfPath
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	var cfgPath = filepath.Join(c.wd, p, "b"+Suffix)
	yml := []byte(`
chart: b.tgz
namespace:
  name: test
  create: true
  inject: true
with:
  serviceEntry:
    github:
      values:
        spec:
          hosts:
          - github.com

values:
  more: true
deploy:
  test:
    with:
      serviceEntry:
        github:
          path: test/
    values:
      more: false
    namespace:
      inject: false
  test2:
    with:
      serviceEntry:
        github:
`)
	if err = afero.WriteFile(c.appFs, cfgPath, yml, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	actual, err := c.Deploys()
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]Deploys{
		"b": {
			"test": {
				Chart:     "b.tgz",
				Name:      "test",
				Component: "b",
				Namespace: Namespace{Name: "test", Create: true, Inject: false},
				Values:    map[string]interface{}{"more": false},
				With: map[string]map[string]With{
					"serviceEntry": {
						"github": {
							Path:   "test/",
							Values: map[string]interface{}{"spec": map[string]interface{}{"hosts": []interface{}{"github.com"}}},
						},
					},
				},
			},
			"test2": {
				Chart:     "b.tgz",
				Name:      "test2",
				Component: "b",
				Namespace: Namespace{Name: "test", Create: true, Inject: true},
				Values:    map[string]interface{}{"more": true},
				With: map[string]map[string]With{
					"serviceEntry": {
						"github": {
							Path:   "",
							Values: map[string]interface{}{"spec": map[string]interface{}{"hosts": []interface{}{"github.com"}}},
						},
					},
				},
			},
		},
	}
	assert.DeepEqual(
		t,
		expected,
		actual,
	)
}

func TestSvc_getConfigPaths(t *testing.T) {
	var err error
	var s = string(os.PathSeparator)
	var p = ConfPath
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())

	if err = c.appFs.Mkdir(p, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	// create a directory to be ignored
	if err = c.appFs.Mkdir(p+s+"a", DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	if err = afero.WriteFile(c.appFs, filepath.Join(c.wd, p, "b")+Suffix, []byte{}, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	if err = afero.WriteFile(c.appFs, filepath.Join(c.wd, p, "c.yaml"), []byte{}, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	paths, err := c.getConfigPaths()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, reflect.DeepEqual(map[string]string{"b": "config/b.yml"}, paths), true)
}

func TestSvc_parseConfig(t *testing.T) {
	for _, tc := range []struct {
		path    string
		content string
		err     string
		expect  map[string]interface{}
	}{
		// invalid content
		{"a.yml", "\t", "error converting YAML to JSON: yaml: found character that cannot start any token", nil},
		// valid content
		{"b.yaml", "a:\n  b: 1", "", map[string]interface{}{"a": map[string]interface{}{"b": float64(1)}}},
	} {
		c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
		if err := c.appFs.Mkdir(ConfPath, DefaultConfigFsPerm); err != nil {
			t.Fatal(err)
		}
		if err := afero.WriteFile(c.appFs, filepath.Join(c.wd, tc.path), []byte(tc.content), DefaultConfigFsPerm); err != nil {
			t.Fatal(err)
		}
		actual, err := c.parseConfig(tc.path)
		if tc.err != "" {
			if err == nil {
				t.Fatal("expected error")
			}
			assert.Equal(t, err.Error(), tc.err)
		}
		if tc.expect != nil {
			assert.DeepEqual(t, tc.expect, actual)
		}
	}
}

func TestSvc_buildDeploys(t *testing.T) {
	// a. test to check boolean false can override boolean true
	// this does not work with mergo library or helm 2
	m := map[string]interface{}{
		"values": map[string]interface{}{
			"overridesTrue": "true",
		},
		"deploy": map[string]interface{}{
			"test": map[string]interface{}{
				"values": map[string]interface{}{
					"overridesTrue": "false",
				},
			},
		},
	}
	component := "test"
	actual, err := buildDeploys(m, component)
	if err != nil {
		t.Error(err)
	}
	expected := Deploys{"test": &Deploy{Values: map[string]interface{}{"overridesTrue": "false"}, Component: "test", Name: "test"}}
	assert.DeepEqual(t, expected, actual)
}

func TestSvc_buildDeploys_without_values(t *testing.T) {
	// a. test to check boolean false can override boolean true
	// this does not work with mergo library or helm 2
	m := map[string]interface{}{
		"deploy": map[string]interface{}{
			"test": map[string]interface{}{},
		},
	}
	component := "test"
	actual, err := buildDeploys(m, component)
	if err != nil {
		t.Error(err)
	}
	expected := Deploys{"test": &Deploy{Values: nil, Component: "test", Name: "test"}}
	assert.DeepEqual(t, expected, actual)
}

func TestNewSvc(t *testing.T) {
	fs := afero.NewMemMapFs()
	c := NewSvc(fs, "/test", logrus.New())
	if err := fs.MkdirAll("/test/config/", DefaultConfigDirPerm); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, "/test/config/myapp.yml", []byte{}, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("myapp.deploy.test.imgSrc", "abc"); err != nil {
		t.Fatal(err)
	}
	conf, err := afero.ReadFile(fs, "/test/config/myapp.yml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, string(conf), "deploy:\n  test:\n    imgSrc: abc\n")
}

func Test_componentName(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"a.yml", args{p: "a.yml"}, "a"},
		{"/path/test.yml", args{p: "/path/test.yml"}, "test"},
		{".yml", args{p: ".yml"}, ""},
		{".yaml", args{p: ".yaml"}, ".yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := componentName(tt.args.p); got != tt.want {
				t.Errorf("componentName() = %v, want %v", got, tt.want)
			}
		})
	}
}
