package cfg

import (
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ConfPath             = "config"
	DeployPath           = "deploy"
	ChartsPath           = "charts"
	WithPath             = "with"
	DefaultConfigFsPerm  = 0655
	DefaultConfigDirPerm = 0755
	Suffix               = ".yml"
)

type (
	Svc struct {
		appFs afero.Afero
		wd    string
		log   *logrus.Logger
	}
	Deploy struct {
		Namespace Namespace              `json:"namespace"`
		Chart     string                 `json:"chart"`
		Disabled  bool                   `json:"enabled"`
		With      Withs                  `json:"with"`
		Values    map[string]interface{} `json:"values"`
		Name      string                 `json:"-"`
		Component string                 `json:"-"`
	}
	Conf struct {
		Deploy
		Deploys Deploys `json:"deploy"`
	}
	Withs map[string]map[string]With
	With  struct {
		Path   string                 `json:"path"`
		Values map[string]interface{} `json:"values"`
	}
	// Deploys is a container for different Deployments of a component
	Deploys   map[string]*Deploy
	Namespace struct {
		Name   string `json:"name"`
		Create bool   `json:"create"`
		Inject bool   `json:"inject"`
		Labels Labels `json:"Labels"`
	}
	Labels map[string]string
)

func (d Deploy) RelativeManifestPath() string {
	return filepath.Join(DeployPath, d.Name, d.Component, "manifest.yaml")
}

func (d Deploy) RelativeChartPath() string {
	return filepath.Join(ChartsPath, d.Chart)
}

func NewSvc(fs afero.Fs, wd string, log *logrus.Logger) *Svc {
	return &Svc{appFs: afero.Afero{Fs: fs}, wd: wd, log: log}
}

// Deploys returns configured deploys for a given config path
func (s Svc) Deploys() (map[string]Deploys, error) {
	deploys := make(map[string]Deploys)
	paths, err := s.getConfigPaths()
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		m, err := s.parseConfig(path)
		if err != nil {
			return nil, err
		}
		component := componentName(path)
		d, err := buildDeploys(m, component)
		if err != nil {
			return nil, err
		}
		deploys[component] = make(Deploys)
		for k, v := range d {
			if _, ok := deploys[component][k]; ok {
				return nil, errors.New("duplicate deploy name found somehow")
			}
			deploys[component][k] = v
			s.log.Debugf("found component: %s for env %s\n", component, k)
		}
	}

	return deploys, nil
}

// Init creates simple-ops directory and structure in path if
// directory not exists or empty.
// force generates directory structure when path not empty
func (s Svc) Init(force bool) error {
	path := s.wd
	f, err := s.appFs.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(f) > 0 && force == false {
		return fmt.Errorf("path %s not empty", path)
	}
	prefix := path + string(os.PathSeparator)
	if err := s.appFs.MkdirAll(prefix+ConfPath, DefaultConfigDirPerm); err != nil {
		return err
	}
	if err := s.appFs.MkdirAll(prefix+DeployPath, DefaultConfigDirPerm); err != nil {
		return err
	}
	if err := s.appFs.MkdirAll(prefix+ChartsPath, DefaultConfigDirPerm); err != nil {
		return err
	}
	if err := s.appFs.MkdirAll(prefix+WithPath, DefaultConfigDirPerm); err != nil {
		return err
	}
	if err := s.appFs.WriteFile(prefix+"simple-ops.yml", []byte{}, DefaultConfigFsPerm); err != nil {
		return err
	}
	return nil
}

// Set adds or modifies a configuration path value.
// The first part of the path specifies the config file, e.g.
// myapp.deploys.staging.imgSrc would target config/myapp.yml
// and would add or modify the imgSrc value in deploys: staging: imgSrc
func (s Svc) Set(path string, value string) error {
	var b []byte
	var err error
	var conf map[string]interface{}

	parts := strings.Split(path, ".")

	configFile := filepath.Join(s.wd, ConfPath, parts[0]) + Suffix

	b, err = s.appFs.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.Unmarshal(b, &conf); err != nil {
		return err
	}

	if conf == nil {
		conf = make(map[string]interface{})
	}

	if err = set(conf, parts[1:], value); err != nil {
		return err
	}
	c, err := yaml.Marshal(conf)

	return s.appFs.WriteFile(configFile, c, DefaultConfigFsPerm)
}

// getConfigPaths produces a list of config files found
func (s Svc) getConfigPaths() (map[string]string, error) {
	var f afero.File
	var files []os.FileInfo
	var err error
	var name string
	path := ConfPath

	configFiles := make(map[string]string)
	if f, err = s.appFs.Open(filepath.Join(s.wd, path)); err != nil {
		return nil, err
	}
	defer func() {
		err = f.Close()
	}()
	if files, err = f.Readdir(0); err != nil {
		return nil, err
	}
	for _, v := range files {
		if v.IsDir() {
			continue
		}
		if !strings.HasSuffix(v.Name(), Suffix) {
			continue
		}
		name = strings.TrimSuffix(v.Name(), Suffix)
		configFiles[name] = path + string(os.PathSeparator) + v.Name()
	}
	return configFiles, nil
}

// parseConfig parses a config file into a map[string]interface{} to aid
// merging configuration
func (s Svc) parseConfig(path string) (map[string]interface{}, error) {
	var data []byte
	var err error
	var c map[string]interface{}

	if data, err = s.appFs.ReadFile(filepath.Join(s.wd, path)); err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	return c, nil
}

// MergeMaps makes a copy of the first map, overrides the values in the copy
// that exist in the 2nd map and returns the result. If a value in the 2nd map
// is nil, the values from the 1st map are used.
// copied and modified from helm 3 // ok || v == nil
func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {

		if v, ok := v.(map[string]interface{}); ok || v == nil {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// buildDeploys merges parent config into Deploy config.
func buildDeploys(m map[string]interface{}, component string) (Deploys, error) {
	ds := make(map[string]map[string]interface{})

	// parent deploy config
	if _, ok := m["deploy"].(map[string]interface{}); ok {
		for k, v := range m["deploy"].(map[string]interface{}) {
			if v != nil {
				ds[k] = v.(map[string]interface{})
			}
		}
	}

	// merge
	for k, _ := range ds {
		ds[k] = MergeMaps(m, ds[k])
	}

	// marshal back to yaml
	yml, err := yaml.Marshal(ds)
	if err != nil {
		return nil, err
	}

	// and then as Deploys
	var deploys Deploys
	if err := yaml.Unmarshal(yml, &deploys); err != nil {
		return nil, err
	}

	// update name and component
	for name, deploy := range deploys {
		deploy.Name = name
		deploy.Component = component
	}

	return deploys, nil
}

func componentName(p string) string {
	parts := strings.Split(p, string(os.PathSeparator))
	return strings.TrimSuffix(parts[len(parts)-1], Suffix)
}

func set(m interface{}, path []string, v string) error {
	l := len(path)
	if l == 0 {
		return errors.New("0 length path")
	}
	p := path[0]

	switch m := m.(type) {
	case map[string]interface{}:
		if l == 1 {
			m[p] = v
			return nil
		}
		if vv, ok := m[p]; !ok || vv == nil {
			m[p] = setType(path[1])
		}
		if err := set(m[p], path[1:], v); err != nil {
			return err
		}
	case []interface{}:
		i, err := strconv.Atoi(p)
		if err != nil {
			return err
		}
		if l == 1 {
			m[i] = v
			return nil
		} else {
			m[i] = setType(path[1])
		}
		return set(m[i], path[1:], v)
	default:
		return errors.New("unhandled type")
	}
	return nil
}

func setType(p string) interface{} {
	if pp, err := strconv.Atoi(p); err == nil {
		// should be a list
		if pp < 0 {
			return errors.New("index less than 0")
		}
		return make([]interface{}, pp+1)
	} else {
		// or map
		return make(map[string]interface{})
	}
}
