package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/meta/show"
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
	showCmd.PersistentFlags().Var(&showType, "type", "show [values, ...]")
	rootCmd.AddCommand(showCmd)
}

func showFn(_ *cobra.Command, args []string) error {
	compName := args[0]
	envName := args[1]
	config := newConfigService()
	deploy, err := config.GetDeploy(compName, envName)
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
