//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"os"
	"testing"
)

func testSetup(t *testing.T) string {
	path, err := afero.TempDir(afero.NewOsFs(), "", "simple-ops-integration-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("using tmp dir %s", path)
	return path
}

func testTearDown(t *testing.T, path string) {
	err := os.RemoveAll(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("removed tmp dir %s", path)
}

func Test_Integration(t *testing.T) {
	stdOut = bytes.NewBuffer(nil)
	workdir = testSetup(t)
	repository = "https://kubernetes-sigs.github.io/metrics-server/"
	version = "3.8.2"
	addConfig = true
	defer testTearDown(t, workdir)

	InitFn(nil, []string{})
	AddFn(nil, []string{"metrics-server"})
	setType = "bool"
	SetFn(nil, []string{"metrics-server.deploy.test.values.metrics.enable", "true"})
	setType = ""
	SetFn(nil, []string{"metrics-server.deploy.test.namespace.name", "metrics-server"})
	setType = "bool"
	SetFn(nil, []string{"metrics-server.deploy.test.namespace.create", "true"})
	SetFn(nil, []string{"metrics-server.deploy.test.namespace.inject", "true"})
	GenerateFn(nil, []string{})
	if err := ContainerResourcesFn(nil, []string{"test.metrics-server"}); err != nil {
		t.Error(err)
	}
	actual := stdOut.(*bytes.Buffer).String()
	expected := "- name: metrics-server\n  parentName: metrics-server\n  parentType: Deployment\n  resources:\n    limits: {}\n    requests: {}\n"
	assert.Equal(t, actual, expected)
}
