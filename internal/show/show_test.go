package show

import (
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"testing"
)

func TestType_Set_Invalid(t *testing.T) {
	var setType Type
	err := setType.Set("notok")
	assert.NotNil(t, err)
}

func TestType_Set_Valid(t *testing.T) {
	var setType Type
	err := setType.Set(string(action.ShowValues))
	assert.Nil(t, err)
	assert.Equal(t, setType.Type(), "ShowType")
}

func TestShow_ValidChartPath(t *testing.T) {
	var setType Type
	_ = setType.Set("values")
	chartPath := "../manifest/testdata/test-0.1.0.tgz"
	actual, err := Show(chartPath, setType)
	assert.NotEmpty(t, actual)
	assert.Nil(t, err)
}

func TestShow_InvalidChartPath(t *testing.T) {
	var setType Type
	_ = setType.Set("values")
	chartPath := "../manifest/testdata/test-0.1.0.tgz.wrong"
	actual, err := Show(chartPath, setType)
	assert.Empty(t, actual)
	assert.NotNil(t, err)
}
