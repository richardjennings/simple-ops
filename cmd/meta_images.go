package cmd

import (
	"errors"
	"github.com/richardjennings/simple-ops/internal/meta"
	"github.com/spf13/cobra"
)

var format imageListFormatType = "unique"

var imageCmd = &cobra.Command{
	Use:   "images [subcommand]",
	Short: "list images in manifests",
	RunE:  images,
}

func init() {
	imageCmd.PersistentFlags().Var(&format, "format", "format [unique, uniquePerFile]")
	metaCmd.AddCommand(imageCmd)
}

func images(_ *cobra.Command, _ []string) error {
	config := newConfigService()
	manifests := newManifestService()
	match := newMatcherService()
	metas := meta.NewSvc(config, manifests, match)
	var res interface{}
	var err error
	switch format {
	case "unique":
		res, err = metas.ListImagesUnique()
	case "uniquePerFile":
		res, err = metas.ListImagesUniquePerFile()
	}
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
