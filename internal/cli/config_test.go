package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runRootCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	_, err := rootCmd.ExecuteC()
	rootCmd.SetArgs(nil)
	return strings.TrimSpace(buf.String()), err
}

func TestConfigSetGetUnsetCommands(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"gateway":{"port":18790},"channels":{"telegram":{"allowFrom":[]}}}`), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t, "config", "set", "gateway.port", "18888"); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	out, err := runRootCommand(t, "config", "get", "gateway.port")
	if err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if out != "18888" {
		t.Fatalf("expected 18888, got %q", out)
	}

	if _, err := runRootCommand(t, "config", "set", "channels.telegram.allowFrom[0]", `"alice"`); err != nil {
		t.Fatalf("config set bracket path failed: %v", err)
	}
	out, err = runRootCommand(t, "config", "get", "channels.telegram.allowFrom[0]")
	if err != nil {
		t.Fatalf("config get bracket path failed: %v", err)
	}
	if out != "alice" {
		t.Fatalf("expected alice, got %q", out)
	}

	if _, err := runRootCommand(t, "config", "set", "custom.section.value", `"hello"`); err != nil {
		t.Fatalf("config set custom key failed: %v", err)
	}

	if _, err := runRootCommand(t, "config", "unset", "custom.section.value"); err != nil {
		t.Fatalf("config unset custom key failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "config.json"))
	if err != nil {
		t.Fatalf("read config after unset: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal config after unset: %v", err)
	}
	custom, ok := m["custom"].(map[string]any)
	if !ok {
		t.Fatalf("expected custom object to exist")
	}
	section, ok := custom["section"].(map[string]any)
	if !ok {
		t.Fatalf("expected custom.section object to exist")
	}
	if _, exists := section["value"]; exists {
		t.Fatal("expected custom.section.value removed from config file")
	}
}
