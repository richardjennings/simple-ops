package matcher

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"strings"
)

var resourcePodSpec = [][]string{
	{"spec", "containers", "*"},
	{"spec", "initContainers", "*"},
	{"spec", "ephemeralContainers", "*"},
}

var resourcePodTemplateSpec = [][]string{
	{"spec", "template", "spec", "containers", "*"},
	{"spec", "template", "spec", "initContainers", "*"},
	{"spec", "template", "spec", "ephemeralContainers", "*"},
}

var resourceJobTemplateSpec = [][]string{
	{"spec", "jobTemplate", "spec", "template", "spec", "containers", "*"},
	{"spec", "jobTemplate", "spec", "template", "spec", "initContainers", "*"},
	{"spec", "jobTemplate", "spec", "template", "spec", "ephemeralContainers", "*"},
}

var ResourceMatchers = []Matcher{
	{"CronJob", resourceJobTemplateSpec},
	{"Pod", resourcePodSpec},
	{"Deployment", resourcePodTemplateSpec},
	{"DaemonSet", resourcePodTemplateSpec},
	{"Job", resourcePodTemplateSpec},
	{"ReplicaSet", resourcePodTemplateSpec},
	{"ReplicationController", resourcePodTemplateSpec},
	{"StatefulSet", resourcePodTemplateSpec},
}

type (
	ResourceResult struct {
		ParentName string
		ParentType string
		Name       string
		Resource   *Resource `json:",omitempty"`
	}
	Conf struct {
		Memory string `json:",omitempty"`
		CPU    string `json:",omitempty"`
	}
	Resource struct {
		Limits   Conf
		Requests Conf
	}
	Resources []ResourceResult
)

// Resources lists resources configuration in the manifest at filePath
func (m Svc) Resources(filePath string) (Resources, error) {
	var result Resources

	matches, err := m.Match(filePath, ResourceMatchers)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		res := ResourceResult{
			ParentType: match.Resource.GetKind(),
			ParentName: match.Resource.GetName(),
		}
		var resource Resource
		if err := match.Node.VisitElements(func(node *yaml.RNode) error {
			if name := node.Field("name"); name != nil {
				n, err := name.Value.String()
				if err != nil {
					return err
				}
				res.Name = strings.TrimSpace(n)
			}
			// mapping node
			if r := node.Field("resources"); r != nil {
				if err = r.Value.YNode().Decode(&resource); err != nil {
					return err
				}
				res.Resource = &resource
			}
			return nil
		}); err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}
