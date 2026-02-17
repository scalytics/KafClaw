package onboarding

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderSystemUnit(t *testing.T) {
	unit := renderSystemUnit(SetupOptions{
		ServiceUser: "kafclaw",
		BinaryPath:  "/usr/local/bin/kafclaw",
		Port:        18790,
		Profile:     "default",
		Version:     "1.2.3",
	}, "/home/kafclaw")

	if !strings.Contains(unit, "ExecStart=/usr/local/bin/kafclaw gateway --port 18790") {
		t.Fatalf("unexpected ExecStart in unit: %s", unit)
	}
	if !strings.Contains(unit, "EnvironmentFile=-/home/kafclaw/.config/kafclaw/env") {
		t.Fatalf("missing env file reference: %s", unit)
	}
	if !strings.Contains(unit, "User=kafclaw") {
		t.Fatalf("missing User directive: %s", unit)
	}
}

func TestRenderOverrideAndEnvFile(t *testing.T) {
	override := renderOverride("/home/kafclaw")
	if !strings.Contains(override, "EnvironmentFile=%h/.config/kafclaw/env") {
		t.Fatalf("missing override env file: %s", override)
	}
	if !strings.Contains(override, "Environment=KAFCLAW_CONFIG=/home/kafclaw/.kafclaw/config.json") {
		t.Fatalf("missing KAFCLAW_CONFIG: %s", override)
	}

	env := renderEnvFile("/home/kafclaw")
	if !strings.Contains(env, "MIKROBOT_GATEWAY_AUTH_TOKEN=") {
		t.Fatalf("missing auth token entry: %s", env)
	}
}

func TestSetupSystemdGatewayWritesFiles(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home", "kafclaw")
	installRoot := filepath.Join(tmp, "root")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	res, err := SetupSystemdGateway(SetupOptions{
		ServiceUser: "kafclaw",
		ServiceHome: home,
		BinaryPath:  "/usr/local/bin/kafclaw",
		Port:        18790,
		Profile:     "prod",
		Version:     "2.0.0",
		InstallRoot: installRoot,
	})
	if err != nil {
		t.Fatalf("setup systemd gateway: %v", err)
	}
	if res.UserCreated {
		t.Fatal("expected UserCreated=false when ServiceHome override is provided")
	}

	for _, p := range []string{res.ServicePath, res.OverridePath, res.EnvPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file to exist: %s (%v)", p, err)
		}
	}
}

func TestSetupSystemdGatewayKeepsExistingEnvFile(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home", "kafclaw")
	installRoot := filepath.Join(tmp, "root")
	envPath := filepath.Join(home, ".config", "kafclaw", "env")
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("EXISTING=1\n"), 0o600); err != nil {
		t.Fatalf("write pre-existing env file: %v", err)
	}

	_, err := SetupSystemdGateway(SetupOptions{
		ServiceUser: "kafclaw",
		ServiceHome: home,
		BinaryPath:  "/usr/local/bin/kafclaw",
		Port:        18790,
		Profile:     "default",
		Version:     "dev",
		InstallRoot: installRoot,
	})
	if err != nil {
		t.Fatalf("setup systemd gateway: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "EXISTING=1" {
		t.Fatalf("expected env file preserved, got: %q", string(data))
	}
}

func TestEnsureNonRootUserFallbacksToCurrentHome(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("requires non-root test environment")
	}
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("current user: %v", err)
	}

	created, home, err := ensureNonRootUser("__definitely_not_a_real_user__")
	if err != nil {
		t.Fatalf("ensureNonRootUser: %v", err)
	}
	if created {
		t.Fatal("expected created=false for non-root fallback")
	}
	if home != cur.HomeDir {
		t.Fatalf("expected fallback home %q, got %q", cur.HomeDir, home)
	}
}

func TestEnsureNonRootUserRootFlowUserAlreadyExists(t *testing.T) {
	origEUID := currentEUID
	origLookup := lookupUserFn
	defer func() {
		currentEUID = origEUID
		lookupUserFn = origLookup
	}()

	currentEUID = func() int { return 0 }
	lookupUserFn = func(name string) (*user.User, error) {
		return &user.User{Username: name, HomeDir: "/home/" + name}, nil
	}

	created, home, err := ensureNonRootUser("svc")
	if err != nil {
		t.Fatalf("ensureNonRootUser: %v", err)
	}
	if created {
		t.Fatal("expected created=false when user exists")
	}
	if home != "/home/svc" {
		t.Fatalf("unexpected home: %q", home)
	}
}

func TestEnsureNonRootUserRootFlowCreatesUser(t *testing.T) {
	origEUID := currentEUID
	origLookup := lookupUserFn
	origRun := runCommandFn
	defer func() {
		currentEUID = origEUID
		lookupUserFn = origLookup
		runCommandFn = origRun
	}()

	currentEUID = func() int { return 0 }
	calls := 0
	lookupUserFn = func(name string) (*user.User, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("not found")
		}
		return &user.User{Username: name, HomeDir: "/home/" + name}, nil
	}
	runCommandFn = func(name string, args ...string) ([]byte, error) {
		return []byte("ok"), nil
	}

	created, home, err := ensureNonRootUser("svc")
	if err != nil {
		t.Fatalf("ensureNonRootUser: %v", err)
	}
	if !created {
		t.Fatal("expected created=true")
	}
	if home != "/home/svc" {
		t.Fatalf("unexpected home: %q", home)
	}
}

func TestEnsureNonRootUserRootFlowCreateFails(t *testing.T) {
	origEUID := currentEUID
	origLookup := lookupUserFn
	origRun := runCommandFn
	defer func() {
		currentEUID = origEUID
		lookupUserFn = origLookup
		runCommandFn = origRun
	}()

	currentEUID = func() int { return 0 }
	lookupUserFn = func(name string) (*user.User, error) {
		return nil, errors.New("not found")
	}
	runCommandFn = func(name string, args ...string) ([]byte, error) {
		return []byte("boom"), errors.New("failed")
	}

	if _, _, err := ensureNonRootUser("svc"); err == nil {
		t.Fatal("expected create-user error")
	}
}

func TestSetupSystemdGatewayValidationErrors(t *testing.T) {
	_, err := SetupSystemdGateway(SetupOptions{})
	if err == nil {
		t.Fatal("expected validation error for empty options")
	}

	_, err = SetupSystemdGateway(SetupOptions{ServiceUser: "kafclaw", BinaryPath: "/bin/kafclaw", Port: 0})
	if err == nil {
		t.Fatal("expected validation error for invalid port")
	}
}

func TestSetupSystemdGatewayUsesEnsureUserWhenHomeMissing(t *testing.T) {
	tmp := t.TempDir()
	installRoot := filepath.Join(tmp, "root")
	home := filepath.Join(tmp, "home", "svc")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	origEnsure := ensureUserFn
	defer func() { ensureUserFn = origEnsure }()
	ensureUserFn = func(name string) (bool, string, error) {
		return true, home, nil
	}

	res, err := SetupSystemdGateway(SetupOptions{
		ServiceUser: "svc",
		BinaryPath:  "/usr/local/bin/kafclaw",
		Port:        18790,
		InstallRoot: installRoot,
	})
	if err != nil {
		t.Fatalf("setup systemd gateway: %v", err)
	}
	if !res.UserCreated {
		t.Fatal("expected UserCreated=true from injected ensureUserFn")
	}
}

func TestSetupSystemdGatewayPropagatesEnsureUserError(t *testing.T) {
	origEnsure := ensureUserFn
	defer func() { ensureUserFn = origEnsure }()
	ensureUserFn = func(name string) (bool, string, error) {
		return false, "", errors.New("boom")
	}

	if _, err := SetupSystemdGateway(SetupOptions{
		ServiceUser: "svc",
		BinaryPath:  "/usr/local/bin/kafclaw",
		Port:        18790,
		InstallRoot: t.TempDir(),
	}); err == nil {
		t.Fatal("expected error from ensureUserFn")
	}
}

func TestShellEscape(t *testing.T) {
	if got := shellEscape("/usr/local/bin/kafclaw"); got != "/usr/local/bin/kafclaw" {
		t.Fatalf("unexpected simple escape: %q", got)
	}
	if got := shellEscape("/tmp/my bin/kafclaw"); !strings.HasPrefix(got, "\"") {
		t.Fatalf("expected quoted path for whitespace, got %q", got)
	}
}
