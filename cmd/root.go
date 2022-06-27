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
	"os"
)

var stdOut = io.ReadWriter(os.Stdout)
var stdIn = io.ReadWriter(os.Stdin)

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
	imageCmd.PersistentFlags().Var(&output, "output", "output [yaml, json]")
	rootCmd.PersistentFlags().StringVarP(&verbosity, "verbosity", "v", logrus.ErrorLevel.String(), "")
	rootCmd.PersistentFlags().StringVarP(&workdir, "workdir", "w", ".", "")
}

func initConfig() {
	lvl, err := logrus.ParseLevel(verbosity)
	cobra.CheckErr(err)
	log.SetLevel(lvl)
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log.SetOutput(stdOut)
}

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func newManifestService() *manifest.Svc {
	return manifest.NewSvc(afero.NewOsFs(), workdir, log)
}

func newConfigService() *cfg.Svc {
	return cfg.NewSvc(afero.NewOsFs(), workdir, log)
}

func newHashService() *hash.Svc {
	return hash.NewSvc(afero.NewOsFs(), log)
}

func newLockService() *cfg.Lock {
	return cfg.NewLock(afero.NewOsFs(), workdir, log)
}

func newMatcherService() *matcher.Svc {
	return matcher.NewSvc(afero.NewOsFs(), workdir, log)
}

func response(l interface{}) error {
	switch output {
	case "yaml":
		return asYaml(l)
	case "json":
		return asJson(l)
	default:
		return fmt.Errorf("output type %s not recognised", output)
	}
}

func asYaml(l interface{}) error {
	data, err := yaml.Marshal(l)
	cobra.CheckErr(err)
	_, err = stdOut.Write(data)
	return err
}

func asJson(l interface{}) error {
	data, err := json.Marshal(l)
	data = append(data, '\n')
	cobra.CheckErr(err)
	_, err = stdOut.Write(data)
	return err
}
