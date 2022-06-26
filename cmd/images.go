package cmd

import (
	"errors"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:   "images [subcommand]",
	Short: "list images in manifests",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  imagesFn,
}

func init() {
	rootCmd.AddCommand(imageCmd)
}

func imagesFn(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		env, comp, err := cfg.DeployIdParts(args[0])
		if err != nil {
			return err
		}
		return imagesForDeploy(env, comp)
	}
	return allImages()
}

func imagesForDeploy(environment string, component string) error {
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	d, err := config.GetDeploy(component, environment)
	if err != nil {
		return err
	}
	imgs, err := match.Images(manifests.ManifestPathForDeploy(d))
	if err != nil {
		return err
	}
	return response(imgs)
}

func allImages() error {
	var images matcher.Images
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	deploys, err := config.Deploys()
	if err != nil {
		return err
	}
	for _, d := range deploys {
		imgs, err := match.Images(manifests.ManifestPathForDeploy(d))
		if err != nil {
			return err
		}
		images = append(images, imgs...)
	}
	return response(images.Unique())
}

type imageListFormatType string

func (o *imageListFormatType) String() string {
	return string(*o)
}
func (o *imageListFormatType) Set(v string) error {
	switch v {
	case "unique", "uniquePerFile":
		*o = imageListFormatType(v)
	default:
		return errors.New("supported output types are [yaml, json]")
	}
	return nil
}
func (o *imageListFormatType) Type() string {
	return "outputType"
}
