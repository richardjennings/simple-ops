package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate deployment manifests from config",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return GenerateFn(newConfigService(), newManifestService())
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func GenerateFn(config *cfg.Svc, manifests *manifest.Svc) error {
	var deploys cfg.Deploys
	var err error
	deploys, err = config.Deploys()
	if err != nil {
		return err
	}
	return manifests.Generate(deploys)
}
