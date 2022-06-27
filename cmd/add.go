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
	RunE: func(_ *cobra.Command, args []string) error {
		return AddFn(args[0], repository, version, addConfig)
	},
}

func init() {
	addCmd.PersistentFlags().StringVar(&repository, "repo", "", "")
	addCmd.PersistentFlags().StringVar(&version, "version", "", "")
	addCmd.PersistentFlags().BoolVar(&addConfig, "add-config", false, "")
	rootCmd.AddCommand(addCmd)
}

func AddFn(name string, repository string, version string, addConfig bool) error {
	manifests := newManifestService()
	if err := manifests.Pull(name, repository, version, addConfig); err != nil {
		return err
	}
	path := manifests.PathForChart(fmt.Sprintf("%s-%s.tgz", name, version))
	cmp := newHashService()
	hash, err := cmp.SHA256File(path)
	if err != nil {
		return err
	}
	lock := newLockService()
	return lock.AddChart(name, repository, version, hash)
}
