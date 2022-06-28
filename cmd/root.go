package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/hash"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/richardjennings/simple-ops/internal/matcher"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"io"
)

type options struct {
	verbosity     string
	workdir       string
	output        string
	addRepository string
	addVersion    string
	addConfig     bool
	initForce     bool
	setStdin      bool
	setType       string
}

var flags options

// the FS to use
var fs afero.Fs

var rootCmd = &cobra.Command{
	Use:   "simple-ops",
	Short: "A simple GitOps workflow tool",
}

var log = logrus.New()

// reset flag option values to defaults allowing for commands to
// be used with flags multiple times without side effects
func defaultFlags() {
	flags.verbosity = "error"
	flags.workdir = "."
	flags.output = "yaml"
	flags.addRepository = ""
	flags.addVersion = ""
	flags.addConfig = false
	flags.initForce = false
	flags.setStdin = false
	flags.setType = "string"
}

func init() {
	defaultFlags()
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&flags.output, "output", "o", "yaml", "output [yaml, json]")
	rootCmd.PersistentFlags().StringVarP(&flags.verbosity, "verbosity", "v", logrus.ErrorLevel.String(), "")
	rootCmd.PersistentFlags().StringVarP(&flags.workdir, "workdir", "w", ".", "")
	log.SetOutput(rootCmd.OutOrStdout())
}

func initConfig() {
	fs = afero.NewOsFs()
	lvl, err := logrus.ParseLevel(flags.verbosity)
	cobra.CheckErr(err)
	log.SetLevel(lvl)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func newManifestService() *manifest.Svc {
	return manifest.NewSvc(fs, flags.workdir, log)
}

func newConfigService() *cfg.Svc {
	return cfg.NewSvc(fs, flags.workdir, log)
}

func newHashService() *hash.Svc {
	return hash.NewSvc(fs, log)
}

func newLockService() *cfg.Lock {
	return cfg.NewLock(fs, flags.workdir, log)
}

func newMatcherService() *matcher.Svc {
	return matcher.NewSvc(fs, flags.workdir, log)
}

func response(l interface{}, w io.Writer) error {
	switch flags.output {
	case "yaml":
		return asYaml(l, w)
	case "json":
		return asJson(l, w)
	default:
		return fmt.Errorf("output type %s not recognised", flags.output)
	}
}

func asYaml(l interface{}, w io.Writer) error {
	data, err := yaml.Marshal(l)
	cobra.CheckErr(err)
	_, err = w.Write(data)
	return err
}

func asJson(l interface{}, w io.Writer) error {
	data, err := json.Marshal(l)
	data = append(data, '\n')
	cobra.CheckErr(err)
	_, err = w.Write(data)
	return err
}
