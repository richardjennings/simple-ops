package cmd

import (
	"github.com/spf13/cobra"
)

var resourcesCmd = &cobra.Command{
	Use:   "resources [component] [deploy]",
	Short: "show resource configuration in ",
	RunE:  resourcesFn,
	Args:  cobra.ExactArgs(2),
}

var missing bool

func init() {
	metaCmd.AddCommand(resourcesCmd)
}

func resourcesFn(_ *cobra.Command, args []string) error {
	compName := args[0]
	envName := args[1]
	config := newConfigService()
	matcher := newMatcherService()
	deploy, err := config.GetDeploy(compName, envName)
	if err != nil {
		return err
	}
	manifestPath, err := config.ManifestPath(*deploy)
	if err != nil {
		return err
	}
	res, err := matcher.Resources(manifestPath)
	if err != nil {
		return err
	}
	return response(res)
}
