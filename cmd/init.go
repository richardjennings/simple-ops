package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var force bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init simple-ops structure",
	Run:   Init,
}

func init() {
	initCmd.PersistentFlags().BoolVar(&force, "force", false, "force init")
	rootCmd.AddCommand(initCmd)
}

func Init(cmd *cobra.Command, args []string) {
	c := cfg.NewSvc(afero.NewOsFs(), workdir, log)
	cobra.CheckErr(c.Init(force))
}
