package testutil

import (
	"io/ioutil"
	"os"
	"testing"
)

// tempDir supplements an alternative to TB.TempDir()[1], which is only available in 1.15.
// [1]: https://github.com/golang/go/issues/35998
func TempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	})

	return dir
}
