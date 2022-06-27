package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var Version = ""

var VsCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Args:  cobra.ExactArgs(0),
	RunE:  Vs,
}

func init() {
	rootCmd.AddCommand(VsCmd)
}

func Vs(cmd *cobra.Command, args []string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
	return err
}
