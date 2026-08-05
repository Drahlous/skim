package keybindings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaults(t *testing.T) {
	km := Defaults()

	if len(km) != len(Registry) {
		t.Fatalf("got %d actions, want %d (one per registry entry)", len(km), len(Registry))
	}

	for _, spec := range Registry {
		got, ok := km[spec.Action]
		if !ok {
			t.Errorf("Defaults() missing action %q", spec.Action)
			continue
		}
		if !reflect.DeepEqual(got, spec.DefaultKeys) {
			t.Errorf("Defaults()[%q] = %v, want %v", spec.Action, got, spec.DefaultKeys)
		}
	}
}

func TestDefaultsReturnsIndependentCopies(t *testing.T) {
	km := Defaults()
	km[Quit][0] = "mutated"

	fresh := Defaults()
	if fresh[Quit][0] == "mutated" {
		t.Fatal("mutating a Defaults() result affected a subsequent call; DefaultKeys slices are shared, not copied")
	}
}

func TestSpecFor(t *testing.T) {
	spec, ok := SpecFor(Quit)
	if !ok {
		t.Fatal("SpecFor(Quit) returned ok=false")
	}
	if spec.Action != Quit {
		t.Errorf("SpecFor(Quit).Action = %q, want %q", spec.Action, Quit)
	}

	if _, ok := SpecFor(Action("not_a_real_action")); ok {
		t.Error("SpecFor with an unknown action returned ok=true")
	}
}

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	km, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(km, Defaults()) {
		t.Errorf("Load() with no config file = %v, want defaults %v", km, Defaults())
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	km := Defaults()
	km[EditRegex] = []string{"e"}

	if err := Save(km); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(loaded[EditRegex], []string{"e"}) {
		t.Errorf("Load() after Save() got EditRegex=%v, want [e]", loaded[EditRegex])
	}
}

func TestLoadMergesPartialConfigWithDefaults(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	skimDir := filepath.Join(configDir, "skim")
	if err := os.MkdirAll(skimDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skimDir, "keybindings.json"), []byte(`{"edit_regex": ["e"]}`), 0o644); err != nil {
		t.Fatalf("failed to write partial config: %v", err)
	}

	km, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if !reflect.DeepEqual(km[EditRegex], []string{"e"}) {
		t.Errorf("Load() got EditRegex=%v, want [e] (from config file)", km[EditRegex])
	}
	if !reflect.DeepEqual(km[Quit], Defaults()[Quit]) {
		t.Errorf("Load() got Quit=%v, want default %v (untouched by partial config)", km[Quit], Defaults()[Quit])
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	skimDir := filepath.Join(configDir, "skim")
	if err := os.MkdirAll(skimDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skimDir, "keybindings.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() with invalid JSON returned no error")
	}
}

func TestSaveCreatesConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	if err := Save(Defaults()); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	path := filepath.Join(configDir, "skim", "keybindings.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file at %s: %v", path, err)
	}
}

// unsetConfigDirEnv clears both env vars os.UserConfigDir consults on Unix,
// forcing it to return an error, so configPath()'s error path can be tested.
func unsetConfigDirEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
}

func TestLoadReturnsDefaultsWhenConfigPathUnresolvable(t *testing.T) {
	unsetConfigDirEnv(t)

	km, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(km, Defaults()) {
		t.Errorf("Load() with unresolvable config path = %v, want defaults %v", km, Defaults())
	}
}

func TestSaveErrorsWhenConfigPathUnresolvable(t *testing.T) {
	unsetConfigDirEnv(t)

	if err := Save(Defaults()); err == nil {
		t.Fatal("Save() with unresolvable config path returned no error")
	}
}

func TestLoadErrorsWhenConfigFileUnreadable(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Make keybindings.json a directory instead of a file, so os.ReadFile
	// fails with an error other than os.IsNotExist (e.g. EISDIR), exercising
	// Load's non-missing-file error path.
	skimDir := filepath.Join(configDir, "skim")
	if err := os.MkdirAll(filepath.Join(skimDir, "keybindings.json"), 0o755); err != nil {
		t.Fatalf("failed to create directory standing in for the config file: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() with config path pointing at a directory returned no error")
	}
}

func TestSaveErrorsWhenConfigDirCannotBeCreated(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Create a regular file where Save needs to mkdir a "skim" directory, so
	// os.MkdirAll fails, exercising Save's directory-creation error path.
	if err := os.WriteFile(filepath.Join(configDir, "skim"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("failed to write blocking file: %v", err)
	}

	if err := Save(Defaults()); err == nil {
		t.Fatal("Save() with a blocked config dir returned no error")
	}
}
