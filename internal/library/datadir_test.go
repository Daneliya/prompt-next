package library

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

func TestSwitchDataDirectoryValidatesAndRollsBack(t *testing.T) {
	base := t.TempDir()
	source, err := openWithLocation(base)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(source)
	original, err := source.SavePrompt(Prompt{Kind: "text", Title: "原目录", Body: "旧库内容"})
	if err != nil {
		t.Fatal(err)
	}
	unrelated := t.TempDir()
	if err := os.WriteFile(filepath.Join(unrelated, "notes.txt"), []byte("personal"), 0600); err != nil {
		t.Fatal(err)
	}
	launch := func() error { return nil }
	if changed, err := switchDataDirectory(source, base, unrelated, launch); changed || err == nil {
		t.Fatalf("unrelated folder was accepted: %v, %v", changed, err)
	}
	target := t.TempDir()
	if changed, err := switchDataDirectory(source, base, target, func() error {
		return os.ErrPermission
	}); changed || err == nil {
		t.Fatalf("failed launch unexpectedly switched: %v, %v", changed, err)
	}
	if _, err := os.Stat(filepath.Join(base, locationFile)); !os.IsNotExist(err) {
		t.Fatalf("failed launch left a location override: %v", err)
	}
	if changed, err := switchDataDirectory(source, base, target, launch); !changed || err != nil {
		t.Fatalf("switch failed: %v, %v", changed, err)
	}
	reopened, err := openWithLocation(base)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(reopened)
	same, err := os.Stat(DataDirectory(reopened))
	wanted, targetErr := os.Stat(target)
	if err != nil || targetErr != nil || !os.SameFile(same, wanted) {
		t.Fatalf("reopened wrong directory: %s", DataDirectory(reopened))
	}
	items, err := reopened.ListPrompts()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.ID == original.ID {
			t.Fatal("opening a new directory copied data from the old library")
		}
	}
	another := t.TempDir()
	if changed, err := switchDataDirectory(reopened, base, another, launch); !changed || err != nil {
		t.Fatalf("could not switch from one custom folder to another: %v, %v", changed, err)
	}
	second, err := openWithLocation(base)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(second)
	if !sameDirectory(DataDirectory(second), another) {
		t.Fatalf("did not open second custom folder: %s", DataDirectory(second))
	}
	if changed, err := resetDataDirectory(base, func() error { return os.ErrPermission }); changed || err == nil {
		t.Fatalf("failed launch unexpectedly reset the location: %v, %v", changed, err)
	}
	saved, err := os.ReadFile(filepath.Join(base, locationFile))
	var selected dataLocation
	if err != nil || json.Unmarshal(saved, &selected) != nil || !sameDirectory(selected.Directory, another) {
		t.Fatalf("failed reset did not restore the selected folder: %s, %v", saved, err)
	}
	if changed, err := resetDataDirectory(base, launch); !changed || err != nil {
		t.Fatalf("could not reset to default: %v, %v", changed, err)
	}
	if _, err := os.Stat(filepath.Join(base, locationFile)); !os.IsNotExist(err) {
		t.Fatalf("reset left the location override in place: %v", err)
	}
	reset, err := openWithLocation(base)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(reset)
	previous, err := reset.ListPrompts()
	if err != nil || !sameDirectory(DataDirectory(reset), DataDirectory(source)) ||
		!slices.ContainsFunc(previous, func(p Prompt) bool { return p.ID == original.ID && p.Body == original.Body }) {
		t.Fatalf("original library was not restored: %s, %v", DataDirectory(reset), err)
	}
}

func sameDirectory(a, b string) bool {
	left, leftErr := os.Stat(a)
	right, rightErr := os.Stat(b)
	return leftErr == nil && rightErr == nil && os.SameFile(left, right)
}

func TestUnavailableSelectedDirectoryFallsBack(t *testing.T) {
	base := t.TempDir()
	missing := filepath.Join(base, "missing")
	raw, err := json.Marshal(dataLocation{Directory: missing})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, locationFile)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	service, err := openWithLocation(base)
	if err != nil {
		t.Fatal(err)
	}
	defer Close(service)
	if service.locationWarning == "" || DataDirectory(service) != filepath.Join(base, "prompt-next") {
		t.Fatalf("fallback: %s, %q", DataDirectory(service), service.locationWarning)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("recreated missing data directory: %v", err)
	}
	if current, err := os.ReadFile(path); err != nil || string(current) != string(raw) {
		t.Fatalf("fallback changed the preference: %s, %v", current, err)
	}
}
