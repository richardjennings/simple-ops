package cmd

import (
	"errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strconv"
)

var stdin bool
var setType string

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "modify configuration",
	Args:  cobra.RangeArgs(1, 2),
	Run:   SetFn,
}

func init() {
	setCmd.PersistentFlags().BoolVar(&stdin, "stdin", false, "")
	setCmd.PersistentFlags().StringVar(&setType, "setType", "", "")
	rootCmd.AddCommand(setCmd)
}

func SetFn(cmd *cobra.Command, args []string) {
	var value string
	var v interface{}
	var err error
	path := args[0]
	config := newConfigService()
	if stdin {
		v, err := ioutil.ReadAll(stdIn)
		cobra.CheckErr(err)
		value = string(v)
	} else {
		if len(args) != 2 {
			cobra.CheckErr(errors.New("expected 2 arguments"))
		}
		value = args[1]
	}
	switch setType {
	case "bool":
		v, err = strconv.ParseBool(value)
		cobra.CheckErr(err)
	case "int":
		v, err = strconv.Atoi(value)
		cobra.CheckErr(err)
	default:
		v = value
	}
	cobra.CheckErr(config.Set(path, v))
}
