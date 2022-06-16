package compare

import (
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"testing"
)

func TestSvc_SHA256(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs())
	if err := c.AppFs.MkdirAll("test/case/", 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case/test.file", []byte("123"), 0755); err != nil {
		t.Fatal(err)
	}
	sha, err := c.SHA256("test")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, sha, "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3")
}

func TestSvc_SHA256_invalid_path(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs())
	sha, err := c.SHA256("/test")
	assert.Equal(t, sha, "")
	assert.Error(t, err, "path: /test not found")
}
