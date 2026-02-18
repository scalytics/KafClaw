package config

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
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

func TestLoadEnvFileCandidatesLoadsSecretsFromTomb(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	tombPath := filepath.Join(home, ".config", "kafclaw", localTombFileName)
	if err := os.MkdirAll(filepath.Dir(tombPath), 0o700); err != nil {
		t.Fatalf("mkdir tomb dir: %v", err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}
	plain, err := json.Marshal(map[string]string{
		"OPENAI_API_KEY": "from_tomb",
	})
	if err != nil {
		t.Fatalf("marshal plain: %v", err)
	}
	nonce := []byte("0123456789ab")
	ciphertext := gcm.Seal(nil, nonce, plain, []byte(localTombEnvAAD))
	doc := localTombDoc{
		Version:       "v1",
		MasterKey:     base64.RawStdEncoding.EncodeToString(key),
		EnvNonce:      base64.RawStdEncoding.EncodeToString(nonce),
		EnvCiphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	}
	docData, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal tomb doc: %v", err)
	}
	if err := os.WriteFile(tombPath, append(docData, '\n'), 0o600); err != nil {
		t.Fatalf("write tomb file: %v", err)
	}

	origHome := os.Getenv("HOME")
	origKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("OPENAI_API_KEY", origKey)
	_ = os.Setenv("HOME", home)
	_ = os.Unsetenv("OPENAI_API_KEY")

	LoadEnvFileCandidates()

	if got := os.Getenv("OPENAI_API_KEY"); got != "from_tomb" {
		t.Fatalf("expected OPENAI_API_KEY loaded from tomb, got %q", got)
	}

	_ = os.Setenv("OPENAI_API_KEY", "already_set")
	LoadEnvFileCandidates()
	if got := os.Getenv("OPENAI_API_KEY"); got != "already_set" {
		t.Fatalf("expected existing env key preserved, got %q", got)
	}
}
