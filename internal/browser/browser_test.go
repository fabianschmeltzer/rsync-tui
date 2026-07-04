package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDirectories(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"visible", ".hidden"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := LocalDirectories(root, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name == ".hidden" {
			t.Fatal("hidden directory was visible")
		}
	}
	entries, err = LocalDirectories(root, true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		found = found || entry.Name == ".hidden"
	}
	if !found {
		t.Fatal("hidden directory was not returned")
	}
}
