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
	ContainerResourceResult struct {
		ParentName string            `json:"parentName"`
		ParentType string            `json:"parentType"`
		Name       string            `json:"name"`
		Resource   ContainerResource `json:"resources,omitempty"`
	}
	Conf struct {
		Memory string `json:"memory,omitempty"`
		CPU    string `json:"cpu,omitempty"`
	}
	ContainerResource struct {
		Limits   Conf `json:"limits"`
		Requests Conf `json:"requests"`
	}
	ContainerResources []ContainerResourceResult
)

// ContainerResources lists resources configuration in the manifest at filePath
func (m Svc) ContainerResources(filePath string) (ContainerResources, error) {
	var result ContainerResources

	matches, err := m.Match(filePath, ResourceMatchers)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		res := ContainerResourceResult{
			ParentType: match.Resource.GetKind(),
			ParentName: match.Resource.GetName(),
		}
		var resource ContainerResource
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
				res.Resource = resource
			}
			return nil
		}); err != nil {
			return nil, err
		}
		result = append(result, res)
	}
	return result, nil
}
