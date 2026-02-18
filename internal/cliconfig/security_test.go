package cliconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSecurityAuditDeepNoInstalledSkills(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"skills":{"enabled":true}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	report, err := RunSecurityAudit(SecurityAuditOptions{Deep: true})
	if err != nil {
		t.Fatalf("run security audit: %v", err)
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "skills_verify_installed" {
			found = true
			if !strings.Contains(c.Message, "no installed skills") {
				t.Fatalf("unexpected deep audit message: %#v", c)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected skills_verify_installed check, got %#v", report.Checks)
	}
}

func TestRunSecurityCheckReportsHashPinningGap(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"skills":{"enabled":true}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	installed := filepath.Join(cfgDir, "skills", "installed")
	if err := os.MkdirAll(filepath.Join(installed, "demo"), 0o700); err != nil {
		t.Fatalf("mkdir installed skill: %v", err)
	}
	meta := `{"name":"demo","source":"https://example.org/demo.zip","sourceHash":"abc","pinnedHash":""}`
	if err := os.WriteFile(filepath.Join(installed, "demo", ".kafclaw-skill.json"), []byte(meta), 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	report, err := RunSecurityCheck()
	if err != nil {
		t.Fatalf("run security check: %v", err)
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "gap_skill_hash_pinning" {
			found = true
			if c.Status != SecurityWarn {
				t.Fatalf("expected hash pinning warning, got %#v", c)
			}
		}
	}
	if !found {
		t.Fatalf("expected gap_skill_hash_pinning check, got %#v", report.Checks)
	}
}
