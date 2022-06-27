package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/show"
	"github.com/spf13/cobra"
	"io"
)

var showType show.Type = "values"

var showCmd = &cobra.Command{
	Use:   "show <type> <environment.component>",
	Short: "show details from a deploy config helm chart",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env, comp, err := cfg.DeployIdParts(args[1])
		if err != nil {
			return err
		}
		return ShowFn(args[0], env, comp, cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func ShowFn(thing string, environment string, component string, w io.Writer) error {
	if err := showType.Set(thing); err != nil {
		return err
	}
	config := newConfigService()
	deploy, err := config.GetDeploy(component, environment)
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
	_, err = fmt.Fprintln(w, output)
	return err
}
