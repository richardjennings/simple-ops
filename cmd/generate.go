package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate deployment manifests from config",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return GenerateFn()
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func GenerateFn() error {
	var deploys cfg.Deploys
	var err error
	config := newConfigService()
	manifests := newManifestService()
	deploys, err = config.Deploys()
	if err != nil {
		return err
	}
	return manifests.Generate(deploys)
}
