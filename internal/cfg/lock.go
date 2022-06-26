package cfg

import (
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"path/filepath"
)

type (
	Lock struct {
		appFs afero.Afero
		wd    string
		log   *logrus.Logger
	}
	LockFile struct {
		Charts []*ChartSource `json:"charts"`
	}
	ChartSource struct {
		Name       string `json:"charts"`
		Repository string `json:"repository"`
		Version    string `json:"version"`
		Digest     string `json:"digest"`
	}
)

func NewLock(fs afero.Fs, wd string, log *logrus.Logger) *Lock {
	return &Lock{appFs: afero.Afero{Fs: fs}, wd: wd, log: log}
}

func (l *Lock) AddChart(name string, repository string, version string) error {
	lf, err := l.readLockFile()
	if err != nil {
		return err
	}
	for _, v := range lf.Charts {
		if v.Name == name {
			if v.Repository == repository && v.Version == version {
				// nothing to do ... ?
				// check digest >?
				// @todo
				return nil
			}
		}
	}
	lf.Charts = append(lf.Charts, &ChartSource{
		Name:       name,
		Repository: repository,
		Version:    version,
		Digest:     "",
	})
	return l.writeLockFile(lf)
}

func (l *Lock) readLockFile() (*LockFile, error) {
	b, err := l.appFs.ReadFile(filepath.Join(l.wd, LockFileName))
	if err != nil {
		return nil, err
	}
	lf := &LockFile{}
	if err := yaml.Unmarshal(b, lf); err != nil {
		return nil, err
	}
	return lf, nil
}

func (l *Lock) writeLockFile(lockFile *LockFile) error {
	b, err := yaml.Marshal(lockFile)
	if err != nil {
		return err
	}
	return l.appFs.WriteFile(filepath.Join(l.wd, LockFileName), b, DefaultConfigFsPerm)
}
