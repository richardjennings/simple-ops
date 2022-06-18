package cmd

import (
	"github.com/spf13/cobra"
)

var metaCmd = &cobra.Command{
	Use:   "meta [command]",
	Short: "retrieve meta information from configurations and dependencies",
}

func init() {
	metaCmd.AddCommand(imageCmd)
	rootCmd.AddCommand(metaCmd)
}
