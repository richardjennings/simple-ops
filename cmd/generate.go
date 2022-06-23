package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate deployment manifests from config",
	Run:   Generate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func Generate(_ *cobra.Command, _ []string) {
	var deploys cfg.Deploys
	var err error
	config := newConfigService()
	manifests := newManifestService()
	deploys, err = config.Deploys()
	cobra.CheckErr(err)
	cobra.CheckErr(manifests.Generate(deploys))
}
