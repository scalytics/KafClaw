package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/spf13/cobra"
)

var configureSubagentsAllowAgents string
var configureClearSubagentsAllowAgents bool
var configureNonInteractive bool
var configureSkillsEnabled bool
var configureSkillsNodeManager string
var configureSkillsScope string
var configureEnableSkills []string
var configureDisableSkills []string
var configureGoogleWorkspaceRead string
var configureM365Read string

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Guided configuration updates",
	RunE:  runConfigure,
}

func init() {
	configureCmd.Flags().StringVar(&configureSubagentsAllowAgents, "subagents-allow-agents", "", "Comma-separated agent IDs allowed for sessions_spawn agentId (use '*' for any)")
	configureCmd.Flags().BoolVar(&configureClearSubagentsAllowAgents, "clear-subagents-allow-agents", false, "Clear subagent allowlist (default behavior: current agent only)")
	configureCmd.Flags().BoolVar(&configureSkillsEnabled, "skills-enabled", false, "Enable or disable global skills system (with --skills-enabled-set)")
	configureCmd.Flags().StringVar(&configureSkillsNodeManager, "skills-node-manager", "", "Set skills node manager (npm|pnpm|bun)")
	configureCmd.Flags().StringVar(&configureSkillsScope, "skills-scope", "", "Set skills scope (all|selected)")
	configureCmd.Flags().StringSliceVar(&configureEnableSkills, "enable-skill", nil, "Enable one or more skills (repeatable)")
	configureCmd.Flags().StringSliceVar(&configureDisableSkills, "disable-skill", nil, "Disable one or more skills (repeatable)")
	configureCmd.Flags().StringVar(&configureGoogleWorkspaceRead, "google-workspace-read", "", "Set google-workspace capabilities (mail,calendar,drive,all)")
	configureCmd.Flags().StringVar(&configureM365Read, "m365-read", "", "Set m365 capabilities (mail,calendar,files,all)")
	configureCmd.Flags().BoolVar(&configureNonInteractive, "non-interactive", false, "Apply flags only and skip prompts")
	configureCmd.Flags().Bool("skills-enabled-set", false, "Apply --skills-enabled value")
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	updatedAllowAgents := cfg.Tools.Subagents.AllowAgents
	hasFlagUpdate := configureClearSubagentsAllowAgents || strings.TrimSpace(configureSubagentsAllowAgents) != ""
	if configureClearSubagentsAllowAgents {
		updatedAllowAgents = nil
	} else if strings.TrimSpace(configureSubagentsAllowAgents) != "" {
		updatedAllowAgents = parseCSVList(configureSubagentsAllowAgents)
	}

	if !hasFlagUpdate && !configureNonInteractive {
		reader := bufio.NewReader(cmd.InOrStdin())
		current := strings.Join(cfg.Tools.Subagents.AllowAgents, ",")
		if current == "" {
			current = "(current agent only)"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Current subagent allowlist: %s\n", current)
		fmt.Fprint(cmd.OutOrStdout(), "Set subagent allowlist (comma-separated, '*' for any, empty to keep): ")
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("read input: %w", readErr)
		}
		line = strings.TrimSpace(line)
		if line != "" {
			updatedAllowAgents = parseCSVList(line)
		}
	}

	cfg.Tools.Subagents.AllowAgents = updatedAllowAgents

	if cmd.Flags().Changed("skills-enabled-set") {
		cfg.Skills.Enabled = configureSkillsEnabled
	}
	if strings.TrimSpace(configureSkillsNodeManager) != "" {
		val := strings.ToLower(strings.TrimSpace(configureSkillsNodeManager))
		switch val {
		case "npm", "pnpm", "bun":
			cfg.Skills.NodeManager = val
		default:
			return fmt.Errorf("invalid --skills-node-manager: %s (expected npm|pnpm|bun)", val)
		}
	}
	if strings.TrimSpace(configureSkillsScope) != "" {
		scope := strings.ToLower(strings.TrimSpace(configureSkillsScope))
		switch scope {
		case "all", "selected":
			cfg.Skills.Scope = scope
		default:
			return fmt.Errorf("invalid --skills-scope: %s (expected all|selected)", scope)
		}
	}
	if cfg.Skills.Entries == nil {
		cfg.Skills.Entries = map[string]config.SkillEntryConfig{}
	}
	for _, name := range configureEnableSkills {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		entry := cfg.Skills.Entries[name]
		entry.Enabled = true
		cfg.Skills.Entries[name] = entry
	}
	for _, name := range configureDisableSkills {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		entry := cfg.Skills.Entries[name]
		entry.Enabled = false
		cfg.Skills.Entries[name] = entry
	}

	if strings.TrimSpace(configureGoogleWorkspaceRead) != "" {
		caps, err := parseCapabilitySelection(configureGoogleWorkspaceRead, map[string]struct{}{
			"mail": {}, "calendar": {}, "drive": {}, "all": {},
		})
		if err != nil {
			return fmt.Errorf("invalid --google-workspace-read: %w", err)
		}
		entry := cfg.Skills.Entries["google-workspace"]
		entry.Enabled = true
		entry.Capabilities = caps
		cfg.Skills.Entries["google-workspace"] = entry
	}
	if strings.TrimSpace(configureM365Read) != "" {
		caps, err := parseCapabilitySelection(configureM365Read, map[string]struct{}{
			"mail": {}, "calendar": {}, "files": {}, "all": {},
		})
		if err != nil {
			return fmt.Errorf("invalid --m365-read: %w", err)
		}
		entry := cfg.Skills.Entries["m365"]
		entry.Enabled = true
		entry.Capabilities = caps
		cfg.Skills.Entries["m365"] = entry
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	state := "(current agent only)"
	if len(cfg.Tools.Subagents.AllowAgents) > 0 {
		state = strings.Join(cfg.Tools.Subagents.AllowAgents, ",")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Updated tools.subagents.allowAgents: %s\n", state)
	fmt.Fprintf(cmd.OutOrStdout(), "Updated skills.enabled: %v\n", cfg.Skills.Enabled)
	fmt.Fprintf(cmd.OutOrStdout(), "Updated skills.nodeManager: %s\n", cfg.Skills.NodeManager)
	fmt.Fprintf(cmd.OutOrStdout(), "Updated skills.scope: %s\n", cfg.Skills.Scope)
	return nil
}

func parseCapabilitySelection(raw string, allowed map[string]struct{}) ([]string, error) {
	parts := parseCSVList(raw)
	if len(parts) == 0 {
		return nil, fmt.Errorf("expected comma-separated capability list")
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" {
			continue
		}
		if _, ok := allowed[v]; !ok {
			return nil, fmt.Errorf("unsupported capability %q", v)
		}
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid capabilities provided")
	}
	return out, nil
}

func parseCSVList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
