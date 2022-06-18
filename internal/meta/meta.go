package meta

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/meta/image"
	"github.com/spf13/cobra"
)

type Svc struct {
	c *cfg.Svc
	m *manifest.Svc
	i *image.Svc
}

type ListImagesUniquePerFileResult struct {
	FilePath string   `json:"path"`
	Images   []string `json:"uniqueImages"`
}

func NewSvc(c *cfg.Svc, m *manifest.Svc, i *image.Svc) *Svc {
	return &Svc{c: c, m: m, i: i}
}

func (s Svc) ListImagesUniquePerFile() ([]ListImagesUniquePerFileResult, error) {
	var result []ListImagesUniquePerFileResult
	all, err := s.ListAllImages()
	if err != nil {
		return nil, err
	}
	for _, is := range all {
		result = append(result, ListImagesUniquePerFileResult{
			FilePath: is.FilePath,
			Images:   is.EveryUniqueImage(),
		})
	}
	return result, nil
}

func (s Svc) ListImagesUnique() ([]string, error) {
	hm := make(map[string]struct{})
	var result []string
	all, err := s.ListAllImages()
	if err != nil {
		return nil, err
	}
	for _, is := range all {
		for _, ir := range is.Images {
			for _, i := range ir.Images {
				if _, ok := hm[i]; !ok {
					result = append(result, i)
					hm[i] = struct{}{}
				}
			}
		}
	}
	return result, nil
}

func (s Svc) ListAllImages() (res []*image.Images, err error) {
	deps, err := s.c.Deploys()
	if err != nil {
		return nil, err
	}
	for _, deploys := range deps {
		for _, deploy := range deploys {
			path := s.m.ManifestPathForDeploy(deploy)
			is, err := s.i.ListImages(path)
			cobra.CheckErr(err)
			res = append(res, is)
		}
	}
	return res, nil
}
