package cmd

import (
	"errors"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
	"io/ioutil"
	"strconv"
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "modify configuration",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var value string
		if flags.setStdin {
			v, err := ioutil.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			value = string(v)
		} else {
			if len(args) != 2 {
				return errors.New("expected 2 arguments")
			}
			value = args[1]
		}
		return SetFn(args[0], value, flags.setType, newConfigService())
	},
}

func init() {
	setCmd.PersistentFlags().BoolVar(&flags.setStdin, "stdin", false, "")
	setCmd.PersistentFlags().StringVar(&flags.setType, "type", "", "--type [string, bool, int]")
	rootCmd.AddCommand(setCmd)
}

func SetFn(path string, value string, setType string, config *cfg.Svc) error {
	var v interface{}
	var err error
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
	return config.Set(path, v)
}
