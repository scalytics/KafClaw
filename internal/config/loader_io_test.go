package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.Model.Name = "saved-model"
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved config file missing: %v", err)
	}

	newDir := filepath.Join(tmpDir, "nested", "dir")
	if err := EnsureDir(newDir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}
	if info, err := os.Stat(newDir); err != nil || !info.IsDir() {
		t.Fatalf("expected created directory, err=%v", err)
	}
}

func TestLoadInvalidJSONReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"model":`), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := Load(); err == nil {
		t.Fatal("expected JSON error, got nil")
	}
}

func TestSubstituteEnvValuesLeavesUnknownToken(t *testing.T) {
	input := map[string]any{
		"value": "${NOT_SET_VAR}",
	}
	out := substituteEnvValues(input).(map[string]any)
	if out["value"] != "${NOT_SET_VAR}" {
		t.Fatalf("expected unknown env token unchanged, got %v", out["value"])
	}
}
