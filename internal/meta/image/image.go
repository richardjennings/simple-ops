package image

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"log"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var specPodSpec = [][]string{
	{"spec", "containers", "*", "image"},
	{"spec", "initContainers", "*", "image"},
	{"spec", "ephemeralContainers", "*", "image"},
}

var specPodTemplateSpec = [][]string{
	{"spec", "template", "spec", "containers", "*", "image"},
	{"spec", "template", "spec", "initContainers", "*", "image"},
	{"spec", "template", "spec", "ephemeralContainers", "*", "image"},
}

var specJobTemplateSpec = [][]string{
	{"spec", "jobTemplate", "spec", "template", "spec", "containers", "*", "image"},
	{"spec", "jobTemplate", "spec", "template", "spec", "initContainers", "*", "image"},
	{"spec", "jobTemplate", "spec", "template", "spec", "ephemeralContainers", "*", "image"},
}

var matchers = []struct {
	kind  string
	paths [][]string
}{
	{"CronJob", specJobTemplateSpec},
	{"Pod", specPodSpec},
	{"Deployment", specPodTemplateSpec},
	{"DaemonSet", specPodTemplateSpec},
	{"Job", specPodTemplateSpec},
	{"ReplicaSet", specPodTemplateSpec},
	{"ReplicationController", specPodTemplateSpec},
	{"StatefulSet", specPodTemplateSpec},
}

type (
	Svc struct {
		appFs afero.Afero
		wd    string
		log   *logrus.Logger
	}
)

func NewSvc(fs afero.Fs, wd string, log *logrus.Logger) *Svc {
	metas := &Svc{
		appFs: afero.Afero{Fs: fs},
		wd:    wd,
		log:   log,
	}
	return metas
}

// ListImages lists all Images found in resource kinds that support images in the
// manifest file at filePath
func (m *Svc) ListImages(filePath string) ([]string, error) {
	file, err := m.appFs.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	reader := kio.ByteReader{Reader: file}
	nodes, err := reader.Read()
	if err != nil {
		return nil, err
	}
	return m.images(nodes)
}

func (m Svc) images(nodes []*yaml.RNode) (images []string, err error) {
	for _, match := range matchers {
		kind := match.kind
		for _, path := range match.paths {
			matcher := yaml.PathMatcher{Path: path}
			for _, n := range nodes {
				if n.GetKind() == kind {
					_, err := matcher.Filter(n)
					if err != nil {
						return images, err
					}
					for m := range matcher.Matches {
						images = append(images, m.Value)
					}
				}
			}
		}
	}
	return images, err
}
