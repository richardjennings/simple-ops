package show

import (
	"helm.sh/helm/v3/pkg/action"
)

func Show(chartPath string, t string) (string, error) {
	show := action.NewShowWithConfig(action.ShowOutputFormat(t), &action.Configuration{})
	return show.Run(chartPath)
}
