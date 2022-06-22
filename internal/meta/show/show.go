package show

import (
	"errors"
	"helm.sh/helm/v3/pkg/action"
)

type Type string

func (o *Type) String() string {
	return string(*o)
}
func (o *Type) Set(v string) error {
	switch action.ShowOutputFormat(v) {
	case action.ShowAll, action.ShowValues, action.ShowChart, action.ShowCRDs, action.ShowReadme:
		*o = Type(v)
	default:
		return errors.New("supported output types are [yaml, json]")
	}
	return nil
}
func (o *Type) Type() string {
	return "ShowType"
}

func Show(chartPath string, t Type) (string, error) {
	show := action.NewShowWithConfig(action.ShowOutputFormat(t.String()), &action.Configuration{})
	return show.Run(chartPath)
}
