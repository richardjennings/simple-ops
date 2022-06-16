package cmd

import (
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var repository string
var version string
var addConfig bool

var addCmd = &cobra.Command{
	Use:   "add [chart name]",
	Short: "add a Helm chart as tgz",
	Args:  cobra.ExactArgs(1),
	Run:   Add,
}

func init() {
	addCmd.PersistentFlags().StringVar(&repository, "repo", "", "")
	addCmd.PersistentFlags().StringVar(&version, "version", "", "")
	addCmd.PersistentFlags().BoolVar(&addConfig, "add-config", false, "")
	rootCmd.AddCommand(addCmd)
}

func Add(cmd *cobra.Command, args []string) {
	s := manifest.NewSvc(afero.NewOsFs(), workdir)
	cobra.CheckErr(s.Pull(args[0], repository, version, addConfig))
}
