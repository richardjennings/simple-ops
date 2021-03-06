package hash

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"path/filepath"
	"testing"
)

func TestSvc_SHA256File(t *testing.T) {
	expected := "39f51ddf0542074ffe55116b2b85ad0abc4a90e51a71347c20bd64b2b26b7bd6"
	c := NewSvc(afero.NewOsFs(), logrus.New())
	path, err := filepath.Abs("../manifest/testdata/test-0.1.0.tgz")
	assert.NilError(t, err)
	actual, err := c.SHA256File(path)
	assert.NilError(t, err)
	assert.Equal(t, actual, expected)
}

func TestSvc_SHA256(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), logrus.New())
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
	c := NewSvc(afero.NewMemMapFs(), logrus.New())
	sha, err := c.SHA256("/test")
	assert.Equal(t, sha, "")
	assert.Error(t, err, "path: /test not found")
}

func TestSvc_SHA256_compare(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), logrus.New())
	if err := c.AppFs.MkdirAll("test/case1/", 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case1/test.file", []byte("123"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.MkdirAll("test/case2/", 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case2/test.file", []byte("123"), 0755); err != nil {
		t.Fatal(err)
	}
	a, err := c.SHA256("test/case1")
	if err != nil {
		t.Error(err)
	}
	b, err := c.SHA256("test/case2")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, a, b)
}

func TestSvc_SHA256_compareMulti(t *testing.T) {
	c := NewSvc(afero.NewMemMapFs(), logrus.New())
	if err := c.AppFs.MkdirAll("test/case1/", 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case1/test.file", []byte("123"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case1/test2.file", []byte("456"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.MkdirAll("test/case2/", 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case2/test.file", []byte("123"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := c.AppFs.WriteFile("test/case2/test2.file", []byte("456"), 0755); err != nil {
		t.Fatal(err)
	}
	a, err := c.SHA256("test/case1")
	if err != nil {
		t.Error(err)
	}
	b, err := c.SHA256("test/case2")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, a, b)
}
