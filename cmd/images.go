package cmd

import (
	"errors"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/meta"
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
	return images()
}

func imagesForDeploy(environment string, component string) error {
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	metas := meta.NewSvc(config, manifests, match)
	var res interface{}
	var err error
	res, err = metas.ListUniqueImagesForDeploy(environment, component)
	cobra.CheckErr(err)
	return response(res)
}

func images() error {
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	metas := meta.NewSvc(config, manifests, match)
	var res interface{}
	var err error
	res, err = metas.ListAllImagesUnique()
	cobra.CheckErr(err)
	return response(res)
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
