package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/spf13/afero"
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

func Generate(cmd *cobra.Command, args []string) {
	var deploys map[string]cfg.Deploys
	var err error
	cfg := cfg.NewSvc(afero.NewOsFs(), workdir, log)
	gen := manifest.NewSvc(afero.NewOsFs(), workdir, log)
	deploys, err = cfg.Deploys()
	cobra.CheckErr(err)
	cobra.CheckErr(gen.Generate(deploys))
}
