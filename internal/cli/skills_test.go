package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillsStatusCommand(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	out, err := runRootCommand(t, "skills", "status")
	if err != nil {
		t.Fatalf("skills status failed: %v", err)
	}
	if !strings.Contains(out, "Skills enabled:") || !strings.Contains(out, "Eligible skills:") {
		t.Fatalf("unexpected status output: %q", out)
	}
}

func TestSkillsListShowsReadinessColumns(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"skills":{"enabled":false}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	out, err := runRootCommand(t, "skills", "list")
	if err != nil {
		t.Fatalf("skills list failed: %v", err)
	}
	if !strings.Contains(out, "Columns: name | source | state | eligible | missing") {
		t.Fatalf("missing readiness columns in list output: %q", out)
	}
}

func TestFormatSkillError(t *testing.T) {
	err := formatSkillError("verify_failed", assertErr("boom"), "fix and retry")
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "[VERIFY_FAILED]") || !strings.Contains(msg, "remediation: fix and retry") {
		t.Fatalf("unexpected formatted error: %s", msg)
	}
}

func TestSkillsEnableDisableLifecycleConfigPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir cfg: %v", err)
	}
	origHome := os.Getenv("HOME")
	origPath := os.Getenv("PATH")
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("PATH", origPath)
	_ = os.Setenv("HOME", tmpDir)

	bin := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	for _, name := range []string{"node", "clawhub"} {
		script := filepath.Join(bin, name)
		if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	_ = os.Setenv("PATH", bin+string(os.PathListSeparator)+origPath)

	if _, err := runRootCommand(t, "skills", "enable", "--install-clawhub=false"); err != nil {
		t.Fatalf("skills enable failed: %v", err)
	}
	if _, err := runRootCommand(t, "skills", "disable"); err != nil {
		t.Fatalf("skills disable failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	skillCfg, ok := cfg["skills"].(map[string]any)
	if !ok {
		t.Fatalf("expected skills section in config")
	}
	enabled, _ := skillCfg["enabled"].(bool)
	if enabled {
		t.Fatalf("expected skills.enabled false after disable, got %#v", skillCfg["enabled"])
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
