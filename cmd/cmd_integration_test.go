//go:build integration

package cmd

import (
	"bytes"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
	"os"
	"strings"
	"testing"
)

type integration struct {
	in  *bytes.Buffer
	out *bytes.Buffer
	err *bytes.Buffer
	t   *testing.T
}

func newIntegration(t *testing.T) integration {
	return integration{
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
		t,
	}
}

func (i integration) resetBuffers() {
	i.out.Reset()
	i.err.Reset()
}

func (i integration) testSetup() {
	// set logger out to buffer
	log.SetOutput(i.out)
	// create tmp in memory working directory
	path, err := afero.TempDir(afero.NewOsFs(), "", "simple-ops-integration-")
	if err != nil {
		i.t.Fatal(err)
	}
	i.t.Logf("using tmp dir %s", path)
	// global used by newSvc functions ...
	workdir = path // or should it be set via -w t.workdir flag ?
	rootCmd.SetOut(i.out)
	rootCmd.SetIn(i.in)
	rootCmd.SetErr(i.err)
}

func (i integration) testTearDown() {
	err := os.RemoveAll(workdir)
	if err != nil {
		i.t.Fatal(err)
	}
	i.t.Logf("removed tmp dir %s", workdir)
}

func (i integration) runCmd(cmd *cobra.Command) (string, string) {
	if err := cmd.Execute(); err != nil {
		i.t.Error(err)
	}
	defer i.resetBuffers()
	return i.out.String(), i.err.String()
}

func (i integration) Init() (string, string) {
	rootCmd.SetArgs([]string{"init"})
	return i.runCmd(rootCmd)
}

func (i integration) Add(name string, repository string, version string, addConfig bool) (string, string) {
	args := []string{"add", name, "--repo", repository, "--version", version}
	if addConfig {
		args = append(args, "--add-config")
	}
	rootCmd.SetArgs(args)
	return i.runCmd(rootCmd)
}

func (i integration) Set(key string, value string, as string) (string, string) {
	rootCmd.SetArgs([]string{"set", key, value, "--type", as})
	return i.runCmd(rootCmd)
}

func (i integration) Generate() (string, string) {
	rootCmd.SetArgs([]string{"generate"})
	return i.runCmd(rootCmd)
}

func (i integration) ContainerResources(id string, outputType string) (string, string) {
	rootCmd.SetArgs([]string{"container-resources", id, "--output", outputType})
	return i.runCmd(rootCmd)
}

func (i integration) Images(id string, outputType string) (string, string) {
	rootCmd.SetArgs([]string{"images", id, "--output", outputType})
	return i.runCmd(rootCmd)
}

func (i integration) Show(thing string, id string) (string, string) {
	rootCmd.SetArgs([]string{"show", thing, id})
	return i.runCmd(rootCmd)
}

func (i integration) Verify() (string, string) {
	rootCmd.SetArgs([]string{"verify"})
	return i.runCmd(rootCmd)
}

func Test_Integration(t *testing.T) {
	var o, e, expected string
	i := newIntegration(t)
	i.testSetup()
	defer i.testTearDown()

	// init
	o, e = i.Init()
	assert.Assert(t, o == "" && e == "")

	// add
	o, e = i.Add("metrics-server", "https://kubernetes-sigs.github.io/metrics-server/", "3.8.2", true)
	assert.Assert(t, o == "" && e == "")

	// set bool value
	o, e = i.Set("metrics-server.deploy.test.values.metrics.enable", "true", "bool")
	assert.Assert(t, o == "" && e == "")

	// set string
	o, e = i.Set("metrics-server.deploy.test.namespace.name", "metrics-server", "")
	assert.Assert(t, o == "" && e == "")

	// set enable namespace create and inject
	o, e = i.Set("metrics-server.deploy.test.namespace.create", "true", "bool")
	assert.Assert(t, o == "" && e == "")
	o, e = i.Set("metrics-server.deploy.test.namespace.inject", "true", "bool")
	assert.Assert(t, o == "" && e == "")

	// generate
	o, e = i.Generate()
	assert.Assert(t, o == "" && e == "")

	// container resources yaml
	o, e = i.ContainerResources("test.metrics-server", "yaml")
	expected = "- name: metrics-server\n  parentName: metrics-server\n  parentType: Deployment\n  resources:\n    limits: {}\n    requests: {}\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// container resources json
	o, e = i.ContainerResources("test.metrics-server", "json")
	expected = "[{\"parentName\":\"metrics-server\",\"parentType\":\"Deployment\",\"name\":\"metrics-server\",\"resources\":{\"limits\":{},\"requests\":{}}}]\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// images for deployment
	o, e = i.Images("test.metrics-server", "yaml")
	expected = "- k8s.gcr.io/metrics-server/metrics-server:v0.6.1\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// show chart
	o, e = i.Show("chart", "test.metrics-server")
	assert.Assert(t, strings.Contains(o, "appVersion: 0.6.1") == true)
	assert.Equal(t, e, "")

	// verify
	o, e = i.Verify()
	expected = "deploy is consistent with configuration\ncharts in lock file are consistent\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

}
