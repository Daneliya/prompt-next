package fileio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicReplacesFileInDestination(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "export destination")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "backup.json")
	for _, data := range []string{"first", "replacement"} {
		if err := WriteAtomic(path, []byte(data)); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(path); err != nil || string(got) != data {
			t.Fatalf("destination contains %q, %v", got, err)
		}
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 1 {
		t.Fatalf("destination has extra temporary files: %v, %v", entries, err)
	}
}
