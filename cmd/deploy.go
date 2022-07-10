package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
	"io"
)

var DepoyCmd = &cobra.Command{
	Use:   "deploy <deployId>",
	Short: "deploy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return DeployFn(cmd.OutOrStdout(), args[0], newConfigService())
	},
}

func init() {
	rootCmd.AddCommand(DepoyCmd)
}

func DeployFn(w io.Writer, id string, config *cfg.Svc) error {
	env, comp, err := cfg.DeployIdParts(id)
	if err != nil {
		return err
	}
	dep, err := config.GetDeploy(comp, env)
	if err != nil {
		return err
	}
	return response(dep, w)
}
