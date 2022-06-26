package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/show"
	"github.com/spf13/cobra"
)

var showType show.Type = "values"

var showCmd = &cobra.Command{
	Use:   "show <type> <environment.component>",
	Short: "show details from a deploy config helm chart",
	RunE:  showFn,
	Args:  cobra.ExactArgs(2),
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func showFn(_ *cobra.Command, args []string) error {
	if err := showType.Set(args[0]); err != nil {
		return err
	}
	env, comp, err := cfg.DeployIdParts(args[1])
	if err != nil {
		return err
	}

	config := newConfigService()
	deploy, err := config.GetDeploy(comp, env)
	if err != nil {
		return err
	}
	chartPath, err := config.ChartPath(*deploy)
	if err != nil {
		return err
	}
	output, err := show.Show(chartPath, showType)
	if err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}
