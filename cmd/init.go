package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init simple-ops structure",
	RunE: func(_ *cobra.Command, _ []string) error {
		return InitFn(flags.initForce, newConfigService())
	},
}

func init() {
	initCmd.PersistentFlags().BoolVar(&flags.initForce, "force", false, "force init")
	rootCmd.AddCommand(initCmd)
}

func InitFn(force bool, config *cfg.Svc) error {
	if force {
		return config.Init(configTemplate)
	}
	return config.InitIfEmpty(configTemplate)
}

var configTemplate = `
# fsslice.labels configures the kustomizable field paths in k8s API resources applicable to labels, 
# optionally creating field paths in resources if they do not exist.
fsslice:
  labels:
    - path: metadata/labels
      create: true
    - path: spec/template/metadata/labels
      create: false

# apply this label to all resources matched by fsslice.labels
labels:
  "app.kubernetes.io/managed-by": "simple-ops"

# Relative directories to be copied into the temporary build context where
# kustimze build is run by simple-ops such that any relative paths
# defined in kustomizations are made to exist by copying from the
# working directory to the temporary build context
kustomizationPaths:
`
