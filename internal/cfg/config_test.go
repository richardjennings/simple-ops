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
	if err := c.InitIfEmpty(""); err != nil {
		t.Fatal(err)
	}
	c = NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	if err := afero.WriteFile(c.appFs, "/test/file", []byte{}, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	// should Not be able to init without force in Non empty directory
	actual := c.InitIfEmpty("")
	assert.Equal(t, actual.Error(), fmt.Errorf("path %s not empty", "/test").Error())
	// should be able to init with force in Non empty directory
	actual = c.Init("")
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
	if err = afero.WriteFile(c.appFs, filepath.Join(c.wd, GlobalConfigFile), []byte(""), DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	actual, err := c.Deploys()
	if err != nil {
		t.Fatal(err)
	}
	expected := Deploys{
		{
			Chart:       "b.tgz",
			Environment: "test",
			Component:   "b",
			Namespace:   Namespace{Name: "test", Create: true, Inject: false},
			Values:      map[string]interface{}{"more": false},
			With: map[string]map[string]With{
				"serviceEntry": {
					"github": {
						Path:   "test/",
						Values: map[string]interface{}{"spec": map[string]interface{}{"hosts": []interface{}{"github.com"}}},
					},
				},
			},
		},
		{
			Chart:       "b.tgz",
			Environment: "test2",
			Component:   "b",
			Namespace:   Namespace{Name: "test", Create: true, Inject: true},
			Values:      map[string]interface{}{"more": true},
			With: map[string]map[string]With{
				"serviceEntry": {
					"github": {
						Path:   "",
						Values: map[string]interface{}{"spec": map[string]interface{}{"hosts": []interface{}{"github.com"}}},
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
	expected := Deploys{&Deploy{Values: map[string]interface{}{"overridesTrue": "false"}, Component: "test", Environment: "test"}}
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
	expected := Deploys{&Deploy{Values: nil, Component: "test", Environment: "test"}}
	assert.DeepEqual(t, expected, actual)
}

func setupSetTest(t *testing.T, configFile string, configBytes []byte) *Svc {
	var err error
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	if err = c.appFs.Mkdir("/test/config", DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	// create a directory to be ignored
	if err = c.appFs.WriteFile(configFile, configBytes, DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSvc_SetNewConfigPathMap(t *testing.T) {
	c := setupSetTest(t, "/test/config/a.yml", []byte("chart: a.yml\ndeploy:\n  example:\n    values:\n"))
	if err := c.Set("a.deploy.example.values.foo", "bar"); err != nil {
		t.Error(err)
	}
	expected := "chart: a.yml\ndeploy:\n  example:\n    values:\n      foo: bar\n"
	actual, err := c.appFs.ReadFile("/test/config/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(actual))
}

func TestSvc_SetExistingConfigPathMap(t *testing.T) {
	conf := []byte("chart: a.yml\ndeploy:\n  example:\n    values:\n      foo: foo")
	c := setupSetTest(t, "/test/config/a.yml", conf)
	if err := c.Set("a.deploy.example.values.foo", "bar"); err != nil {
		t.Error(err)
	}
	expected := "chart: a.yml\ndeploy:\n  example:\n    values:\n      foo: bar\n"
	actual, err := c.appFs.ReadFile("/test/config/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(actual))
}

func TestSvc_SetNewListString(t *testing.T) {
	conf := []byte("chart: a.yml\ndeploy:\n  example:\n    values:\n")
	c := setupSetTest(t, "/test/config/a.yml", conf)
	if err := c.Set("a.deploy.example.values.foo.0", "bar"); err != nil {
		t.Error(err)
	}
	expected := "chart: a.yml\ndeploy:\n  example:\n    values:\n      foo:\n      - bar\n"
	actual, err := c.appFs.ReadFile("/test/config/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(actual))
}

func TestSvc_SetNewListMapString(t *testing.T) {
	conf := []byte("chart: a.yml\ndeploy:\n  example:\n    values:\n")
	c := setupSetTest(t, "/test/config/a.yml", conf)
	if err := c.Set("a.deploy.example.values.foo.0.foo", "bar"); err != nil {
		t.Error(err)
	}
	expected := "chart: a.yml\ndeploy:\n  example:\n    values:\n      foo:\n      - foo: bar\n"
	actual, err := c.appFs.ReadFile("/test/config/a.yml")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, string(actual))
}

func TestSvc_GetDeploy_None(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	if err := c.appFs.Mkdir("/test/config", DefaultConfigFsPerm); err != nil {
		t.Error(err)
	}
	if err := afero.WriteFile(c.appFs, "/test/simple-ops.yml", []byte(""), DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	d, err := c.GetDeploy("a", "b")
	assert.Assert(t, d == nil)
	assert.ErrorContains(t, err, "deploy b.a not found")
}

func TestSvc_GetDeploy_Exists(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	if err := c.appFs.Mkdir("/test/config", DefaultConfigFsPerm); err != nil {
		t.Error(err)
	}
	if err := afero.WriteFile(c.appFs, "/test/simple-ops.yml", []byte(""), DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(c.appFs, "/test/config/a.yml", []byte("deploy:\n  b:\n    enabled: false\n"), DefaultConfigFsPerm); err != nil {
		t.Fatal(err)
	}
	d, err := c.GetDeploy("a", "b")
	assert.Assert(t, d != nil)
	assert.NilError(t, err)
	assert.Equal(t, d.Environment, "b")
	assert.Equal(t, d.Component, "a")
}

func TestSvc_ManifestPath(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	d := Deploy{Environment: "a", Component: "b"}
	s, err := c.ManifestPath(d)
	assert.NilError(t, err)
	assert.Equal(t, s, "/test/deploy/a/b/manifest.yaml")
}

func TestSvc_ChartPath(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), "/test", logrus.New())
	d := Deploy{Chart: "a.b.tgz"}
	s, err := c.ChartPath(d)
	assert.NilError(t, err)
	assert.Equal(t, s, "/test/charts/a.b.tgz")
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

func Test_DeployIdParts(t *testing.T) {
	e, c, err := DeployIdParts("a.b")
	assert.NilError(t, err)
	assert.Equal(t, e, "a")
	assert.Equal(t, c, "b")
}
func Test_DeployIdParts_Invalid(t *testing.T) {
	e, c, err := DeployIdParts("a.b.c")
	assert.ErrorContains(t, err, "invalid a.b.c")
	assert.Equal(t, e, "")
	assert.Equal(t, c, "")
}

func Test_Deploy_Id(t *testing.T) {
	d := Deploy{Environment: "a", Component: "b"}
	assert.Equal(t, d.Id(), "a.b")
}

func Test_MergeMaps(t *testing.T) {
	for _, tc := range []struct {
		a map[string]interface{}
		b map[string]interface{}
		e map[string]interface{}
	}{
		{map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}},
		{ // lists are replaced
			map[string]interface{}{"a": []string{"1,2,4"}},
			map[string]interface{}{"a": []string{"1,2,3"}},
			map[string]interface{}{"a": []string{"1,2,3"}},
		},
		{
			// bool a.b can be set from true to false
			a: map[string]interface{}{"a": map[string]interface{}{"b": true}},
			b: map[string]interface{}{"a": map[string]interface{}{"b": false}},
			e: map[string]interface{}{"a": map[string]interface{}{"b": false}},
		},
		{
			// bool a.b can be set from false to true
			a: map[string]interface{}{"a": map[string]interface{}{"b": false}},
			b: map[string]interface{}{"a": map[string]interface{}{"b": true}},
			e: map[string]interface{}{"a": map[string]interface{}{"b": true}},
		},
		{
			// can add key to map value
			a: map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": "test"}}},
			b: map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"d": "test"}}},
			e: map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": "test", "d": "test"}}},
		},
	} {
		assert.DeepEqual(t, MergeMaps(tc.a, tc.b), tc.e)
	}
}
