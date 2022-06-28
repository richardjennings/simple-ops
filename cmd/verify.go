package cmd

import (
	"errors"
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/richardjennings/simple-ops/internal/hash"
	"github.com/richardjennings/simple-ops/internal/manifest"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "verify deployment manifests match config",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return VerifyFn(cmd.OutOrStdout(), newConfigService(), newManifestService(), newLockService(), newHashService())
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func VerifyFn(w io.Writer, config *cfg.Svc, manifests *manifest.Svc, lock *cfg.Lock, h *hash.Svc) error {
	var deploys cfg.Deploys
	var err error
	var invalid bool
	deploys, err = config.Deploys()
	if err != nil {
		return err
	}
	correct, err := manifests.Verify(deploys)
	if err != nil {
		return err
	}
	if !correct {
		log.Error("deploy is not consistent with configuration")
		invalid = true
	}
	_, err = fmt.Fprintln(w, "deploy is consistent with configuration")
	if err != nil {
		return err
	}
	// verify charts
	l, err := lock.LockFile()
	if err != nil {
		return err
	}
	for _, c := range l.Charts {
		path := manifests.PathForChart(fmt.Sprintf("%s-%s.tgz", c.Name, c.Version))
		digest, err := h.SHA256File(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Errorf("Chart %s-%s.tgz missing", c.Name, c.Version)
			}
		}
		if err != nil {
			return err
		}
		if digest != c.Digest {
			log.Errorf("Chart %s-%s.tgz lock digest mismatch", c.Name, c.Version)
			invalid = true
		} else {
			log.Debugf("Chart %s-%s.tgz digest matched lock file", c.Name, c.Version)
		}
	}
	// check no tgz charts outside of lock file
	dirEntries, err := os.ReadDir(filepath.Join(flags.workdir, cfg.ChartsPath))
	if err != nil {
		return err
	}
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
		return errors.New("inconsistent")
	}
	_, err = fmt.Fprintln(w, "charts in lock file are consistent")
	return err
}
