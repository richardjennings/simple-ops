package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/meta/show"
	"github.com/spf13/cobra"
	"path/filepath"
)

var showType show.Type = "values"

var showCmd = &cobra.Command{
	Use:   "show [component] [deploy]",
	Short: "show details from a deploy configuration helm chart",
	RunE:  showFn,
	Args:  cobra.ExactArgs(2),
}

func init() {
	showCmd.PersistentFlags().Var(&showType, "type", "show [values, ...]")
	metaCmd.AddCommand(showCmd)
}

func showFn(_ *cobra.Command, args []string) error {
	var deploy *cfg.Deploy
	compName := args[0]
	envName := args[1]
	config := newConfigService()
	deps, err := config.Deploys()
	cobra.CheckErr(err)
	if _, ok := deps[compName]; !ok {
		return fmt.Errorf("component %s not found", compName)
	}
	dep := deps[compName]
	for _, d := range dep {
		if d.Name == envName {
			deploy = d
		}
	}
	if deploy == nil {
		return fmt.Errorf("environment %s not found", envName)
	}
	chartPath, err := filepath.Abs(filepath.Join(workdir, deploy.RelativeChartPath()))
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
