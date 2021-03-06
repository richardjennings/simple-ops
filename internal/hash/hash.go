package hash

import (
	"crypto/sha256"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"io"
	"os"
)

type (
	Svc struct {
		AppFs afero.Afero
		log   *logrus.Logger
	}
)

func NewSvc(fs afero.Fs, log *logrus.Logger) *Svc {
	return &Svc{AppFs: afero.Afero{Fs: fs}, log: log}
}

func (s Svc) SHA256File(path string) (string, error) {
	hash := sha256.New()
	var f afero.File
	var err error
	if f, err = s.AppFs.Fs.Open(path); err != nil {
		return "", err
	}
	if _, err := io.Copy(hash, f); err != nil {
		if err2 := f.Close(); err2 != nil {
			return "", fmt.Errorf("%s & %s", err, err2)
		}
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SHA256 computes the hash of files in path directory
func (s Svc) SHA256(path string) (string, error) {
	hash := sha256.New()
	if err := s.AppFs.Walk(
		path,
		func(path string, info os.FileInfo, e error) error {
			var f afero.File
			var err error
			if info == nil {
				return fmt.Errorf("path: %s not found", path)
			}
			if info.IsDir() {
				return nil
			}
			if f, err = s.AppFs.Fs.Open(path); err != nil {
				return err
			}
			if _, err := io.Copy(hash, f); err != nil {
				if err2 := f.Close(); err2 != nil {
					return fmt.Errorf("%s & %s", err, err2)
				}
				return err
			}
			return f.Close()
		},
	); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
