package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

var stdin bool

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "modify configuration",
	Args:  cobra.RangeArgs(1, 2),
	Run:   Set,
}

func init() {
	setCmd.PersistentFlags().BoolVar(&stdin, "stdin", false, "")
	rootCmd.AddCommand(setCmd)
}

func Set(cmd *cobra.Command, args []string) {
	var value string
	path := args[0]
	config := newConfigService()
	if stdin {
		v, err := ioutil.ReadAll(os.Stdin)
		cobra.CheckErr(err)
		value = string(v)
	} else {
		if len(args) != 2 {
			cobra.CheckErr(errors.New("expected 2 arguments"))
		}
		value = args[1]
	}
	cobra.CheckErr(config.Set(path, value))
}
