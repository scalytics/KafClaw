package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithIncludeAndEnvSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	basePath := filepath.Join(configDir, "base.json")
	mainPath := filepath.Join(configDir, "config.json")
	baseCfg := `{
		"model": { "name": "base-model", "maxTokens": 1024 },
		"gateway": { "host": "127.0.0.1", "port": 9000 }
	}`
	mainCfg := `{
		"$include": "base.json",
		"model": { "name": "${TEST_MODEL}" },
		"gateway": { "port": 7777 }
	}`
	if err := os.WriteFile(basePath, []byte(baseCfg), 0o600); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte(mainCfg), 0o600); err != nil {
		t.Fatalf("write main config: %v", err)
	}

	origHome := os.Getenv("HOME")
	origModel := os.Getenv("TEST_MODEL")
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("TEST_MODEL", origModel)
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Setenv("TEST_MODEL", "env-model")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Model.Name != "env-model" {
		t.Fatalf("expected env-substituted model name, got %q", cfg.Model.Name)
	}
	if cfg.Model.MaxTokens != 1024 {
		t.Fatalf("expected maxTokens from include file, got %d", cfg.Model.MaxTokens)
	}
	if cfg.Gateway.Port != 7777 {
		t.Fatalf("expected main config override for gateway.port, got %d", cfg.Gateway.Port)
	}
}

func TestLoadWithIncludeArrayMergeOrder(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	first := `{"model": {"name": "first", "maxTokens": 1000}}`
	second := `{"model": {"name": "second"}}`
	main := `{"$include": ["first.json", "second.json"], "model": {"temperature": 0.3}}`

	_ = os.WriteFile(filepath.Join(configDir, "first.json"), []byte(first), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "second.json"), []byte(second), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(main), 0o600)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Model.Name != "second" {
		t.Fatalf("expected second include to override first, got %q", cfg.Model.Name)
	}
	if cfg.Model.MaxTokens != 1000 {
		t.Fatalf("expected maxTokens preserved from first include, got %d", cfg.Model.MaxTokens)
	}
	if cfg.Model.Temperature != 0.3 {
		t.Fatalf("expected temperature from main config, got %v", cfg.Model.Temperature)
	}
}

func TestLoadWithInvalidIncludeTypeReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	main := `{"$include": 123}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(main), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid $include error, got nil")
	}
}

func TestLoadWithIncludeCycleReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	main := `{"$include": "a.json"}`
	a := `{"$include": "b.json"}`
	b := `{"$include": "a.json"}`
	_ = os.WriteFile(filepath.Join(configDir, "config.json"), []byte(main), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "a.json"), []byte(a), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "b.json"), []byte(b), 0o600)

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := Load(); err == nil {
		t.Fatal("expected include cycle error, got nil")
	}
}

func TestParseIncludes(t *testing.T) {
	got, err := parseIncludes("one.json")
	if err != nil || len(got) != 1 || got[0] != "one.json" {
		t.Fatalf("unexpected parse result: got=%v err=%v", got, err)
	}
	got, err = parseIncludes([]any{"one.json", "two.json"})
	if err != nil || len(got) != 2 {
		t.Fatalf("unexpected array parse: got=%v err=%v", got, err)
	}
	if _, err := parseIncludes([]any{"ok.json", 42}); err == nil {
		t.Fatal("expected parse error for non-string include item")
	}
}
