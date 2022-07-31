package manifest

import (
	"bytes"
	"fmt"
	"github.com/richardjennings/simple-ops/internal/cfg"
	"github.com/spf13/afero"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	lbs "sigs.k8s.io/kustomize/api/filters/labels"
	ns "sigs.k8s.io/kustomize/api/filters/namespace"

	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sort"
	"strings"
)

type chainFn func(deploy *cfg.Deploy, man *bytes.Buffer, crds *bytes.Buffer, s Svc) error

var actions = map[string]chainFn{
	"helm":      helm,
	"with":      with,
	"namespace": namespace,
	"labels":    labels,
	"kustomize": kustomize,
	"jsonnet":   jsonnetAction,
}

func kustomize(deploy *cfg.Deploy, man *bytes.Buffer, _ *bytes.Buffer, s Svc) error {
	if len(deploy.Kustomizations) == 0 {
		return nil
	}

	// only if buffer not empty, always reset after ...
	if man.Len() > 0 {
		if err := s.writeTmp(deploy, man); err != nil {
			return err
		}
		man.Reset()
	}

	// copy preserve paths
	if err := s.copyKustomizationPaths(deploy); err != nil {
		return err
	}
	return s.kustomizeDeploy(deploy)
}

func jsonnetAction(deploy *cfg.Deploy, man *bytes.Buffer, _ *bytes.Buffer, s Svc) error {
	if len(deploy.Jsonnet) == 0 {
		return nil
	}
	// only if buffer not empty, always reset after ...
	if man.Len() > 0 {
		if err := s.writeTmp(deploy, man); err != nil {
			return err
		}
		man.Reset()
	}
	man.Reset()
	return s.jsonnetDeploy(deploy, nil)
}

// Labels action templates namespace config
func labels(deploy *cfg.Deploy, man *bytes.Buffer, _ *bytes.Buffer, s Svc) error {
	buf := bytes.Buffer{}
	fslice := types.FsSlice{
		{Path: "metadata/labels", CreateIfNotPresent: true},
		{Path: "spec/template/metadata/labels", CreateIfNotPresent: false},
	}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: man}},
		Filters: []kio.Filter{lbs.Filter{Labels: deploy.Labels, FsSlice: fslice}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
	}.Execute()
	*man = buf
	return err
}

// Namespace action templates namespace config
func namespace(deploy *cfg.Deploy, man *bytes.Buffer, _ *bytes.Buffer, s Svc) error {
	buf := bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: man}},
		Filters: []kio.Filter{ns.Filter{Namespace: deploy.Namespace.Name, FsSlice: types.FsSlice{}}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
	}.Execute()
	*man = buf
	return err
}

// Helm action renders a helm chart
func helm(deploy *cfg.Deploy, man *bytes.Buffer, crds *bytes.Buffer, s Svc) error {
	client := action.NewInstall(&action.Configuration{})

	if deploy.Chart == "" {
		return nil
	}
	var chrt *chart.Chart
	var err error

	if strings.HasSuffix(deploy.Chart, ".tgz") {
		var f afero.File
		f, err = s.appFs.Open(s.PathForChart(deploy.Chart))
		if err != nil {
			return err
		}
		defer func() {
			_ = f.Close()
		}()
		chrt, err = loader.LoadArchive(f)
	} else {
		chrt, err = loader.Load(s.PathForChart(deploy.Chart))
	}
	if err != nil {
		return err
	}

	client.DryRun = true
	client.ClientOnly = true
	client.ReleaseName = chrt.Name()
	client.Namespace = deploy.Namespace.Name
	client.CreateNamespace = false
	client.IncludeCRDs = false
	client.SkipCRDs = true

	// render the helm chart
	rel, err := client.Run(chrt, deploy.Values)
	if err != nil {
		return err
	}
	man.Write([]byte(rel.Manifest))
	s.log.Debugf("rendered chart %s.%s for %s", chrt.Name(), chrt.Metadata.Version, deploy.Id())
	for _, f := range chrt.Files {
		if strings.HasPrefix(f.Name, "crds/") {
			if crds.Len() > 0 {
				crds.Write([]byte("---\n"))
			}
			s.log.Debugf("added crd %s for %s", f.Name, deploy.Id())
			crds.Write(f.Data)
		}
	}
	return nil
}

// With action adds with templates
func with(deploy *cfg.Deploy, man *bytes.Buffer, crds *bytes.Buffer, s Svc) error {
	var t []byte
	var err error

	// ordered with templates
	var orderedFiles []string
	for p := range deploy.With {
		orderedFiles = append(orderedFiles, p)
	}
	sort.Strings(orderedFiles)
	for _, p := range orderedFiles {
		withs, ok := deploy.With[p]
		if !ok {
			return fmt.Errorf("could not find with %s", p)
		}
		var ordered []string
		// iterate in-order such that the generated output
		// is idempotent
		for name := range withs {
			ordered = append(ordered, name)
		}
		sort.Strings(ordered)
		for _, name := range ordered {
			with, ok := withs[name]
			if !ok {
				return fmt.Errorf("could not find with %s", name)
			}
			if with.Path == "" {
				t, err = s.generateWith(p, with, name)
				if err != nil {
					return err
				}
				_, err = man.Write([]byte("---\n"))
				if err != nil {
					return err
				}
				_, err = man.Write([]byte(fmt.Sprintf("# Source: simple-ops with %s.yml\n", p)))
				if err != nil {
					return err
				}
				_, err = man.Write(t)
				if err != nil {
					return err
				}
				s.log.Debugf("generated with %s type %s for %s", name, p, deploy.Id())

			} else {
				if err := s.generateWithToPath(p, with, name); err != nil {
					return err
				}
				s.log.Debugf("generated with %s type %s for %s to path %s", name, p, deploy.Id(), with.Path)
			}
		}
	}
	return nil
}
