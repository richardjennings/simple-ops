package cmd

import (
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/meta/image"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"os"
)

var verbosity string
var workdir string

var rootCmd = &cobra.Command{
	Use:   "simple-ops",
	Short: "A simple GitOps workflow tool",
}

var log = logrus.New()

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.ErrorLevel.String(), "")
	rootCmd.PersistentFlags().StringVarP(&workdir, "workdir", "w", ".", "")
}

func initConfig() {
	lvl, err := logrus.ParseLevel(verbosity)
	cobra.CheckErr(err)
	log.SetLevel(lvl)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log.SetOutput(os.Stdout)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func newManifestService() *manifest.Svc {
	return manifest.NewSvc(afero.NewOsFs(), workdir, log)
}

func newConfigService() *cfg.Svc {
	return cfg.NewSvc(afero.NewOsFs(), workdir, log)
}

func newMetaImageService() *image.Svc {
	return image.NewSvc(afero.NewOsFs(), workdir, log)
}
