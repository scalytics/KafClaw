package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/spf13/cobra"
)

func TestOnboardNonInteractiveRequiresAcceptRisk(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)
	origAcceptRisk := onboardAcceptRisk
	defer func() { onboardAcceptRisk = origAcceptRisk }()
	onboardAcceptRisk = false

	_, err := runRootCommand(t, "onboard", "--non-interactive", "--mode=local", "--llm=skip", "--skip-skills")
	if err == nil {
		t.Fatal("expected onboard to fail without --accept-risk in non-interactive mode")
	}
}

func TestOnboardJSONSummary(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	out, err := runRootCommand(t, "onboard", "--non-interactive", "--accept-risk", "--mode=local", "--llm=skip", "--skip-skills", "--json")
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if !strings.Contains(out, "\"configPath\"") || !strings.Contains(out, "\"skills\"") {
		t.Fatalf("expected onboarding JSON summary in output, got %q", out)
	}
}

func TestOnboardSkillsBootstrapPath(t *testing.T) {
	tmpDir := t.TempDir()
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

	out, err := runRootCommand(t, "onboard", "--non-interactive", "--accept-risk", "--mode=local", "--llm=skip", "--skip-skills=false", "--json")
	if err != nil {
		t.Fatalf("onboard failed: %v", err)
	}
	if !strings.Contains(out, "\"enabled\": true") && !strings.Contains(out, "\"enabled\":true") {
		t.Fatalf("expected skills enabled in onboarding output, got %q", out)
	}
}

func TestOnboardConfiguresOAuthCapabilities(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t,
		"onboard",
		"--non-interactive",
		"--accept-risk",
		"--mode=local",
		"--llm=skip",
		"--skip-skills",
		"--google-workspace-read=mail,drive",
		"--m365-read=calendar",
	); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".kafclaw", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	skillsCfg, ok := cfg["skills"].(map[string]any)
	if !ok {
		t.Fatalf("missing skills config")
	}
	entries, ok := skillsCfg["entries"].(map[string]any)
	if !ok {
		t.Fatalf("missing skills.entries")
	}
	if _, ok := entries["google-workspace"].(map[string]any); !ok {
		t.Fatalf("expected google-workspace entry, got %#v", entries["google-workspace"])
	}
	if _, ok := entries["m365"].(map[string]any); !ok {
		t.Fatalf("expected m365 entry, got %#v", entries["m365"])
	}
}

func TestOnboardConfiguresKafkaSecurity(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t,
		"onboard",
		"--non-interactive",
		"--accept-risk",
		"--skip-skills",
		"--mode=local-kafka",
		"--llm=skip",
		"--kafka-brokers=broker1:9092,broker2:9092",
		"--kafka-security-protocol=SASL_SSL",
		"--kafka-sasl-mechanism=SCRAM-SHA-512",
		"--kafka-sasl-username=svc-user",
		"--kafka-sasl-password=svc-pass",
		"--kafka-tls-ca-file=/etc/ssl/kafka-ca.pem",
		"--kafka-tls-cert-file=/etc/ssl/client.pem",
		"--kafka-tls-key-file=/etc/ssl/client.key",
	); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".kafclaw", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	group, ok := cfg["group"].(map[string]any)
	if !ok {
		t.Fatalf("missing group section")
	}
	if v, _ := group["kafkaSecurityProtocol"].(string); v != "SASL_SSL" {
		t.Fatalf("expected kafkaSecurityProtocol SASL_SSL, got %q", v)
	}
	if v, _ := group["kafkaSaslMechanism"].(string); v != "SCRAM-SHA-512" {
		t.Fatalf("expected kafkaSaslMechanism SCRAM-SHA-512, got %q", v)
	}
	if v, _ := group["kafkaSaslUsername"].(string); v != "svc-user" {
		t.Fatalf("expected kafkaSaslUsername svc-user, got %q", v)
	}
	if v, _ := group["kafkaTlsCAFile"].(string); v != "/etc/ssl/kafka-ca.pem" {
		t.Fatalf("expected kafkaTlsCAFile set, got %q", v)
	}
}

func TestOnboardNonInteractiveRequiresModeAndLLM(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)
	origMode := onboardMode
	origProfile := onboardProfile
	origLLM := onboardLLMPreset
	defer func() {
		onboardMode = origMode
		onboardProfile = origProfile
		onboardLLMPreset = origLLM
	}()
	onboardMode = ""
	onboardProfile = ""
	onboardLLMPreset = ""

	if _, err := runRootCommand(t, "onboard", "--non-interactive", "--accept-risk", "--skip-skills"); err == nil {
		t.Fatal("expected non-interactive onboarding to fail without mode/profile and llm")
	}
}

func TestOnboardNonInteractiveNonLoopbackRequiresAck(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t, "onboard",
		"--non-interactive",
		"--accept-risk",
		"--skip-skills",
		"--mode=remote",
		"--llm=skip",
	); err == nil {
		t.Fatal("expected remote non-loopback onboarding to require --allow-non-loopback")
	}

	if _, err := runRootCommand(t, "onboard",
		"--non-interactive",
		"--accept-risk",
		"--skip-skills",
		"--mode=remote",
		"--llm=skip",
		"--allow-non-loopback",
	); err != nil {
		t.Fatalf("expected onboarding to pass with non-loopback acknowledgement: %v", err)
	}
}

func TestOnboardPersistsGatewayPort(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	_ = os.Setenv("HOME", tmpDir)

	if _, err := runRootCommand(t, "onboard",
		"--non-interactive",
		"--accept-risk",
		"--skip-skills",
		"--mode=local",
		"--llm=skip",
		"--gateway-port=19990",
	); err != nil {
		t.Fatalf("onboard failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".kafclaw", "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	gateway, ok := cfg["gateway"].(map[string]any)
	if !ok {
		t.Fatalf("missing gateway config")
	}
	port, ok := gateway["port"].(float64)
	if !ok {
		t.Fatalf("missing gateway.port")
	}
	if int(port) != 19990 {
		t.Fatalf("expected gateway.port 19990, got %d", int(port))
	}
}

func TestValidateOnboardNonInteractiveFlags(t *testing.T) {
	origNonInteractive := onboardNonInteractive
	origMode := onboardMode
	origProfile := onboardProfile
	origLLM := onboardLLMPreset
	defer func() {
		onboardNonInteractive = origNonInteractive
		onboardMode = origMode
		onboardProfile = origProfile
		onboardLLMPreset = origLLM
	}()

	onboardNonInteractive = false
	if err := validateOnboardNonInteractiveFlags(); err != nil {
		t.Fatalf("expected nil when non-interactive is disabled, got %v", err)
	}

	onboardNonInteractive = true
	onboardMode = ""
	onboardProfile = ""
	onboardLLMPreset = "skip"
	if err := validateOnboardNonInteractiveFlags(); err == nil {
		t.Fatal("expected mode/profile validation error")
	}

	onboardMode = "local"
	onboardLLMPreset = ""
	if err := validateOnboardNonInteractiveFlags(); err == nil {
		t.Fatal("expected llm validation error")
	}

	onboardMode = "local"
	onboardLLMPreset = "skip"
	if err := validateOnboardNonInteractiveFlags(); err != nil {
		t.Fatalf("expected valid non-interactive flags, got %v", err)
	}
}

func TestPreflightOnboardingConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".kafclaw", "config.json")

	if err := preflightOnboardingConfig(nil, cfgPath); err == nil {
		t.Fatal("expected nil config preflight error")
	}

	cfg := config.DefaultConfig()
	cfg.Paths.Workspace = ""
	if err := preflightOnboardingConfig(cfg, cfgPath); err == nil {
		t.Fatal("expected empty workspace preflight error")
	}

	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(tmpDir, "workspace")
	cfg.Gateway.Port = 0
	if err := preflightOnboardingConfig(cfg, cfgPath); err == nil {
		t.Fatal("expected invalid gateway.port preflight error")
	}

	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(tmpDir, "workspace")
	cfg.Gateway.Port = 18790
	cfg.Gateway.DashboardPort = 18790
	if err := preflightOnboardingConfig(cfg, cfgPath); err == nil {
		t.Fatal("expected gateway port collision preflight error")
	}

	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(tmpDir, "workspace")
	cfg.Gateway.Port = 18790
	cfg.Gateway.DashboardPort = 0
	if err := preflightOnboardingConfig(cfg, cfgPath); err == nil {
		t.Fatal("expected invalid gateway.dashboardPort preflight error")
	}

	cfgDirAsFile := filepath.Join(tmpDir, "cfg-parent-file")
	if err := os.WriteFile(cfgDirAsFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write cfg parent file: %v", err)
	}
	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(tmpDir, "workspace")
	cfg.Gateway.Port = 18790
	cfg.Gateway.DashboardPort = 18791
	if err := preflightOnboardingConfig(cfg, filepath.Join(cfgDirAsFile, "config.json")); err == nil {
		t.Fatal("expected config directory creation preflight error")
	}

	wsParentAsFile := filepath.Join(tmpDir, "workspace-parent-file")
	if err := os.WriteFile(wsParentAsFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write workspace parent file: %v", err)
	}
	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(wsParentAsFile, "workspace")
	cfg.Gateway.Port = 18790
	cfg.Gateway.DashboardPort = 18791
	if err := preflightOnboardingConfig(cfg, cfgPath); err == nil {
		t.Fatal("expected workspace mkdir preflight error")
	}

	cfg = config.DefaultConfig()
	cfg.Paths.Workspace = filepath.Join(tmpDir, "workspace")
	cfg.Gateway.Port = 18790
	cfg.Gateway.DashboardPort = 18791
	if err := preflightOnboardingConfig(cfg, cfgPath); err != nil {
		t.Fatalf("expected valid preflight, got %v", err)
	}
}

func TestEnforceGatewayBindSafety(t *testing.T) {
	origNonInteractive := onboardNonInteractive
	origAllowNonLoopback := onboardAllowNonLoopback
	defer func() {
		onboardNonInteractive = origNonInteractive
		onboardAllowNonLoopback = origAllowNonLoopback
	}()

	cmd := &cobra.Command{}
	in := bytes.NewBufferString("")
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)

	if err := enforceGatewayBindSafety(nil, cmd); err != nil {
		t.Fatalf("expected nil for nil config, got %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.Gateway.Host = "127.0.0.1"
	if err := enforceGatewayBindSafety(cfg, cmd); err != nil {
		t.Fatalf("expected nil for loopback host, got %v", err)
	}

	cfg.Gateway.Host = "0.0.0.0"
	cfg.Gateway.AuthToken = ""
	if err := enforceGatewayBindSafety(cfg, cmd); err == nil {
		t.Fatal("expected auth token enforcement error")
	}

	cfg.Gateway.AuthToken = "token"
	onboardAllowNonLoopback = true
	onboardNonInteractive = true
	if err := enforceGatewayBindSafety(cfg, cmd); err != nil {
		t.Fatalf("expected acknowledged non-loopback to pass, got %v", err)
	}

	onboardAllowNonLoopback = false
	onboardNonInteractive = true
	if err := enforceGatewayBindSafety(cfg, cmd); err == nil {
		t.Fatal("expected non-interactive acknowledgement error")
	}

	onboardNonInteractive = false
	cmd = &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("n\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := enforceGatewayBindSafety(cfg, cmd); err == nil {
		t.Fatal("expected interactive abort for non-loopback bind")
	}

	cmd = &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString("y\n"))
	cmd.SetOut(&bytes.Buffer{})
	if err := enforceGatewayBindSafety(cfg, cmd); err != nil {
		t.Fatalf("expected interactive approval to pass, got %v", err)
	}
}
