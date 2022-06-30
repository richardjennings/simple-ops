package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
	"io"
)

var containerResourcesCmd = &cobra.Command{
	Use:   "container-resources [environment.component]",
	Short: "show container-resource configuration in ",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			env, comp, err := cfg.DeployIdParts(args[0])
			if err != nil {
				return err
			}
			return ContainerResourcesForDeploy(comp, env, cmd.OutOrStdout(), newConfigService(), newMatcherService())
		}
		return ContainerResourcesForDeploys(cmd.OutOrStdout(), newConfigService(), newMatcherService())
	},
}

func init() {
	rootCmd.AddCommand(containerResourcesCmd)
}

func ContainerResourcesForDeploy(compName string, envName string, w io.Writer, config *cfg.Svc, match *matcher.Svc) error {
	deploy, err := config.GetDeploy(compName, envName)
	if err != nil {
		return err
	}
	manifestPath, err := config.ManifestPath(deploy)
	if err != nil {
		return err
	}
	res, err := match.ContainerResources(manifestPath)
	if err != nil {
		return err
	}
	return response(res, w)
}

type DeployContainerResources struct {
	Name      string
	Resources matcher.ContainerResources
}

func ContainerResourcesForDeploys(w io.Writer, config *cfg.Svc, match *matcher.Svc) error {
	var result []DeployContainerResources
	deploys, err := config.Deploys()
	if err != nil {
		return err
	}
	for _, d := range deploys {
		manifestPath, err := config.ManifestPath(d)
		if err != nil {
			return err
		}
		res, err := match.ContainerResources(manifestPath)
		if err != nil {
			return err
		}
		result = append(result, DeployContainerResources{Name: d.Id(), Resources: res})
	}
	return response(result, w)
}
