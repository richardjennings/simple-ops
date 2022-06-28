package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/hash"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [chart name]",
	Short: "add a Helm chart as tgz",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return AddFn(args[0], flags.addRepository, flags.addVersion, flags.addConfig, newManifestService(), newHashService(), newLockService())
	},
}

func init() {
	addCmd.PersistentFlags().StringVar(&flags.addRepository, "repo", "", "")
	addCmd.PersistentFlags().StringVar(&flags.addVersion, "version", "", "")
	addCmd.PersistentFlags().BoolVar(&flags.addConfig, "add-config", false, "")
	rootCmd.AddCommand(addCmd)
}

func AddFn(name string, repository string, version string, addConfig bool, manifests *manifest.Svc, cmp *hash.Svc, lock *cfg.Lock) error {
	if err := manifests.Pull(name, repository, version); err != nil {
		return err
	}
	if addConfig {
		if err := manifests.PullAddConfig(name, version); err != nil {
			return err
		}
	}
	path := manifests.PathForChart(fmt.Sprintf("%s-%s.tgz", name, version))
	digest, err := cmp.SHA256File(path)
	if err != nil {
		return err
	}
	return lock.AddChart(name, repository, version, digest)
}
