package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/spf13/cobra"
)

var containerResourcesCmd = &cobra.Command{
	Use:   "container-resources [environment.component]",
	Short: "show container-resource configuration in ",
	RunE:  ContainerResourcesFn,
	Args:  cobra.RangeArgs(0, 1),
}

func init() {
	rootCmd.AddCommand(containerResourcesCmd)
}

func ContainerResourcesFn(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		env, comp, err := cfg.DeployIdParts(args[0])
		if err != nil {
			return err
		}
		return containerResourcesForDeploy(comp, env)
	}
	return containerResourcesForDeploys()
}

func containerResourcesForDeploy(compName string, envName string) error {
	config := newConfigService()
	match := newMatcherService()
	deploy, err := config.GetDeploy(compName, envName)
	if err != nil {
		return err
	}
	manifestPath, err := config.ManifestPath(*deploy)
	if err != nil {
		return err
	}
	res, err := match.ContainerResources(manifestPath)
	if err != nil {
		return err
	}
	return response(res)
}

type DeployContainerResources struct {
	Name      string
	Resources matcher.ContainerResources
}

func containerResourcesForDeploys() error {
	var result []DeployContainerResources
	config := newConfigService()
	match := newMatcherService()
	deploys, err := config.Deploys()
	if err != nil {
		return err
	}
	for _, d := range deploys {
		manifestPath, err := config.ManifestPath(*d)
		if err != nil {
			return err
		}
		res, err := match.ContainerResources(manifestPath)
		if err != nil {
			return err
		}
		result = append(result, DeployContainerResources{Name: d.Id(), Resources: res})
	}
	return response(result)
}
