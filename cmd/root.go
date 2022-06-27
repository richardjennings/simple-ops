package cmd

import (
	"encoding/json"
	"errors"
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

// the FS to use
var fs afero.Fs

var output outputType = "yaml"
var verbosity string
var workdir string

var rootCmd = &cobra.Command{
	Use:   "simple-ops",
	Short: "A simple GitOps workflow tool",
}

var log = logrus.New()

type outputType string

func (o *outputType) String() string {
	return string(*o)
}
func (o *outputType) Set(v string) error {
	switch v {
	case "yaml", "json":
		*o = outputType(v)
	default:
		return errors.New("supported output types are [yaml, json]")
	}
	return nil
}
func (o *outputType) Type() string {
	return "outputType"
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().Var(&output, "output", "output [yaml, json]")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.ErrorLevel.String(), "")
	rootCmd.PersistentFlags().StringVarP(&workdir, "workdir", "w", ".", "")
	log.SetOutput(rootCmd.OutOrStdout())
}

func initConfig() {
	fs = afero.NewOsFs()
	lvl, err := logrus.ParseLevel(verbosity)
	cobra.CheckErr(err)
	log.SetLevel(lvl)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func newManifestService() *manifest.Svc {
	return manifest.NewSvc(fs, workdir, log)
}

func newConfigService() *cfg.Svc {
	return cfg.NewSvc(fs, workdir, log)
}

func newHashService() *hash.Svc {
	return hash.NewSvc(fs, log)
}

func newLockService() *cfg.Lock {
	return cfg.NewLock(fs, workdir, log)
}

func newMatcherService() *matcher.Svc {
	return matcher.NewSvc(fs, workdir, log)
}

func response(l interface{}, w io.Writer) error {
	switch output {
	case "yaml":
		return asYaml(l, w)
	case "json":
		return asJson(l, w)
	default:
		return fmt.Errorf("output type %s not recognised", output)
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
