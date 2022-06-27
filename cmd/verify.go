package cmd

import (
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "verify deployment manifests match config",
	RunE:  VerifyFn,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func VerifyFn(cmd *cobra.Command, _ []string) error {
	var deploys cfg.Deploys
	var err error
	var invalid bool
	w := cmd.OutOrStdout()
	config := newConfigService()
	manifests := newManifestService()
	deploys, err = config.Deploys()
	cobra.CheckErr(err)
	correct, err := manifests.Verify(deploys)
	cobra.CheckErr(err)
	if !correct {
		log.Error("deploy is not consistent with configuration")
		invalid = true
	}
	_, err = fmt.Fprintln(w, "deploy is consistent with configuration")
	if err != nil {
		return err
	}
	// verify charts
	lock := newLockService()
	l, err := lock.LockFile()
	cobra.CheckErr(err)
	for _, c := range l.Charts {
		path := manifests.PathForChart(fmt.Sprintf("%s-%s.tgz", c.Name, c.Version))
		hash := newHashService()
		digest, err := hash.SHA256File(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Errorf("Chart %s-%s.tgz missing", c.Name, c.Version)
			}
		}
		cobra.CheckErr(err)
		if digest != c.Digest {
			log.Errorf("Chart %s-%s.tgz lock digest mismatch", c.Name, c.Version)
			invalid = true
		} else {
			log.Debugf("Chart %s-%s.tgz digest matched lock file", c.Name, c.Version)
		}
	}
	// check no tgz charts outside of lock file
	dirEntries, err := os.ReadDir(filepath.Join(workdir, cfg.ChartsPath))
	cobra.CheckErr(err)
	for _, d := range dirEntries {
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".tgz") {
			matched := false
			for _, c := range l.Charts {
				if d.Name() == fmt.Sprintf("%s-%s.tgz", c.Name, c.Version) {
					matched = true
					break
				}
			}
			if !matched {
				log.Errorf("%s not in lock file", d.Name())
				invalid = true
			}
		}
	}

	if invalid {
		os.Exit(1)
	}
	_, err = fmt.Fprintln(w, "charts in lock file are consistent")
	return err
}
