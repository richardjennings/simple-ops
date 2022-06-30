package cfg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func TestLock_AddChart(t *testing.T) {
	name := "name"
	repository := "repository"
	version := "1.0.0"
	digest := "abc"

	l := NewLock(afero.NewMemMapFs(), "/test", logrus.New())
	err := l.appFs.WriteFile("/test/simple-ops.lock", []byte(""), 0777)
	assert.NilError(t, err)
	assert.NilError(t, l.AddChart(name, repository, version, digest))
	actual, err := l.appFs.ReadFile("/test/simple-ops.lock")
	assert.NilError(t, err)
	expected := fmt.Sprintf("charts:\n- chart: %s\n  digest: %s\n  repository: %s\n  version: %s\n", name, digest, repository, version)
	assert.Equal(t, string(actual), expected)
}

func TestLock_AddChart_NoLockFile(t *testing.T) {
	l := NewLock(afero.NewMemMapFs(), "/test", logrus.New())
	err := l.AddChart("a", "b", "c", "d")
	assert.ErrorContains(t, err, "open /test/simple-ops.lock: file does not exist")
}

func TestLock_AddChart_SkipDuplicate(t *testing.T) {
	l := NewLock(afero.NewMemMapFs(), "/test", logrus.New())
	err := l.appFs.WriteFile("/test/simple-ops.lock", []byte(""), 0777)
	assert.NilError(t, err)
	err = l.AddChart("a", "b", "c", "d")
	assert.NilError(t, err)
	err = l.AddChart("a", "b", "c", "d")
	assert.NilError(t, err)
	lf, err := l.LockFile()
	assert.NilError(t, err)
	assert.Equal(t, len(lf.Charts), 1)
	assert.DeepEqual(t, lf.Charts, []*ChartSource{{
		Name:       "a",
		Repository: "b",
		Version:    "c",
		Digest:     "d",
	}})
}
