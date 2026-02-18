package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/KafClaw/KafClaw/internal/config"
	"github.com/KafClaw/KafClaw/internal/skills"
	"github.com/spf13/cobra"
)

var (
	skillsListJSON         bool
	skillsStatusJSON       bool
	skillsEnableInstallHub bool
	skillsEnableNodeMajor  string
	skillsVerifyJSON       bool
	skillsInstallJSON      bool
	skillsUpdateJSON       bool
	skillsApproveWarnings  bool
	skillsPrereqDryRun     bool
	skillsPrereqJSON       bool
	skillsPrereqYes        bool
	skillsAuthProfile      string
	skillsAuthClientID     string
	skillsAuthClientSecret string
	skillsAuthRedirectURI  string
	skillsAuthScopes       string
	skillsAuthAccess       string
	skillsAuthTenantID     string
	skillsAuthCode         string
	skillsAuthState        string
	skillsAuthCallbackURL  string
	skillsAuthJSON         bool
	skillsExecJSON         bool
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage bundled and external skills",
}

var skillsPrereqCmd = &cobra.Command{
	Use:   "prereq",
	Short: "Check/install skill prerequisites",
}

var skillsAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Headless auth flows for skills",
}

type skillListRow struct {
	Name       string   `json:"name"`
	Source     string   `json:"source"`
	Enabled    bool     `json:"enabled"`
	Configured bool     `json:"configured"`
	Eligible   bool     `json:"eligible"`
	Missing    []string `json:"missing,omitempty"`
}

type skillsStatusPayload struct {
	SkillsEnabled    bool           `json:"skillsEnabled"`
	Scope            string         `json:"scope"`
	RuntimeIsolation string         `json:"runtimeIsolation"`
	NodeFound        bool           `json:"nodeFound"`
	ClawhubFound     bool           `json:"clawhubFound"`
	Rows             []skillListRow `json:"rows"`
}

var skillsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable the skills system and bootstrap prerequisites",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		cfg.Skills.Enabled = true
		if cfg.Skills.NodeManager == "" {
			cfg.Skills.NodeManager = "npm"
		}
		if err := config.Save(cfg); err != nil {
			return formatSkillError("CONFIG_SAVE_FAILED", err, "verify write permissions for ~/.kafclaw")
		}
		if _, err := skills.EnsureStateDirs(); err != nil {
			return formatSkillError("STATE_DIRS_FAILED", err, "check permissions on ~/.kafclaw/skills")
		}
		if _, err := skills.EnsureNVMRC(cfg.Paths.WorkRepoPath, skillsEnableNodeMajor); err != nil {
			return formatSkillError("NVMRC_WRITE_FAILED", err, "check work repo path permissions")
		}
		if !skills.HasBinary("node") {
			return formatSkillError("NODE_MISSING", fmt.Errorf("node not found in PATH"), "install Node.js, then rerun `kafclaw skills enable`")
		}
		if err := skills.EnsureClawhub(skillsEnableInstallHub); err != nil {
			return formatSkillError("CLAWHUB_BOOTSTRAP_FAILED", err, "install clawhub (`npm install -g --ignore-scripts clawhub`) or rerun with --install-clawhub=false")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Skills system enabled.")
		return nil
	},
}

var skillsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable the skills system",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		cfg.Skills.Enabled = false
		if err := config.Save(cfg); err != nil {
			return formatSkillError("CONFIG_SAVE_FAILED", err, "verify write permissions for ~/.kafclaw")
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Skills system disabled.")
		return nil
	},
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bundled and configured skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		payload := buildSkillsStatus(cfg)

		if skillsListJSON {
			data, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Skills enabled: %v\n", payload.SkillsEnabled)
		fmt.Fprintln(cmd.OutOrStdout(), "Columns: name | source | state | eligible | missing")
		for _, row := range payload.Rows {
			state := "disabled"
			if row.Enabled {
				state = "enabled"
			}
			eligible := "yes"
			if !row.Eligible {
				eligible = "no"
			}
			missing := "-"
			if len(row.Missing) > 0 {
				missing = strings.Join(row.Missing, ", ")
			}
			override := ""
			if row.Configured {
				override = " [override]"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "- %s | %s | %s%s | %s | %s\n", row.Name, row.Source, state, override, eligible, missing)
		}
		return nil
	},
}

var skillsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show skills readiness summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		payload := buildSkillsStatus(cfg)
		if skillsStatusJSON {
			data, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Skills enabled: %v\n", payload.SkillsEnabled)
		fmt.Fprintf(cmd.OutOrStdout(), "Scope: %s\n", payload.Scope)
		fmt.Fprintf(cmd.OutOrStdout(), "Runtime isolation: %s\n", payload.RuntimeIsolation)
		fmt.Fprintf(cmd.OutOrStdout(), "Node found: %v\n", payload.NodeFound)
		fmt.Fprintf(cmd.OutOrStdout(), "Clawhub found: %v\n", payload.ClawhubFound)
		ready := 0
		for _, row := range payload.Rows {
			if row.Eligible {
				ready++
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Eligible skills: %d/%d\n", ready, len(payload.Rows))
		return nil
	},
}

var skillsEnableSkillCmd = &cobra.Command{
	Use:   "enable-skill <name>",
	Short: "Enable one skill by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return formatSkillError("SKILL_NAME_REQUIRED", fmt.Errorf("skill name is required"), "provide a non-empty skill name")
		}
		if err := setSkillEnabled(name, true); err != nil {
			return formatSkillError("SKILL_ENABLE_FAILED", err, "check config write permissions")
		}
		if name == "google-cli" {
			check, err := skills.CheckPrerequisite("google-cli")
			if err == nil && !check.Installed {
				fmt.Fprintln(cmd.OutOrStdout(), "Prerequisite missing: google-cli (gcloud). Run: kafclaw skills prereq install google-cli --yes")
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Skill enabled: %s\n", name)
		return nil
	},
}

var skillsDisableSkillCmd = &cobra.Command{
	Use:   "disable-skill <name>",
	Short: "Disable one skill by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(args[0])
		if name == "" {
			return formatSkillError("SKILL_NAME_REQUIRED", fmt.Errorf("skill name is required"), "provide a non-empty skill name")
		}
		if err := setSkillEnabled(name, false); err != nil {
			return formatSkillError("SKILL_DISABLE_FAILED", err, "check config write permissions")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Skill disabled: %s\n", name)
		return nil
	},
}

var skillsVerifyCmd = &cobra.Command{
	Use:   "verify <path-or-url>",
	Short: "Verify a skill package/source before install",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])
		if target == "" {
			return formatSkillError("TARGET_REQUIRED", fmt.Errorf("target is required"), "pass a skill path, URL, or slug")
		}
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		report, err := skills.VerifySkillSource(cfg, target)
		if err != nil {
			return formatSkillError("VERIFY_FAILED", err, "run `kafclaw skills verify --json <target>` for detailed findings")
		}
		if skillsVerifyJSON {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Skill: %s\n", report.SkillName)
			fmt.Fprintf(cmd.OutOrStdout(), "Source: %s (%s)\n", report.ResolvedTarget, report.SourceType)
			fmt.Fprintf(cmd.OutOrStdout(), "Files: %d, Links: %d\n", report.FileCount, report.LinkCount)
			for _, f := range report.Findings {
				if f.File != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "- [%s] %s (%s) in %s\n", f.Severity, f.Code, f.Message, f.File)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "- [%s] %s (%s)\n", f.Severity, f.Code, f.Message)
				}
			}
			if len(report.Findings) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No findings.")
			}
		}
		if !report.OK {
			return formatSkillError("VERIFY_CRITICAL", fmt.Errorf("verification failed with %d critical finding(s)", report.CriticalCount()), "fix critical findings before install/update")
		}
		return nil
	},
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install <slug-or-url>",
	Short: "Install a skill from clawhub slug or URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.TrimSpace(args[0])
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		res, err := skills.InstallSkill(cfg, target, skillsApproveWarnings)
		if err != nil {
			return formatSkillError("INSTALL_FAILED", err, "run `kafclaw skills verify <target>` and retry with --approve-warnings if intended")
		}
		if skillsInstallJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		action := "Installed"
		if res.Updated {
			action = "Updated"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s skill: %s\n", action, res.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "Path: %s\n", res.InstallPath)
		if res.WarningCount > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Warnings: %d (approved)\n", res.WarningCount)
		}
		return nil
	},
}

var skillsUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update one skill or all skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = strings.TrimSpace(args[0])
		}
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		results, err := skills.UpdateSkills(cfg, name, skillsApproveWarnings)
		if err != nil {
			return formatSkillError("UPDATE_FAILED", err, "run `kafclaw skills list` and verify installed skill metadata")
		}
		if skillsUpdateJSON {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		for _, res := range results {
			fmt.Fprintf(cmd.OutOrStdout(), "Updated skill: %s (%s)\n", res.Name, res.InstallPath)
		}
		return nil
	},
}

var skillsPrereqCheckCmd = &cobra.Command{
	Use:   "check <name>",
	Short: "Check if a prerequisite is installed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := skills.CheckPrerequisite(args[0])
		if err != nil {
			return formatSkillError("PREREQ_CHECK_FAILED", err, "use one of the supported names (e.g. google-cli)")
		}
		if skillsPrereqJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		state := "missing"
		if res.Installed {
			state = "installed"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", res.Name, state)
		return nil
	},
}

var skillsPrereqInstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install a prerequisite using Go-native routines",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !skillsPrereqDryRun && !skillsPrereqYes {
			return formatSkillError("PREREQ_CONFIRM_REQUIRED", fmt.Errorf("prerequisite installs require explicit confirmation"), "pass --yes (or use --dry-run)")
		}
		res, err := skills.InstallPrerequisite(args[0], skillsPrereqDryRun)
		if err != nil {
			return formatSkillError("PREREQ_INSTALL_FAILED", err, "rerun with --dry-run first to review install steps")
		}
		if skillsPrereqJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		if skillsPrereqDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "Dry-run install plan for %s:\n", res.Name)
			for _, m := range res.Messages {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", m)
			}
			return nil
		}
		state := "missing"
		if res.Installed {
			state = "installed"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", res.Name, state)
		return nil
	},
}

var skillsAuthStartCmd = &cobra.Command{
	Use:   "start <provider>",
	Short: "Start headless OAuth flow for provider (google-workspace|m365)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		provider := skills.OAuthProvider(strings.TrimSpace(args[0]))
		resolvedScopes, err := resolveOAuthScopes(cfg, provider, skillsAuthScopes, skillsAuthAccess)
		if err != nil {
			return formatSkillError("AUTH_SCOPE_RESOLVE_FAILED", err, "set --access (mail,calendar,drive/files,all) or explicit --scopes")
		}
		input := skills.OAuthStartInput{
			Provider:     provider,
			Profile:      skillsAuthProfile,
			ClientID:     skillsAuthClientID,
			ClientSecret: skillsAuthClientSecret,
			RedirectURI:  skillsAuthRedirectURI,
			Scopes:       resolvedScopes,
			TenantID:     skillsAuthTenantID,
		}
		res, err := skills.StartOAuthFlow(input)
		if err != nil {
			return formatSkillError("AUTH_START_FAILED", err, "validate client-id/client-secret/redirect-uri and try again")
		}
		if skillsAuthJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\n", res.Provider)
		fmt.Fprintf(cmd.OutOrStdout(), "Profile: %s\n", res.Profile)
		fmt.Fprintf(cmd.OutOrStdout(), "State: %s\n", res.State)
		fmt.Fprintln(cmd.OutOrStdout(), "Open this URL, authenticate, then run `kafclaw skills auth complete ...` with code + state:")
		fmt.Fprintln(cmd.OutOrStdout(), res.AuthorizeURL)
		return nil
	},
}

var skillsAuthCompleteCmd = &cobra.Command{
	Use:   "complete <provider>",
	Short: "Complete headless OAuth flow and store tokens securely",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := strings.TrimSpace(skillsAuthCode)
		state := strings.TrimSpace(skillsAuthState)
		if strings.TrimSpace(skillsAuthCallbackURL) != "" {
			parsedCode, parsedState, err := skills.ParseOAuthCallbackURL(skillsAuthCallbackURL)
			if err != nil {
				return formatSkillError("AUTH_CALLBACK_PARSE_FAILED", err, "pass --code and --state directly if callback URL parsing fails")
			}
			if code == "" {
				code = parsedCode
			}
			if state == "" {
				state = parsedState
			}
		}
		res, err := skills.CompleteOAuthFlow(skills.OAuthCompleteInput{
			Provider: skills.OAuthProvider(strings.TrimSpace(args[0])),
			Profile:  skillsAuthProfile,
			Code:     code,
			State:    state,
		})
		if err != nil {
			return formatSkillError("AUTH_COMPLETE_FAILED", err, "ensure state matches `skills auth start` output")
		}
		if skillsAuthJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Token stored for %s/%s\n", res.Provider, res.Profile)
		fmt.Fprintf(cmd.OutOrStdout(), "Path: %s\n", res.TokenPath)
		if !res.ExpiresAt.IsZero() {
			fmt.Fprintf(cmd.OutOrStdout(), "Expires: %s\n", res.ExpiresAt.Format(time.RFC3339))
		}
		return nil
	},
}

var skillsExecCmd = &cobra.Command{
	Use:   "exec <skill> <command> [args...]",
	Short: "Execute an installed skill command with runtime policy enforcement",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return formatSkillError("CONFIG_LOAD_FAILED", err, "check ~/.kafclaw/config.json")
		}
		res, err := skills.ExecuteSkillCommand(cfg, strings.TrimSpace(args[0]), append([]string{}, args[1:]...))
		if skillsExecJSON {
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
		}
		if res != nil && !skillsExecJSON {
			if res.Stdout != "" {
				fmt.Fprint(cmd.OutOrStdout(), res.Stdout)
				if !strings.HasSuffix(res.Stdout, "\n") {
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			if res.Stderr != "" {
				fmt.Fprint(cmd.OutOrStdout(), res.Stderr)
				if !strings.HasSuffix(res.Stderr, "\n") {
					fmt.Fprintln(cmd.OutOrStdout())
				}
			}
			if res.OutputTruncated {
				fmt.Fprintln(cmd.OutOrStdout(), "[output truncated by policy]")
			}
		}
		if err != nil {
			return formatSkillError("EXEC_FAILED", err, "check SKILL-POLICY.json and command allow/deny rules")
		}
		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsEnableCmd)
	skillsCmd.AddCommand(skillsDisableCmd)
	skillsCmd.AddCommand(skillsListCmd)
	skillsCmd.AddCommand(skillsStatusCmd)
	skillsCmd.AddCommand(skillsEnableSkillCmd)
	skillsCmd.AddCommand(skillsDisableSkillCmd)
	skillsCmd.AddCommand(skillsVerifyCmd)
	skillsCmd.AddCommand(skillsInstallCmd)
	skillsCmd.AddCommand(skillsUpdateCmd)
	skillsCmd.AddCommand(skillsPrereqCmd)
	skillsCmd.AddCommand(skillsAuthCmd)
	skillsCmd.AddCommand(skillsExecCmd)

	skillsPrereqCmd.AddCommand(skillsPrereqCheckCmd)
	skillsPrereqCmd.AddCommand(skillsPrereqInstallCmd)
	skillsAuthCmd.AddCommand(skillsAuthStartCmd)
	skillsAuthCmd.AddCommand(skillsAuthCompleteCmd)

	skillsListCmd.Flags().BoolVar(&skillsListJSON, "json", false, "Output JSON")
	skillsStatusCmd.Flags().BoolVar(&skillsStatusJSON, "json", false, "Output JSON")
	skillsEnableCmd.Flags().BoolVar(&skillsEnableInstallHub, "install-clawhub", true, "Install clawhub via npm if missing")
	skillsEnableCmd.Flags().StringVar(&skillsEnableNodeMajor, "node-major", "20", "Node major version to pin in .nvmrc when missing")
	skillsVerifyCmd.Flags().BoolVar(&skillsVerifyJSON, "json", false, "Output JSON")
	skillsInstallCmd.Flags().BoolVar(&skillsInstallJSON, "json", false, "Output JSON")
	skillsUpdateCmd.Flags().BoolVar(&skillsUpdateJSON, "json", false, "Output JSON")
	skillsInstallCmd.Flags().BoolVar(&skillsApproveWarnings, "approve-warnings", false, "Allow install when verify emits warnings")
	skillsUpdateCmd.Flags().BoolVar(&skillsApproveWarnings, "approve-warnings", false, "Allow update when verify emits warnings")
	skillsPrereqInstallCmd.Flags().BoolVar(&skillsPrereqDryRun, "dry-run", false, "Show install plan without running commands")
	skillsPrereqInstallCmd.Flags().BoolVar(&skillsPrereqYes, "yes", false, "Confirm running install routine")
	skillsPrereqCheckCmd.Flags().BoolVar(&skillsPrereqJSON, "json", false, "Output JSON")
	skillsPrereqInstallCmd.Flags().BoolVar(&skillsPrereqJSON, "json", false, "Output JSON")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthProfile, "profile", "default", "Credential profile name")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthClientID, "client-id", "", "OAuth client ID")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthClientSecret, "client-secret", "", "OAuth client secret")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthRedirectURI, "redirect-uri", "http://localhost:53682/callback", "OAuth redirect URI")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthScopes, "scopes", "", "Comma-separated OAuth scopes")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthAccess, "access", "", "Capability preset list (google: mail,calendar,drive,all; m365: mail,calendar,files,all)")
	skillsAuthStartCmd.Flags().StringVar(&skillsAuthTenantID, "tenant-id", "", "M365 tenant ID (optional, defaults common)")
	skillsAuthStartCmd.Flags().BoolVar(&skillsAuthJSON, "json", false, "Output JSON")
	skillsAuthCompleteCmd.Flags().StringVar(&skillsAuthProfile, "profile", "default", "Credential profile name")
	skillsAuthCompleteCmd.Flags().StringVar(&skillsAuthCode, "code", "", "OAuth authorization code")
	skillsAuthCompleteCmd.Flags().StringVar(&skillsAuthState, "state", "", "OAuth state returned by provider")
	skillsAuthCompleteCmd.Flags().StringVar(&skillsAuthCallbackURL, "callback-url", "", "Full callback URL containing code+state")
	skillsAuthCompleteCmd.Flags().BoolVar(&skillsAuthJSON, "json", false, "Output JSON")
	skillsExecCmd.Flags().BoolVar(&skillsExecJSON, "json", false, "Output JSON")
	_ = skillsAuthStartCmd.MarkFlagRequired("client-id")
	_ = skillsAuthStartCmd.MarkFlagRequired("client-secret")

	rootCmd.AddCommand(skillsCmd)
}

func setSkillEnabled(skillName string, enabled bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Skills.Entries == nil {
		cfg.Skills.Entries = map[string]config.SkillEntryConfig{}
	}
	cfg.Skills.Entries[skillName] = config.SkillEntryConfig{Enabled: enabled}
	return config.Save(cfg)
}

func buildSkillsStatus(cfg *config.Config) skillsStatusPayload {
	rows := make([]skillListRow, 0, len(skills.BundledCatalog))
	bundledSet := map[string]struct{}{}
	nodeFound := skills.HasBinary("node")
	clawhubFound := skills.HasBinary("clawhub")
	for _, s := range skills.BundledCatalog {
		bundledSet[s.Name] = struct{}{}
		_, configured := cfg.Skills.Entries[s.Name]
		enabled := skills.EffectiveSkillEnabled(cfg, s.Name)
		missing := skillMissingRequirements(cfg, s.Name)
		rows = append(rows, skillListRow{
			Name:       s.Name,
			Source:     "bundled",
			Enabled:    enabled,
			Configured: configured,
			Eligible:   len(missing) == 0,
			Missing:    missing,
		})
	}
	for name := range cfg.Skills.Entries {
		if _, ok := bundledSet[name]; ok {
			continue
		}
		missing := skillMissingRequirements(cfg, name)
		rows = append(rows, skillListRow{
			Name:       name,
			Source:     "configured",
			Enabled:    skills.EffectiveSkillEnabled(cfg, name),
			Configured: true,
			Eligible:   len(missing) == 0,
			Missing:    missing,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return skillsStatusPayload{
		SkillsEnabled:    cfg.Skills.Enabled,
		Scope:            cfg.Skills.Scope,
		RuntimeIsolation: cfg.Skills.RuntimeIsolation,
		NodeFound:        nodeFound,
		ClawhubFound:     clawhubFound,
		Rows:             rows,
	}
}

func skillMissingRequirements(cfg *config.Config, name string) []string {
	missing := make([]string, 0)
	if !cfg.Skills.Enabled {
		missing = append(missing, "skills system disabled")
	}
	if !skills.HasBinary("node") {
		missing = append(missing, "node missing")
	}
	if cfg.Skills.ExternalInstalls && !skills.HasBinary("clawhub") {
		missing = append(missing, "clawhub missing")
	}
	if name == "google-cli" && !skills.HasBinary("gcloud") {
		missing = append(missing, "gcloud missing")
	}
	return missing
}

func resolveOAuthScopes(cfg *config.Config, provider skills.OAuthProvider, explicitScopes string, accessSelection string) ([]string, error) {
	if scopes := parseScopeCSV(explicitScopes); len(scopes) > 0 {
		return scopes, nil
	}
	access := strings.TrimSpace(accessSelection)
	if access == "" && cfg != nil {
		if entry, ok := cfg.Skills.Entries[strings.TrimSpace(string(provider))]; ok && len(entry.Capabilities) > 0 {
			access = strings.Join(entry.Capabilities, ",")
		}
	}
	switch provider {
	case skills.ProviderGoogleWorkspace:
		return googleWorkspaceScopesForAccess(access)
	case skills.ProviderM365:
		return m365ScopesForAccess(access)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func parseScopeCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
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

func googleWorkspaceScopesForAccess(access string) ([]string, error) {
	base := []string{"openid", "email", "profile"}
	selection := strings.TrimSpace(access)
	if selection == "" {
		return append(base, "https://www.googleapis.com/auth/userinfo.email"), nil
	}
	parts, err := parseCapabilitySelection(selection, map[string]struct{}{
		"mail": {}, "calendar": {}, "drive": {}, "all": {},
	})
	if err != nil {
		return nil, err
	}
	all := map[string]struct{}{}
	for _, p := range parts {
		switch p {
		case "all":
			all["mail"] = struct{}{}
			all["calendar"] = struct{}{}
			all["drive"] = struct{}{}
		default:
			all[p] = struct{}{}
		}
	}
	if _, ok := all["mail"]; ok {
		base = append(base, "https://www.googleapis.com/auth/gmail.readonly")
	}
	if _, ok := all["calendar"]; ok {
		base = append(base, "https://www.googleapis.com/auth/calendar.readonly")
	}
	if _, ok := all["drive"]; ok {
		base = append(base, "https://www.googleapis.com/auth/drive.readonly")
	}
	if len(all) == 0 {
		base = append(base, "https://www.googleapis.com/auth/userinfo.email")
	}
	return base, nil
}

func m365ScopesForAccess(access string) ([]string, error) {
	base := []string{"openid", "offline_access", "User.Read"}
	selection := strings.TrimSpace(access)
	if selection == "" {
		return base, nil
	}
	parts, err := parseCapabilitySelection(selection, map[string]struct{}{
		"mail": {}, "calendar": {}, "files": {}, "all": {},
	})
	if err != nil {
		return nil, err
	}
	all := map[string]struct{}{}
	for _, p := range parts {
		switch p {
		case "all":
			all["mail"] = struct{}{}
			all["calendar"] = struct{}{}
			all["files"] = struct{}{}
		default:
			all[p] = struct{}{}
		}
	}
	if _, ok := all["mail"]; ok {
		base = append(base, "Mail.Read")
	}
	if _, ok := all["calendar"]; ok {
		base = append(base, "Calendars.Read")
	}
	if _, ok := all["files"]; ok {
		base = append(base, "Files.Read")
	}
	return base, nil
}

func formatSkillError(code string, err error, remediation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[%s] %v. remediation: %s", strings.ToUpper(strings.TrimSpace(code)), err, remediation)
}
