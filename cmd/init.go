package cmd

import (
	"github.com/spf13/cobra"
)

var force bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "init simple-ops structure",
	Run:   Init,
}

func init() {
	initCmd.PersistentFlags().BoolVar(&force, "force", false, "force init")
	rootCmd.AddCommand(initCmd)
}

func Init(cmd *cobra.Command, args []string) {
	config := newConfigService()
	cobra.CheckErr(config.Init(force, configTemplate))
}

var configTemplate = `
# paths used for labels and if the path should be created for a resource
# if it does not already exist.
fsslice:
  labels:
    - path: metadata/labels
      create: true
    - path: spec/template/metadata/labels
      create: false

labels:
  "app.kubernetes.io/managed-by": "simple-ops",

`
