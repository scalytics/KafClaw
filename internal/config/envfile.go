package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

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
