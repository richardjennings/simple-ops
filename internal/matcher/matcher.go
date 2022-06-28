package matcher

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"log"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type (
	Svc struct {
		appFs afero.Afero
		wd    string
		log   *logrus.Logger
	}
	Matcher struct {
		kind  string
		paths [][]string
	}
	Match struct {
		Resource *yaml.RNode
		Node     *yaml.RNode
		Value    string
	}
	Matches []Match
)

func NewSvc(fs afero.Fs, wd string, log *logrus.Logger) *Svc {
	return &Svc{
		appFs: afero.Afero{Fs: fs},
		wd:    wd,
		log:   log,
	}
}

// Match resources with configured paths
func (m Svc) Match(filePath string, matchers []Matcher) (Matches, error) {
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
	return m.match(nodes, matchers)
}

func (_ Svc) match(nodes []*yaml.RNode, matchers []Matcher) (Matches, error) {
	var matches Matches
	for _, match := range matchers {
		kind := match.kind
		for _, path := range match.paths {
			matcher := yaml.PathMatcher{Path: path}
			for _, n := range nodes {
				if n.GetKind() == kind {
					node, err := matcher.Filter(n)
					if err != nil {
						return matches, err
					}
					for k := range matcher.Matches {
						matches = append(matches, Match{
							Resource: n,
							Node:     node,
							Value:    k.Value,
						})
					}
				}
			}
		}
	}
	return matches, nil
}
