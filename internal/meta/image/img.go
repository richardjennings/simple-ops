package image

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"log"
	"os"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

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

func (m *Svc) ListImages(path string) ([]string, error) {
	var images []string
	file, err := os.Open(path)
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
	// kind: Deployment
	images = append(images, m.deploymentImages(nodes)...)
	// kind: DaemonSet
	images = append(images, m.daemonSetImages(nodes)...)
	// kind: Job
	images = append(images, m.jobImages(nodes)...)
	// kind: ReplicaSet
	images = append(images, m.replicaSetImages(nodes)...)
	// kind: ReplicationController
	images = append(images, m.replicationControllerImages(nodes)...)
	// kind: StatefulSet
	images = append(images, m.statefulSetImages(nodes)...)

	return images, nil
}

func (m Svc) statefulSetImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "StatefulSet")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "StatefulSet")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "StatefulSet")...)
	return images
}

func (m Svc) replicationControllerImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicationController")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicationController")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicationController")...)
	return images
}

func (m Svc) replicaSetImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicaSet")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicaSet")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "ReplicaSet")...)
	return images
}

func (m Svc) jobImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Job")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Job")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Job")...)
	return images
}

func (m Svc) daemonSetImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "DaemonSet")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "DaemonSet")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "DaemonSet")...)
	return images
}

func (m Svc) deploymentImages(nodes []*yaml.RNode) (images []string) {
	// containers
	matcher := yaml.PathMatcher{Path: []string{"spec", "template", "spec", "containers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Deployment")...)
	// initContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "initContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Deployment")...)
	// ephemeralContainers
	matcher = yaml.PathMatcher{Path: []string{"spec", "template", "spec", "ephemeralContainers", "*", "image"}}
	images = append(images, match(&matcher, nodes, "Deployment")...)

	return images
}

func match(matcher *yaml.PathMatcher, nodes []*yaml.RNode, kind string) []string {
	var images []string
	for _, n := range nodes {
		if n.GetKind() == kind {
			_, err := matcher.Filter(n)
			if err != nil {
				log.Fatal(err)
			}
			for m, _ := range matcher.Matches {
				images = append(images, m.Value)
			}
		}
	}
	return images
}