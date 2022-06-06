package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/pkg/config"
	"github.com/richardjennings/simple-ops/pkg/manifest"
	"github.com/spf13/afero"
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
	var deploys map[string]config.Deploys
	var err error
	cfg := config.NewSvc(afero.NewOsFs(), workdir)
	gen := manifest.NewSvc(afero.NewOsFs(), workdir)
	deploys, err = cfg.Deploys()
	cobra.CheckErr(err)
	correct, err := gen.Verify(deploys)
	cobra.CheckErr(err)
	if !correct {
		fmt.Println("deploy is not consistent with configuration")
		os.Exit(1)
	}
	fmt.Println("deploy is consistent with configuration")
}
