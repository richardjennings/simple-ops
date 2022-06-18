package cmd

import (
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"os"
)

var imageCmd = &cobra.Command{
	Use:   "images [subcommand]",
	Short: "retrieve meta information from configurations and dependencies",
	Run:   images,
}

type listResult map[string]map[string][]string

func images(cmd *cobra.Command, args []string) {
	config := newConfigService()
	manifests := newManifestService()
	alldeploys, err := config.Deploys()
	cobra.CheckErr(err)
	img := newMetaImageService()
	result := make(listResult)

	for _, deploys := range alldeploys {
		for _, deploy := range deploys {
			path := manifests.ManifestPathForDeploy(deploy)
			if _, ok := result[deploy.Name]; !ok {
				result[deploy.Name] = make(map[string][]string)
			}
			result[deploy.Name][deploy.Component], err = img.ListImages(path)
		}
	}
	err = asYaml(result)
	cobra.CheckErr(err)
}

func asYaml(l listResult) error {
	var data []byte
	var err error
	data, err = yaml.Marshal(l)
	cobra.CheckErr(err)
	_, err = os.Stdout.Write(data)
	return err
}
