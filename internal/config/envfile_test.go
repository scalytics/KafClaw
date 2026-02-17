package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileParsesAndRespectsExistingValues(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, "env")
	content := `
# comment
export FOO=bar
QUOTED="hello world"
SINGLE='x y'
INVALID_LINE
`
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	origFoo := os.Getenv("FOO")
	origQuoted := os.Getenv("QUOTED")
	defer os.Setenv("FOO", origFoo)
	defer os.Setenv("QUOTED", origQuoted)

	_ = os.Setenv("FOO", "existing")
	_ = os.Unsetenv("QUOTED")
	_ = os.Unsetenv("SINGLE")

	if err := loadEnvFile(envPath); err != nil {
		t.Fatalf("load env file: %v", err)
	}

	if got := os.Getenv("FOO"); got != "existing" {
		t.Fatalf("expected existing FOO preserved, got %q", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello world" {
		t.Fatalf("expected QUOTED loaded, got %q", got)
	}
	if got := os.Getenv("SINGLE"); got != "x y" {
		t.Fatalf("expected SINGLE loaded, got %q", got)
	}
}

func TestLoadEnvFileCandidatesFromExplicitPath(t *testing.T) {
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, "kafclaw.env")
	if err := os.WriteFile(envPath, []byte("EXPLICIT_KEY=42\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	origExplicit := os.Getenv("KAFCLAW_ENV_FILE")
	origKey := os.Getenv("EXPLICIT_KEY")
	defer os.Setenv("KAFCLAW_ENV_FILE", origExplicit)
	defer os.Setenv("EXPLICIT_KEY", origKey)
	_ = os.Setenv("KAFCLAW_ENV_FILE", envPath)
	_ = os.Unsetenv("EXPLICIT_KEY")

	LoadEnvFileCandidates()

	if got := os.Getenv("EXPLICIT_KEY"); got != "42" {
		t.Fatalf("expected EXPLICIT_KEY loaded from explicit env file, got %q", got)
	}
}
