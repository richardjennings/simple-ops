package matcher

import (
	"gotest.tools/assert"
	"testing"
)

func TestImages_Unique(t *testing.T) {
	images := Images{
		"c",
		"a",
		"b",
		"a",
		"b",
		"b",
		"d",
	}
	assert.DeepEqual(t, images.Unique(), Images{"c", "a", "b", "d"})
}
