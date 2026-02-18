package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/KafClaw/KafClaw/internal/skills"
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

func TestResolveOAuthScopesFromAccessSelection(t *testing.T) {
	cfg := config.DefaultConfig()
	scopes, err := resolveOAuthScopes(cfg, skills.ProviderGoogleWorkspace, "", "mail,calendar")
	if err != nil {
		t.Fatalf("resolve google scopes: %v", err)
	}
	if !containsAll(scopes, []string{
		"openid",
		"email",
		"profile",
		"https://www.googleapis.com/auth/gmail.readonly",
		"https://www.googleapis.com/auth/calendar.readonly",
	}) {
		t.Fatalf("unexpected google scopes: %#v", scopes)
	}

	mScopes, err := resolveOAuthScopes(cfg, skills.ProviderM365, "", "all")
	if err != nil {
		t.Fatalf("resolve m365 scopes: %v", err)
	}
	if !containsAll(mScopes, []string{"Mail.Read", "Calendars.Read", "Files.Read"}) {
		t.Fatalf("unexpected m365 scopes: %#v", mScopes)
	}
}

func TestResolveOAuthScopesFromConfiguredCapabilities(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Skills.Entries = map[string]config.SkillEntryConfig{
		"google-workspace": {Enabled: true, Capabilities: []string{"drive"}},
	}
	scopes, err := resolveOAuthScopes(cfg, skills.ProviderGoogleWorkspace, "", "")
	if err != nil {
		t.Fatalf("resolve scopes from config: %v", err)
	}
	if !reflect.DeepEqual(scopes, []string{
		"openid",
		"email",
		"profile",
		"https://www.googleapis.com/auth/drive.readonly",
	}) {
		t.Fatalf("unexpected scopes from config: %#v", scopes)
	}
}

func containsAll(have []string, want []string) bool {
	set := map[string]struct{}{}
	for _, s := range have {
		set[s] = struct{}{}
	}
	for _, s := range want {
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
