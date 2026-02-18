package config

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const localTombFileName = "tomb.rr"
const localTombEnvAAD = "kafclaw-local-env-secrets-v1"

type localTombDoc struct {
	Version       string `json:"version"`
	MasterKey     string `json:"masterKey"`
	EnvNonce      string `json:"envNonce,omitempty"`
	EnvCiphertext string `json:"envCiphertext,omitempty"`
}

// LoadEnvFileCandidates loads environment variables from known files.
// Existing process env vars are never overridden.
func LoadEnvFileCandidates() {
	candidates := make([]string, 0, 4)
	if explicit := strings.TrimSpace(os.Getenv("KAFCLAW_ENV_FILE")); explicit != "" {
		candidates = append(candidates, explicit)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".config", "kafclaw", "env"),
			filepath.Join(home, ".kafclaw", "env"),
			filepath.Join(home, ".kafclaw", ".env"),
		)
	}
	seen := map[string]struct{}{}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		abs := p
		if !filepath.IsAbs(abs) {
			if resolved, err := filepath.Abs(p); err == nil {
				abs = resolved
			}
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		_ = loadEnvFile(abs)
	}
	// Merge tomb-managed secrets into process env (without overriding explicit env).
	if kv, err := loadEnvSecretsFromTomb(); err == nil {
		for k, v := range kv {
			if _, exists := os.LookupEnv(k); exists {
				continue
			}
			_ = os.Setenv(k, v)
		}
	}
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		i := strings.IndexRune(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[i+1:])
		val = trimOptionalQuotes(val)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}

func trimOptionalQuotes(v string) string {
	if len(v) < 2 {
		return v
	}
	if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
		return v[1 : len(v)-1]
	}
	if strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") {
		return v[1 : len(v)-1]
	}
	return v
}

func loadEnvSecretsFromTomb() (map[string]string, error) {
	tombPath, err := resolveLocalTombPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(tombPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	doc, err := decodeLocalTombDoc(data)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.EnvNonce) == "" || strings.TrimSpace(doc.EnvCiphertext) == "" {
		return map[string]string{}, nil
	}
	key, err := decodeMasterKey(doc.MasterKey)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(doc.EnvNonce))
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(doc.EnvCiphertext))
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte(localTombEnvAAD))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(plain, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func resolveLocalTombPath() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("KAFCLAW_OAUTH_TOMB_FILE")); explicit != "" {
		if strings.HasPrefix(explicit, "~") {
			home, err := resolveEnvHomeDir()
			if err != nil {
				return "", err
			}
			return expandTildePath(explicit, home), nil
		}
		return explicit, nil
	}
	home, err := resolveEnvHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kafclaw", localTombFileName), nil
}

func resolveEnvHomeDir() (string, error) {
	if h := strings.TrimSpace(os.Getenv("KAFCLAW_HOME")); h != "" {
		if strings.HasPrefix(h, "~") {
			base, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			return expandTildePath(h, base), nil
		}
		return h, nil
	}
	return os.UserHomeDir()
}

func expandTildePath(path string, home string) string {
	switch {
	case path == "~":
		return home
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[2:])
	default:
		return path
	}
}

func decodeLocalTombDoc(data []byte) (*localTombDoc, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, os.ErrInvalid
	}
	var doc localTombDoc
	if err := json.Unmarshal([]byte(trimmed), &doc); err == nil && strings.TrimSpace(doc.MasterKey) != "" {
		return &doc, nil
	}
	// Backward-compatible: raw base64 master key.
	if _, err := decodeMasterKey(trimmed); err == nil {
		return &localTombDoc{Version: "v1", MasterKey: trimmed}, nil
	}
	return nil, os.ErrInvalid
}

func decodeMasterKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	decoded := make([]byte, base64.RawStdEncoding.DecodedLen(len(trimmed)))
	n, err := base64.RawStdEncoding.Decode(decoded, []byte(trimmed))
	if err != nil {
		return nil, err
	}
	if n != 32 {
		return nil, os.ErrInvalid
	}
	return decoded[:n], nil
}
