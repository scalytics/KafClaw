package cli

import (
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
