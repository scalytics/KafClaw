package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCSVList(t *testing.T) {
	got := parseCSVList(" agent-a,agent-b,agent-a,*, ")
	want := []string{"agent-a", "agent-b", "*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parse result: got=%v want=%v", got, want)
	}
}

func TestConfigureSkillsFlags(t *testing.T) {
	tmpDir := t.TempDir()
	cfgDir := filepath.Join(tmpDir, ".kafclaw")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"skills":{"enabled":false,"nodeManager":"npm","entries":{}}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t,
		"configure",
		"--non-interactive",
		"--skills-enabled-set",
		"--skills-enabled=true",
		"--skills-node-manager=pnpm",
		"--enable-skill=github",
		"--disable-skill=weather",
	); err != nil {
		t.Fatalf("configure skills flags failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(cfgDir, "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	skills, ok := cfg["skills"].(map[string]any)
	if !ok {
		t.Fatalf("missing skills section")
	}
	if enabled, _ := skills["enabled"].(bool); !enabled {
		t.Fatalf("expected skills enabled true, got %#v", skills["enabled"])
	}
	if nm, _ := skills["nodeManager"].(string); nm != "pnpm" {
		t.Fatalf("expected skills.nodeManager pnpm, got %q", nm)
	}
	entries, ok := skills["entries"].(map[string]any)
	if !ok {
		t.Fatalf("expected skills.entries object")
	}
	gh, ok := entries["github"].(map[string]any)
	if !ok {
		t.Fatalf("expected github entry override")
	}
	if v, _ := gh["enabled"].(bool); !v {
		t.Fatalf("expected github enabled override true")
	}
	weather, ok := entries["weather"].(map[string]any)
	if !ok {
		t.Fatalf("expected weather entry override")
	}
	if v, _ := weather["enabled"].(bool); v {
		t.Fatalf("expected weather enabled override false")
	}
}
