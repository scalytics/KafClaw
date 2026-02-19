package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KafClaw/KafClaw/internal/config"
)

func TestPreflightUpdateCompatibility(t *testing.T) {
	if err := preflightUpdateCompatibility("2.6.3", "v2.7.0", false); err != nil {
		t.Fatalf("expected compatible version, got %v", err)
	}
	if err := preflightUpdateCompatibility("2.6.3", "v4.0.0", false); err == nil {
		t.Fatal("expected major jump failure")
	}
	if err := preflightUpdateCompatibility("2.6.3", "v2.5.9", false); err == nil {
		t.Fatal("expected downgrade failure without allow flag")
	}
	if err := preflightUpdateCompatibility("2.6.3", "v2.5.9", true); err != nil {
		t.Fatalf("expected allowed downgrade, got %v", err)
	}
}

func TestCreateAndRestoreUpdateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	origKHome := os.Getenv("KAFCLAW_HOME")
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("KAFCLAW_HOME", origKHome)
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Setenv("KAFCLAW_HOME", tmpDir)

	cfgPath, err := config.ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("mkdir cfg dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(`{"gateway":{"port":18790}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	envPath := filepath.Join(tmpDir, ".config", "kafclaw", "env")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o700); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("A=1\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	dbPath := filepath.Join(tmpDir, ".kafclaw", "timeline.db")
	if err := os.WriteFile(dbPath, []byte("db"), 0o600); err != nil {
		t.Fatalf("write db: %v", err)
	}

	origNow := nowFn
	nowFn = func() time.Time { return time.Date(2026, 2, 19, 12, 0, 0, 0, time.UTC) }
	defer func() { nowFn = origNow }()

	backupRoot := filepath.Join(tmpDir, "backups")
	snapshot, _, err := createUpdateBackup(backupRoot)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(snapshot, "manifest.json")); err != nil {
		t.Fatalf("missing manifest: %v", err)
	}

	if err := os.WriteFile(cfgPath, []byte(`{"gateway":{"port":19999}}`), 0o600); err != nil {
		t.Fatalf("mutate config: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("A=2\n"), 0o600); err != nil {
		t.Fatalf("mutate env: %v", err)
	}

	if _, err := restoreUpdateBackup(snapshot, false); err != nil {
		t.Fatalf("restore backup: %v", err)
	}

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(cfgData), "18790") {
		t.Fatalf("expected config restored, got %s", string(cfgData))
	}
	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if strings.TrimSpace(string(envData)) != "A=1" {
		t.Fatalf("expected env restored, got %s", string(envData))
	}
}

func TestDetectConfigDrift(t *testing.T) {
	before := map[string]any{
		"gateway": map[string]any{
			"port": float64(18790),
			"host": "127.0.0.1",
		},
	}
	after := map[string]any{
		"gateway": map[string]any{
			"port": float64(19990),
		},
		"group": map[string]any{
			"enabled": true,
		},
	}
	drift := detectConfigDrift(before, after)
	if len(drift) < 3 {
		t.Fatalf("expected at least 3 drift entries, got %v", drift)
	}
}
