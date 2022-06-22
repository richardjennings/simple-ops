package meta

import "github.com/spf13/cobra"

type (
	Images struct {
		FilePath string
		Images   []string
	}
)

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

func (s Svc) ListImagesUnique() ([]string, error) {
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
	deps, err := s.c.Deploys()
	if err != nil {
		return nil, err
	}
	for _, deploys := range deps {
		for _, deploy := range deploys {
			path := s.m.ManifestPathForDeploy(deploy)
			is, err := s.i.Images(path)
			cobra.CheckErr(err)
			res = append(res, Images{FilePath: path, Images: is})
		}
	}
	return res, nil
}
