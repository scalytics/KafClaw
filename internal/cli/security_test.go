package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityCheckCommand(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	out, err := runRootCommand(t, "security", "check")
	if err != nil && !strings.Contains(out, "[FAIL]") {
		t.Fatalf("security check failed unexpectedly: %v", err)
	}
	if !strings.Contains(out, "doctor:") {
		t.Fatalf("expected doctor-derived checks in output, got %q", out)
	}
}

func TestSecurityFixRequiresYes(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	_, err := runRootCommand(t, "security", "fix")
	if err == nil {
		t.Fatal("expected security fix to require --yes")
	}
}

func TestSecurityFixAppliesSkillPolicyDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	cfg := `{
	  "skills": {
	    "enabled": true,
	    "linkPolicy": {
	      "mode": "allowlist",
	      "allowDomains": [],
	      "allowHttp": true,
	      "maxLinksPerSkill": 0
	    }
	  }
	}`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	_, _ = runRootCommand(t, "security", "fix", "--yes")

	data, err := os.ReadFile(filepath.Join(cfgDir, "config.json"))
	if err != nil {
		t.Fatalf("read config after fix: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"allowHttp": false`) && !strings.Contains(text, `"allowHttp":false`) {
		t.Fatalf("expected allowHttp=false after fix, got: %s", text)
	}
	if !strings.Contains(text, "clawhub.ai") {
		t.Fatalf("expected default allow domain after fix, got: %s", text)
	}
}
