package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnboardNonInteractiveRequiresAcceptRisk(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	_, err := runRootCommand(t, "onboard", "--non-interactive", "--skip-skills")
	if err == nil {
		t.Fatal("expected onboard to fail without --accept-risk in non-interactive mode")
	}
}

func TestOnboardJSONSummary(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	out, err := runRootCommand(t, "onboard", "--non-interactive", "--accept-risk", "--skip-skills", "--json")
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if !strings.Contains(out, "\"configPath\"") || !strings.Contains(out, "\"skills\"") {
		t.Fatalf("expected onboarding JSON summary in output, got %q", out)
	}
}

func TestOnboardSkillsBootstrapPath(t *testing.T) {
	tmpDir := t.TempDir()
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

	out, err := runRootCommand(t, "onboard", "--non-interactive", "--accept-risk", "--skip-skills=false", "--json")
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if !strings.Contains(out, "\"enabled\": true") && !strings.Contains(out, "\"enabled\":true") {
		t.Fatalf("expected skills enabled in onboarding output, got %q", out)
	}
}

func TestOnboardConfiguresOAuthCapabilities(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t,
		"onboard",
		"--non-interactive",
		"--accept-risk",
		"--skip-skills",
		"--google-workspace-read=mail,drive",
		"--m365-read=calendar",
	); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".kafclaw", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	skillsCfg, ok := cfg["skills"].(map[string]any)
	if !ok {
		t.Fatalf("missing skills config")
	}
	entries, ok := skillsCfg["entries"].(map[string]any)
	if !ok {
		t.Fatalf("missing skills.entries")
	}
	if _, ok := entries["google-workspace"].(map[string]any); !ok {
		t.Fatalf("expected google-workspace entry, got %#v", entries["google-workspace"])
	}
	if _, ok := entries["m365"].(map[string]any); !ok {
		t.Fatalf("expected m365 entry, got %#v", entries["m365"])
	}
}
