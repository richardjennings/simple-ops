package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
	"os"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "verify deployment manifests match config",
	Run:   Verify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func Verify(cmd *cobra.Command, args []string) {
	var deploys map[string]cfg.Deploys
	var err error
	config := newConfigService()
	manifests := newManifestService()
	deploys, err = config.Deploys()
	cobra.CheckErr(err)
	correct, err := manifests.Verify(deploys)
	cobra.CheckErr(err)
	if !correct {
		fmt.Println("deploy is not consistent with configuration")
		os.Exit(1)
	}
	fmt.Println("deploy is consistent with configuration")
}
