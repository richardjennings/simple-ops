package images

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
)

type (
	Svc struct {
		c *cfg.Svc
		m *manifest.Svc
		i *matcher.Svc
	}
	Images struct {
		FilePath string
		Images   []string
	}
)

func NewSvc(c *cfg.Svc, m *manifest.Svc, i *matcher.Svc) *Svc {
	return &Svc{c: c, m: m, i: i}
}

func (s Svc) ListImagesUniquePerFile() ([]Images, error) {
	var result []Images
	all, err := s.ListAllImages()
	if err != nil {
		return nil, err
	}
	for i, is := range all {
		hm := make(map[string]struct{})
		var result []string
		for _, img := range is.Images {
			if _, ok := hm[img]; !ok {
				hm[img] = struct{}{}
				result = append(result, img)
			}
		}
		all[i].Images = result
	}
	return result, nil
}

func (s Svc) ListAllImagesUnique() ([]string, error) {
	hm := make(map[string]struct{})
	var result []string
	all, err := s.ListAllImages()
	if err != nil {
		return nil, err
	}
	for _, is := range all {
		for _, i := range is.Images {
			if _, ok := hm[i]; !ok {
				result = append(result, i)
				hm[i] = struct{}{}
			}
		}
	}
	return result, nil
}

func (s Svc) ListAllImages() (res []Images, err error) {
	deploys, err := s.c.Deploys()
	if err != nil {
		return nil, err
	}
	for _, deploy := range deploys {
		path := s.m.ManifestPathForDeploy(deploy)
		is, err := s.i.Images(path)
		cobra.CheckErr(err)
		res = append(res, Images{FilePath: path, Images: is})
	}
	return res, nil
}

func (s Svc) ListUniqueImagesForDeploy(env string, comp string) ([]string, error) {
	var images []string
	hm := make(map[string]struct{})
	imgs, err := s.ListImagesForDeploy(env, comp)
	if err != nil {
		return nil, err
	}
	for _, i := range imgs {
		if _, ok := hm[i]; !ok {
			images = append(images, i)
			hm[i] = struct{}{}
		}
	}
	return images, nil
}

func (s Svc) ListImagesForDeploy(env string, comp string) ([]string, error) {
	deploy, err := s.c.GetDeploy(comp, env)
	if err != nil {
		return nil, err
	}
	path := s.m.ManifestPathForDeploy(deploy)
	return s.i.Images(path)
}
