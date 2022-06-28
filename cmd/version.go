package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
)

var Version = ""

var VsCmd = &cobra.Command{
	Use:   "version",
	Short: "version",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return Vs(cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(VsCmd)
}

func Vs(w io.Writer) error {
	_, err := fmt.Fprintln(w, Version)
	return err
}
