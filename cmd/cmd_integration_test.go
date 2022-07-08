//go:build integration

package cmd

import (
	"bytes"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gotest.tools/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test_Integration runs through the process of:
// Init a new working directory
// Add a chart
// Set configuration values
// Generate deployment manifests
// Inspect container resources configuration
// Inspect images
// Show chart values
// Verify manifests are correct
// Print out version
// Add a Kustomization config
// Generate
// Check Kustomization has been applied
func Test_Integration(t *testing.T) {
	var o, e, expected string
	i := newIntegration(t)
	i.testSetup()
	defer i.testTearDown(i.workDir)

	// init
	o, e = i.Init(false)
	assert.Assert(t, o == "" && e == "")

	// add
	o, e = i.Add("metrics-server", "https://kubernetes-sigs.github.io/metrics-server/", "3.8.2", true)
	assert.Assert(t, o == "" && e == "")

	// set bool value
	o, e = i.Set("metrics-server.deploy.test.values.metrics.enable", "true", "bool", false)
	assert.Assert(t, o == "" && e == "")

	// set string
	o, e = i.Set("metrics-server.deploy.test.namespace.name", "metrics-server", "", false)
	assert.Assert(t, o == "" && e == "")

	// set enable namespace create bool from stdin
	i.in.Write([]byte("true"))
	o, e = i.Set("metrics-server.deploy.test.namespace.create", "", "bool", true)
	assert.Assert(t, o == "" && e == "")

	// set enable namespace inject bool
	o, e = i.Set("metrics-server.deploy.test.namespace.inject", "true", "bool", false)
	assert.Assert(t, o == "" && e == "")

	// generate
	o, e = i.Generate()
	assert.Assert(t, o == "" && e == "")

	// container resources yaml
	o, e = i.ContainerResources("test.metrics-server", "yaml")
	expected = "- name: metrics-server\n  parentName: metrics-server\n  parentType: Deployment\n  resources:\n    limits: {}\n    requests: {}\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// container resources all json
	o, e = i.ContainerResources("", "json")
	expected = "[{\"Name\":\"test.metrics-server\",\"Resources\":[{\"parentName\":\"metrics-server\",\"parentType\":\"Deployment\",\"name\":\"metrics-server\",\"resources\":{\"limits\":{},\"requests\":{}}}]}]\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// images for deployment yaml
	o, e = i.Images("test.metrics-server", "yaml")
	expected = "- k8s.gcr.io/metrics-server/metrics-server:v0.6.1\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// images all json
	o, e = i.Images("", "json")
	expected = "[\"k8s.gcr.io/metrics-server/metrics-server:v0.6.1\"]\n"
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

	Version = "1.2.3"
	o, e = i.Version()
	expected = "1.2.3\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")

	// additionally, update the metrics-server deployment resource config to specify requests and limits using Kustomize
	kustomization := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: metrics-server
resources:
- manifest.yaml
patchesJson6902:
- patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/resources
      value:
        limits:
          cpu: 50m
          memory: 50Mi
        requests:
          cpu: 50m
          memory: 50Mi
  target:
    kind: Deployment
    name: metrics-server
`
	key := "metrics-server.deploy.test.kustomizations.deployment_resources"
	o, e = i.Set(key, kustomization, "yaml", false)
	assert.Equal(t, o, "")
	assert.Equal(t, e, "")

	// add a jsonnet config
	key = "jsonnet.deploy.test.jsonnet.test.inline"
	jsonnetInline := `{
	  Martini: {
	    local drink = self,
	    ingredients: [
	      { kind: "Farmer's Gin", qty: 1 },
	      {
	        kind: 'Dry White Vermouth',
	        qty: drink.ingredients[0].qty,
	      },
	    ],
	    garnish: 'Olive',
	    served: 'Straight Up',
	  },
	}
	`
	o, e = i.Set(key, jsonnetInline, "string", false)
	assert.Equal(t, o, "")
	assert.Equal(t, e, "")

	// generate again to apply the Kustomization
	o, e = i.Generate()
	assert.Equal(t, o, "")
	assert.Equal(t, e, "")

	// check jsonnet content
	actual := i.readFile("deploy/test/jsonnet/manifest.yaml")
	expected = `# Source: simple-ops jsonnet test
Martini:
  garnish: Olive
  ingredients:
  - kind: Farmer's Gin
    qty: 1
  - kind: Dry White Vermouth
    qty: 1
  served: Straight Up
`
	assert.Equal(t, string(actual), expected)

	// container resources yaml should now output the new config values
	o, e = i.ContainerResources("test.metrics-server", "yaml")
	expected = "- name: metrics-server\n  parentName: metrics-server\n  parentType: Deployment\n  resources:\n    limits:\n      cpu: 50m\n      memory: 50Mi\n    requests:\n      cpu: 50m\n      memory: 50Mi\n"
	assert.Equal(t, o, expected)
	assert.Equal(t, e, "")
}

type integration struct {
	in      *bytes.Buffer
	out     *bytes.Buffer
	err     *bytes.Buffer
	t       *testing.T
	workDir string
}

func newIntegration(t *testing.T) integration {
	return integration{
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
		t,
		"",
	}
}

func (i integration) readFile(path string) []byte {
	b, err := os.ReadFile(filepath.Join(i.workDir, path))
	if err != nil {
		i.t.Error(err)
	}
	return b
}

func (i integration) resetBuffers() {
	i.in.Reset()
	i.out.Reset()
	i.err.Reset()
}

func (i *integration) testSetup() {
	// set logger out to buffer
	log.SetOutput(i.out)
	// create tmp in memory working directory
	path, err := afero.TempDir(afero.NewOsFs(), "", "simple-ops-integration-")
	if err != nil {
		i.t.Fatal(err)
	}
	i.t.Logf("using tmp dir %s", path)
	// global used by newSvc functions ...
	i.workDir = path
	rootCmd.SetOut(i.out)
	rootCmd.SetIn(i.in)
	rootCmd.SetErr(i.err)
}

func (i integration) testTearDown(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		i.t.Fatal(err)
	}
	i.t.Logf("removed tmp dir %s", path)
}

func (i integration) runCmd(cmd *cobra.Command) (string, string) {
	if err := cmd.Execute(); err != nil {
		i.t.Error(err)
	}
	defer i.resetBuffers()
	return i.out.String(), i.err.String()
}

func (i integration) Init(force bool) (string, string) {
	defaultFlags()
	rootCmd.SetArgs([]string{"init", "-w", i.workDir})
	return i.runCmd(rootCmd)
}

func (i integration) Add(name string, repository string, version string, addConfig bool) (string, string) {
	defaultFlags()
	args := []string{"add", name, "--repo", repository, "--version", version, "-w", i.workDir}
	if addConfig {
		args = append(args, "--add-config")
	}
	rootCmd.SetArgs(args)
	return i.runCmd(rootCmd)
}

func (i integration) Set(key string, value string, as string, stdin bool) (string, string) {
	defaultFlags()
	args := []string{"set", "--type", as, key, "-w", i.workDir}
	if !stdin {
		args = append(args, value)
	} else {
		args = append(args, "--stdin")
	}
	rootCmd.SetArgs(args)
	return i.runCmd(rootCmd)
}

func (i integration) Generate() (string, string) {
	defaultFlags()
	rootCmd.SetArgs([]string{"generate", "-w", i.workDir})
	return i.runCmd(rootCmd)
}

func (i integration) ContainerResources(id string, outputType string) (string, string) {
	defaultFlags()
	args := []string{"container-resources", "--output", outputType, "-w", i.workDir}
	if id != "" {
		args = append(args, id)
	}
	rootCmd.SetArgs(args)
	return i.runCmd(rootCmd)
}

func (i integration) Images(id string, outputType string) (string, string) {
	defaultFlags()
	args := []string{"images", "--output", outputType, "-w", i.workDir}
	if id != "" {
		args = append(args, id)
	}
	rootCmd.SetArgs(args)
	return i.runCmd(rootCmd)
}

func (i integration) Show(thing string, id string) (string, string) {
	defaultFlags()
	rootCmd.SetArgs([]string{"show", thing, id, "-w", i.workDir})
	return i.runCmd(rootCmd)
}

func (i integration) Verify() (string, string) {
	defaultFlags()
	rootCmd.SetArgs([]string{"verify", "-w", i.workDir})
	return i.runCmd(rootCmd)
}

func (i integration) Version() (string, string) {
	defaultFlags()
	rootCmd.SetArgs([]string{"version", "-w", i.workDir})
	return i.runCmd(rootCmd)
}
