package show

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShow_ValidChartPath(t *testing.T) {
	chartPath := "../manifest/testdata/test-0.1.0.tgz"
	actual, err := Show(chartPath, "values")
	assert.NotEmpty(t, actual)
	assert.Nil(t, err)
}

func TestShow_InvalidChartPath(t *testing.T) {
	chartPath := "../manifest/testdata/test-0.1.0.tgz.wrong"
	actual, err := Show(chartPath, "values")
	assert.Empty(t, actual)
	assert.NotNil(t, err)
}
