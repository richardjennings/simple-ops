package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
	"io"
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

func imagesFn(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	if len(args) == 1 {
		env, comp, err := cfg.DeployIdParts(args[0])
		if err != nil {
			return err
		}
		return imagesForDeploy(env, comp, w)
	}
	return allImages(w)
}

func imagesForDeploy(environment string, component string, w io.Writer) error {
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
	return response(imgs, w)
}

func allImages(w io.Writer) error {
	var images matcher.Images
	var imgs matcher.Images
	var deploys cfg.Deploys
	var err error
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	deploys, err = config.Deploys()
	if err != nil {
		return err
	}
	for _, d := range deploys {
		imgs, err = match.Images(manifests.ManifestPathForDeploy(d))
		if err != nil {
			return err
		}
		images = append(images, imgs...)
	}
	return response(images.Unique(), w)
}
