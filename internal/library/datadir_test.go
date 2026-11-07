package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDataDirectoryMovesLegacyFiles(t *testing.T) {
	for _, oldName := range []string{"PromptManager", "prompt-manager"} {
		t.Run(oldName, func(t *testing.T) {
			base := t.TempDir()
			old := filepath.Join(base, oldName)
			if err := os.MkdirAll(filepath.Join(old, "images"), 0700); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				"prompts.db":                         "existing database",
				"custom-image-models.json":           "existing settings",
				filepath.Join("images", "image.png"): "existing image",
			}
			for name, content := range files {
				if err := os.WriteFile(filepath.Join(old, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			target, err := resolveDataDirectory(base)
			if err != nil || target != filepath.Join(base, "prompt-next") {
				t.Fatalf("resolve: %q, %v", target, err)
			}
			for name, content := range files {
				data, err := os.ReadFile(filepath.Join(target, name))
				if err != nil || string(data) != content {
					t.Fatalf("%s: %q, %v", name, data, err)
				}
			}
			if _, err := os.Stat(old); !os.IsNotExist(err) {
				t.Fatalf("old directory still exists: %v", err)
			}
			if reopened, err := resolveDataDirectory(base); err != nil || reopened != target {
				t.Fatalf("reopen: %q, %v", reopened, err)
			}
		})
	}
}

func TestResolveDataDirectoryLeavesExistingTargetAlone(t *testing.T) {
	base := t.TempDir()
	for _, name := range []string{"PromptManager", "prompt-next"} {
		dir := filepath.Join(base, name)
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "prompts.db"), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	target, err := resolveDataDirectory(base)
	if err != nil || target != filepath.Join(base, "prompt-next") {
		t.Fatalf("resolve: %q, %v", target, err)
	}
	for _, name := range []string{"PromptManager", "prompt-next"} {
		data, err := os.ReadFile(filepath.Join(base, name, "prompts.db"))
		if err != nil || string(data) != name {
			t.Fatalf("%s changed: %q, %v", name, data, err)
		}
	}
}
