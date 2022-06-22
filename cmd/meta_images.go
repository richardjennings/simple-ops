package cmd

import (
	"encoding/json"
	"errors"
	"github.com/ghodss/yaml"
	"github.com/richardjennings/simple-ops/internal/meta"
	"github.com/spf13/cobra"
	"os"
)

var format imageListFormatType = "unique"

var imageCmd = &cobra.Command{
	Use:   "images [subcommand]",
	Short: "list images in manifests",
	Run:   images,
}

func init() {
	imageCmd.PersistentFlags().Var(&format, "format", "format [unique, uniquePerFile]")
	metaCmd.AddCommand(imageCmd)
}

func images(_ *cobra.Command, _ []string) {
	config := newConfigService()
	manifests := newManifestService()
	img := newMetaImageService()
	metas := meta.NewSvc(config, manifests, img)
	var res interface{}
	var err error
	switch format {
	case "unique":
		res, err = metas.ListImagesUnique()
	case "uniquePerFile":
		res, err = metas.ListImagesUniquePerFile()
	}
	cobra.CheckErr(err)
	switch output {
	case "yaml":
		cobra.CheckErr(asYaml(res))
	case "json":
		cobra.CheckErr(asJson(res))
	}
}

func asYaml(l interface{}) error {
	data, err := yaml.Marshal(l)
	cobra.CheckErr(err)
	_, err = os.Stdout.Write(data)
	return err
}

func asJson(l interface{}) error {
	data, err := json.Marshal(l)
	data = append(data, '\n')
	cobra.CheckErr(err)
	_, err = os.Stdout.Write(data)
	return err
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
