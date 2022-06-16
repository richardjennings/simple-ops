package compare

import (
	"crypto/sha256"
	"fmt"
	"github.com/spf13/afero"
	"io"
	"os"
)

type (
	Svc struct {
		AppFs afero.Afero
	}
)

func NewSvc(fs afero.Fs) *Svc {
	return &Svc{AppFs: afero.Afero{Fs: fs}}
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
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			return nil
		},
	); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
