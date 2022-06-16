package cmd

import (
	"errors"
	"github.com/richardjennings/simple-ops/internal/config"
	"github.com/spf13/afero"
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
	c := config.NewSvc(afero.NewOsFs(), workdir)
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
	cobra.CheckErr(c.Set(path, value))
}
