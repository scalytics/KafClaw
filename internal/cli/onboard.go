package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/KafClaw/KafClaw/internal/identity"
	"github.com/KafClaw/KafClaw/internal/onboarding"
	skillruntime "github.com/KafClaw/KafClaw/internal/skills"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Initialize configuration and scaffold workspace",
	RunE:  runOnboard,
}

var onboardForce bool
var onboardSystemd bool
var onboardServiceUser string
var onboardServiceBinary string
var onboardServicePort int
var onboardInstallRoot string
var onboardProfile string
var onboardMode string
var onboardLLMPreset string
var onboardLLMToken string
var onboardLLMAPIBase string
var onboardLLMModel string
var onboardKafkaBrokers string
var onboardGroupName string
var onboardAgentID string
var onboardRole string
var onboardRemoteAuth string
var onboardSubMaxSpawnDepth int
var onboardSubMaxChildren int
var onboardSubMaxConcurrent int
var onboardSubArchiveMins int
var onboardSubModel string
var onboardSubThinking string
var onboardSubAllowAgents string
var onboardNonInteractive bool
var onboardAcceptRisk bool
var onboardJSON bool
var onboardSkipSkills bool
var onboardInstallClawhub bool
var onboardSkillsNodeMajor string

type onboardingSkillsSummary struct {
	Enabled      bool     `json:"enabled"`
	StateDirsOK  bool     `json:"stateDirsOk"`
	NodeFound    bool     `json:"nodeFound"`
	ClawhubFound bool     `json:"clawhubFound"`
	Remediation  []string `json:"remediation,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type onboardingSummary struct {
	ConfigPath string                  `json:"configPath"`
	Workspace  string                  `json:"workspace"`
	WorkRepo   string                  `json:"workRepo"`
	Skills     onboardingSkillsSummary `json:"skills"`
}

func init() {
	onboardCmd.Flags().BoolVarP(&onboardForce, "force", "f", false, "Overwrite existing config and soul files")
	onboardCmd.Flags().BoolVar(&onboardNonInteractive, "non-interactive", false, "Run onboarding without prompts")
	onboardCmd.Flags().BoolVar(&onboardAcceptRisk, "accept-risk", false, "Acknowledge risk for non-interactive onboarding")
	onboardCmd.Flags().BoolVar(&onboardJSON, "json", false, "Output onboarding summary as JSON")
	onboardCmd.Flags().StringVar(&onboardProfile, "profile", "", "Preset profile: local | local-kafka | remote")
	onboardCmd.Flags().StringVar(&onboardMode, "mode", "", "Runtime mode: local | local-kafka | remote")
	onboardCmd.Flags().StringVar(&onboardLLMPreset, "llm", "", "LLM setup: cli-token | openai-compatible | skip")
	onboardCmd.Flags().StringVar(&onboardLLMToken, "llm-token", "", "LLM API token")
	onboardCmd.Flags().StringVar(&onboardLLMAPIBase, "llm-api-base", "", "OpenAI-compatible API base (e.g. http://localhost:11434/v1)")
	onboardCmd.Flags().StringVar(&onboardLLMModel, "llm-model", "", "Default model name")
	onboardCmd.Flags().StringVar(&onboardKafkaBrokers, "kafka-brokers", "", "Kafka brokers for local-kafka mode")
	onboardCmd.Flags().StringVar(&onboardGroupName, "group-name", "", "Group name for local-kafka mode")
	onboardCmd.Flags().StringVar(&onboardAgentID, "agent-id", "", "Agent ID for local-kafka mode")
	onboardCmd.Flags().StringVar(&onboardRole, "role", "", "Orchestrator role for local-kafka mode")
	onboardCmd.Flags().StringVar(&onboardRemoteAuth, "remote-auth-token", "", "Gateway auth token for remote mode")
	onboardCmd.Flags().IntVar(&onboardSubMaxSpawnDepth, "subagents-max-spawn-depth", 0, "Override subagent max spawn depth")
	onboardCmd.Flags().IntVar(&onboardSubMaxChildren, "subagents-max-children", 0, "Override max active children per parent session")
	onboardCmd.Flags().IntVar(&onboardSubMaxConcurrent, "subagents-max-concurrent", 0, "Override max active subagents globally")
	onboardCmd.Flags().IntVar(&onboardSubArchiveMins, "subagents-archive-minutes", 0, "Override subagent archive-after minutes")
	onboardCmd.Flags().StringVar(&onboardSubAllowAgents, "subagents-allow-agents", "", "Comma-separated agent IDs allowed for sessions_spawn agentId (use '*' for any)")
	onboardCmd.Flags().StringVar(&onboardSubModel, "subagents-model", "", "Default model for spawned subagents")
	onboardCmd.Flags().StringVar(&onboardSubThinking, "subagents-thinking", "", "Default thinking level for spawned subagents")
	onboardCmd.Flags().BoolVar(&onboardSkipSkills, "skip-skills", false, "Skip skills bootstrap")
	onboardCmd.Flags().BoolVar(&onboardInstallClawhub, "install-clawhub", true, "Install clawhub via npm if missing during skills bootstrap")
	onboardCmd.Flags().StringVar(&onboardSkillsNodeMajor, "skills-node-major", "20", "Node major version to pin in .nvmrc for skills")
	onboardCmd.Flags().BoolVar(&onboardSystemd, "systemd", false, "Install systemd service + override + env file (Linux)")
	onboardCmd.Flags().StringVar(&onboardServiceUser, "service-user", "kafclaw", "Service user for systemd setup")
	onboardCmd.Flags().StringVar(&onboardServiceBinary, "service-binary", "/usr/local/bin/kafclaw", "kafclaw binary path for systemd ExecStart")
	onboardCmd.Flags().IntVar(&onboardServicePort, "service-port", 18790, "Gateway port for systemd ExecStart")
	onboardCmd.Flags().StringVar(&onboardInstallRoot, "service-install-root", "/", "Installation root for systemd files (testing/packaging)")
	_ = onboardCmd.Flags().MarkHidden("service-install-root")
	rootCmd.AddCommand(onboardCmd)
}

func runOnboard(cmd *cobra.Command, args []string) error {
	if onboardNonInteractive && !onboardAcceptRisk {
		return fmt.Errorf("non-interactive onboarding requires --accept-risk")
	}

	printHeader("🚀 KafClaw Onboard")
	fmt.Println("Initializing KafClaw...")

	cfgPath, _ := config.ConfigPath()

	if _, err := os.Stat(cfgPath); err == nil && !onboardForce {
		fmt.Printf("Config already exists at: %s\n", cfgPath)
		fmt.Println("Use --force (-f) to overwrite.")
	} else {
		cfg := config.DefaultConfig()
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
		} else {
			fmt.Printf("Config created at: %s\n", cfgPath)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config warning: %v (using defaults)\n", err)
	}
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if err := onboarding.RunProfileWizard(cfg, cmd.InOrStdin(), cmd.OutOrStdout(), onboarding.WizardParams{
		Profile:          onboardProfile,
		Mode:             onboardMode,
		LLMPreset:        onboardLLMPreset,
		LLMToken:         onboardLLMToken,
		LLMAPIBase:       onboardLLMAPIBase,
		LLMModel:         onboardLLMModel,
		KafkaBrokers:     onboardKafkaBrokers,
		GroupName:        onboardGroupName,
		AgentID:          onboardAgentID,
		Role:             onboardRole,
		RemoteAuth:       onboardRemoteAuth,
		SubMaxSpawnDepth: onboardSubMaxSpawnDepth,
		SubMaxChildren:   onboardSubMaxChildren,
		SubMaxConcurrent: onboardSubMaxConcurrent,
		SubArchiveMins:   onboardSubArchiveMins,
		SubAllowAgents:   onboardSubAllowAgents,
		SubModel:         onboardSubModel,
		SubThinking:      onboardSubThinking,
		NonInteractive:   onboardNonInteractive,
	}); err != nil {
		fmt.Printf("Onboarding wizard error: %v\n", err)
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), onboarding.BuildProfileSummary(cfg))
	if !onboardNonInteractive {
		confirmReader := bufio.NewReader(cmd.InOrStdin())
		ok, err := onboarding.ConfirmApply(confirmReader, cmd.OutOrStdout())
		if err != nil {
			fmt.Printf("Onboarding confirmation error: %v\n", err)
			return nil
		}
		if !ok {
			fmt.Println("Onboarding aborted before writing config.")
			return nil
		}
	}
	if err := config.Save(cfg); err != nil {
		fmt.Printf("Error saving configured onboarding profile: %v\n", err)
		return nil
	}
	fmt.Printf("Updated configuration at: %s\n", cfgPath)

	fmt.Printf("\nWorkspace: %s\n", cfg.Paths.Workspace)
	result, err := identity.ScaffoldWorkspace(cfg.Paths.Workspace, onboardForce)
	if err != nil {
		fmt.Printf("Error scaffolding workspace: %v\n", err)
	} else {
		for _, name := range result.Created {
			fmt.Printf("  + %s\n", name)
		}
		for _, name := range result.Skipped {
			fmt.Printf("  ~ %s (exists, skipped)\n", name)
		}
		for _, e := range result.Errors {
			fmt.Printf("  ! %s\n", e)
		}
	}

	if warn, err := config.EnsureWorkRepo(cfg.Paths.WorkRepoPath); err != nil {
		fmt.Printf("Work repo error: %v\n", err)
	} else {
		fmt.Printf("\nWork repo: %s\n", cfg.Paths.WorkRepoPath)
		if warn != "" {
			fmt.Printf("  Warning: %s\n", warn)
		}
	}

	skillsSummary := onboardingSkillsSummary{
		Enabled:      cfg.Skills.Enabled,
		NodeFound:    skillruntime.HasBinary("node"),
		ClawhubFound: skillruntime.HasBinary("clawhub"),
	}
	if onboardSkipSkills {
		fmt.Println("\nSkills bootstrap: skipped (--skip-skills)")
	} else {
		enableSkills := onboardNonInteractive
		if !onboardNonInteractive {
			fmt.Print("\nEnable skills now? [Y/n]: ")
			confirmReader := bufio.NewReader(cmd.InOrStdin())
			line, _ := confirmReader.ReadString('\n')
			answer := strings.TrimSpace(strings.ToLower(line))
			enableSkills = answer == "" || answer == "y" || answer == "yes"
		}
		if enableSkills {
			cfg.Skills.Enabled = true
			skillsSummary.Enabled = true
			if cfg.Skills.NodeManager == "" {
				cfg.Skills.NodeManager = "npm"
			}
			if err := config.Save(cfg); err != nil {
				fmt.Printf("Skills bootstrap warning: failed saving config: %v\n", err)
				skillsSummary.Warnings = append(skillsSummary.Warnings, err.Error())
			}
			if _, err := skillruntime.EnsureStateDirs(); err != nil {
				fmt.Printf("Skills bootstrap warning: failed creating secure dirs: %v\n", err)
				skillsSummary.Warnings = append(skillsSummary.Warnings, err.Error())
			} else {
				skillsSummary.StateDirsOK = true
			}
			if _, err := skillruntime.EnsureNVMRC(cfg.Paths.WorkRepoPath, onboardSkillsNodeMajor); err != nil {
				fmt.Printf("Skills bootstrap warning: failed writing .nvmrc: %v\n", err)
				skillsSummary.Warnings = append(skillsSummary.Warnings, err.Error())
			}
			skillsSummary.NodeFound = skillruntime.HasBinary("node")
			if !skillsSummary.NodeFound {
				fmt.Println("Skills bootstrap warning: node not found in PATH; install Node.js and run 'kafclaw skills enable'.")
				skillsSummary.Remediation = append(skillsSummary.Remediation, "Install Node.js and rerun `kafclaw skills enable`.")
			} else if err := skillruntime.EnsureClawhub(onboardInstallClawhub); err != nil {
				fmt.Printf("Skills bootstrap warning: %v\n", err)
				skillsSummary.Warnings = append(skillsSummary.Warnings, err.Error())
				skillsSummary.Remediation = append(skillsSummary.Remediation, "Install clawhub (`npm install -g --ignore-scripts clawhub`) or rerun with --install-clawhub.")
			}
			skillsSummary.ClawhubFound = skillruntime.HasBinary("clawhub")
			fmt.Println("Skills bootstrap complete. Use 'kafclaw skills list' to inspect status.")
		} else {
			fmt.Println("\nSkills bootstrap: not enabled")
		}
	}

	fmt.Println("\nNext steps:")
	fmt.Println("1. Review ~/.kafclaw/config.json and ~/.config/kafclaw/env.")
	fmt.Println("2. Customize soul files in your workspace (SOUL.md, USER.md, etc.)")
	fmt.Println("3. Run 'kafclaw agent -m \"hello\"' to test.")

	if onboardJSON {
		summary := onboardingSummary{
			ConfigPath: cfgPath,
			Workspace:  cfg.Paths.Workspace,
			WorkRepo:   cfg.Paths.WorkRepoPath,
			Skills:     skillsSummary,
		}
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	}

	if onboardSystemd {
		if runtime.GOOS != "linux" {
			fmt.Println("\nSystemd setup is only supported on Linux.")
			return nil
		}
		result, err := onboarding.SetupSystemdGateway(onboarding.SetupOptions{
			ServiceUser: onboardServiceUser,
			BinaryPath:  onboardServiceBinary,
			Port:        onboardServicePort,
			Profile:     "default",
			Version:     version,
			InstallRoot: onboardInstallRoot,
		})
		if err != nil {
			fmt.Printf("\nSystemd setup failed: %v\n", err)
			return nil
		}

		fmt.Println("\nSystemd setup complete:")
		if result.UserCreated {
			fmt.Printf("  + Created user: %s\n", onboardServiceUser)
		}
		fmt.Printf("  + Service unit: %s\n", result.ServicePath)
		fmt.Printf("  + Override file: %s\n", result.OverridePath)
		fmt.Printf("  + Env file: %s\n", result.EnvPath)
		fmt.Println("  Next (as root): systemctl daemon-reload && systemctl enable --now kafclaw-gateway.service")
	}
	return nil
}
