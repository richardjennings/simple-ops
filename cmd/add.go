package cmd

import (
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
	name := args[0]
	manifests := newManifestService()
	cobra.CheckErr(manifests.Pull(name, repository, version, addConfig))
	lock := newLockService()
	cobra.CheckErr(lock.AddChart(name, repository, version))
}
