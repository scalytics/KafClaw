package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPathRespectsKafclawConfigAndHome(t *testing.T) {
	origCfg := os.Getenv("KAFCLAW_CONFIG")
	origHome := os.Getenv("KAFCLAW_HOME")
	defer os.Setenv("KAFCLAW_CONFIG", origCfg)
	defer os.Setenv("KAFCLAW_HOME", origHome)

	_ = os.Setenv("KAFCLAW_HOME", "/srv/kafhome")
	_ = os.Setenv("KAFCLAW_CONFIG", "~/.kafclaw/custom.json")

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if path != filepath.Join("/srv/kafhome", ".kafclaw", "custom.json") {
		t.Fatalf("unexpected config path: %q", path)
	}
}

func TestLoadUsesEnvFileCandidateForKafclawPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	envDir := filepath.Join(tmpDir, ".config", "kafclaw")
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	envPath := filepath.Join(envDir, "env")
	if err := os.WriteFile(envPath, []byte("KAFCLAW_GATEWAY_PORT=19999\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	origHome := os.Getenv("HOME")
	origPort := os.Getenv("KAFCLAW_GATEWAY_PORT")
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("KAFCLAW_GATEWAY_PORT", origPort)
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Unsetenv("KAFCLAW_GATEWAY_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Gateway.Port != 19999 {
		t.Fatalf("expected gateway port from env file, got %d", cfg.Gateway.Port)
	}
}
