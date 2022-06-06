package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var verbosity string
var workdir string

var rootCmd = &cobra.Command{
	Use:   "simple-ops",
	Short: "A simple GitOps workflow tool",
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.ErrorLevel.String(), "")
	rootCmd.PersistentFlags().StringVarP(&workdir, "workdir", "w", ".", "")
}

func initConfig() {
	lvl, err := logrus.ParseLevel(verbosity)
	cobra.CheckErr(err)
	logrus.SetLevel(lvl)
	logrus.SetOutput(os.Stdout)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}
