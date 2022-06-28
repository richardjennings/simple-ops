package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
	"io"
)

var imageCmd = &cobra.Command{
	Use:   "images [subcommand]",
	Short: "list images in manifests",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		if len(args) == 1 {
			env, comp, err := cfg.DeployIdParts(args[0])
			if err != nil {
				return err
			}
			return ImagesForDeploy(env, comp, w, newConfigService(), newManifestService(), newMatcherService())
		}
		return AllImages(w, newConfigService(), newManifestService(), newMatcherService())
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
}

func ImagesForDeploy(environment string, component string, w io.Writer, config *cfg.Svc, manifests *manifest.Svc, match *matcher.Svc) error {
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

func AllImages(w io.Writer, config *cfg.Svc, manifests *manifest.Svc, match *matcher.Svc) error {
	var images matcher.Images
	var imgs matcher.Images
	var deploys cfg.Deploys
	var err error
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
