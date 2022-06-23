package matcher

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

var ImageMatchers = []Matcher{
	{"CronJob", specJobTemplateSpec},
	{"Pod", specPodSpec},
	{"Deployment", specPodTemplateSpec},
	{"DaemonSet", specPodTemplateSpec},
	{"Job", specPodTemplateSpec},
	{"ReplicaSet", specPodTemplateSpec},
	{"ReplicationController", specPodTemplateSpec},
	{"StatefulSet", specPodTemplateSpec},
}

type Images []string

func (m Svc) Images(filePath string) ([]string, error) {
	var imgs []string
	matches, err := m.Match(filePath, ImageMatchers)
	if err != nil {
		return nil, err
	}
	for _, v := range matches {
		imgs = append(imgs, v.Value) //@todo
	}
	return imgs, nil
}
