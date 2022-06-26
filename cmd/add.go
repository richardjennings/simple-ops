package cmd

import (
	"fmt"
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

func Add(_ *cobra.Command, args []string) {
	name := args[0]
	manifests := newManifestService()
	cobra.CheckErr(manifests.Pull(name, repository, version, addConfig))
	path := manifests.PathForChart(fmt.Sprintf("%s-%s.tgz", name, version))
	cmp := newCompareService()
	hash, err := cmp.SHA256File(path)
	cobra.CheckErr(err)
	lock := newLockService()
	cobra.CheckErr(lock.AddChart(name, repository, version, hash))
}
