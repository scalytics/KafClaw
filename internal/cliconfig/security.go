package cliconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KafClaw/KafClaw/internal/config"
	skillruntime "github.com/KafClaw/KafClaw/internal/skills"
)

type SecurityStatus string

const (
	SecurityPass SecurityStatus = "pass"
	SecurityWarn SecurityStatus = "warn"
	SecurityFail SecurityStatus = "fail"
)

type SecurityCheck struct {
	Name    string         `json:"name"`
	Status  SecurityStatus `json:"status"`
	Message string         `json:"message"`
}

type SecurityReport struct {
	Mode    string          `json:"mode"`
	Checks  []SecurityCheck `json:"checks"`
	Created time.Time       `json:"created"`
}

type SecurityAuditOptions struct {
	Deep bool
}

type SecurityFixOptions struct {
	Yes bool
}

func (r SecurityReport) HasFailures() bool {
	for _, c := range r.Checks {
		if c.Status == SecurityFail {
			return true
		}
	}
	return false
}

func RunSecurityCheck() (SecurityReport, error) {
	return runSecurity(SecurityAuditOptions{Deep: false})
}

func RunSecurityAudit(opts SecurityAuditOptions) (SecurityReport, error) {
	return runSecurity(opts)
}

func RunSecurityFix(opts SecurityFixOptions) (SecurityReport, error) {
	if !opts.Yes {
		return SecurityReport{}, fmt.Errorf("security fix requires explicit confirmation (--yes)")
	}
	report, err := runSecurityFix()
	if err != nil {
		return report, err
	}
	return report, nil
}

func runSecurity(opts SecurityAuditOptions) (SecurityReport, error) {
	report := SecurityReport{
		Mode:    "check",
		Checks:  make([]SecurityCheck, 0, 12),
		Created: time.Now().UTC(),
	}
	if opts.Deep {
		report.Mode = "audit-deep"
	}

	cfg, err := config.Load()
	if err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "config_load",
			Status:  SecurityFail,
			Message: fmt.Sprintf("failed to load config: %v", err),
		})
		return report, nil
	}

	doctorReport, err := RunDoctorWithOptions(DoctorOptions{})
	if err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "doctor_run",
			Status:  SecurityFail,
			Message: fmt.Sprintf("doctor run failed: %v", err),
		})
	} else {
		for _, c := range doctorReport.Checks {
			status := SecurityPass
			if c.Status == DoctorWarn {
				status = SecurityWarn
			}
			if c.Status == DoctorFail {
				status = SecurityFail
			}
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "doctor:" + c.Name,
				Status:  status,
				Message: c.Message,
			})
		}
	}

	appendGapChecks(cfg, &report)
	if opts.Deep {
		appendDeepSkillAuditChecks(cfg, &report)
	}
	return report, nil
}

func runSecurityFix() (SecurityReport, error) {
	report := SecurityReport{
		Mode:    "fix",
		Checks:  make([]SecurityCheck, 0, 16),
		Created: time.Now().UTC(),
	}

	cfg, err := config.Load()
	if err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "config_load",
			Status:  SecurityFail,
			Message: fmt.Sprintf("failed to load config: %v", err),
		})
		return report, nil
	}

	if _, err := RunDoctorWithOptions(DoctorOptions{Fix: true}); err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "doctor_fix",
			Status:  SecurityFail,
			Message: fmt.Sprintf("doctor --fix failed: %v", err),
		})
	} else {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "doctor_fix",
			Status:  SecurityPass,
			Message: "doctor --fix remediations applied",
		})
	}

	changed := false
	if !cfg.Skills.Enabled {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "skills_policy_defaults",
			Status:  SecurityPass,
			Message: "skills disabled; no policy default changes needed",
		})
	} else {
		mode := strings.ToLower(strings.TrimSpace(cfg.Skills.LinkPolicy.Mode))
		switch mode {
		case "allowlist", "denylist", "open":
		default:
			cfg.Skills.LinkPolicy.Mode = "allowlist"
			changed = true
		}
		if strings.EqualFold(cfg.Skills.LinkPolicy.Mode, "allowlist") && len(cfg.Skills.LinkPolicy.AllowDomains) == 0 {
			cfg.Skills.LinkPolicy.AllowDomains = []string{"clawhub.ai"}
			changed = true
		}
		if cfg.Skills.LinkPolicy.AllowHTTP {
			cfg.Skills.LinkPolicy.AllowHTTP = false
			changed = true
		}
		if cfg.Skills.LinkPolicy.MaxLinksPerSkill <= 0 {
			cfg.Skills.LinkPolicy.MaxLinksPerSkill = 20
			changed = true
		}
		if !strings.EqualFold(strings.TrimSpace(cfg.Skills.Scope), "selected") {
			cfg.Skills.Scope = "selected"
			changed = true
		}
		if !strings.EqualFold(strings.TrimSpace(cfg.Skills.RuntimeIsolation), "strict") {
			cfg.Skills.RuntimeIsolation = "strict"
			changed = true
		}
		if changed {
			if err := config.Save(cfg); err != nil {
				report.Checks = append(report.Checks, SecurityCheck{
					Name:    "skills_policy_defaults",
					Status:  SecurityFail,
					Message: fmt.Sprintf("failed to save security defaults: %v", err),
				})
			} else {
				report.Checks = append(report.Checks, SecurityCheck{
					Name:    "skills_policy_defaults",
					Status:  SecurityPass,
					Message: "applied secure skills defaults (allowlist, https-only, link cap, scope=selected, runtimeIsolation=strict)",
				})
			}
		} else {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_policy_defaults",
				Status:  SecurityPass,
				Message: "skills policy defaults already secure",
			})
		}
	}

	post, err := runSecurity(SecurityAuditOptions{Deep: false})
	if err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "post_fix_check",
			Status:  SecurityFail,
			Message: fmt.Sprintf("post-fix check failed: %v", err),
		})
		return report, nil
	}
	report.Checks = append(report.Checks, post.Checks...)
	return report, nil
}

func appendGapChecks(cfg *config.Config, report *SecurityReport) {
	if cfg == nil || report == nil {
		return
	}
	if cfg.Skills.Enabled {
		mode := strings.ToLower(strings.TrimSpace(cfg.Skills.RuntimeIsolation))
		if mode != "strict" {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "gap_runtime_isolation",
				Status:  SecurityWarn,
				Message: "skills runtime isolation is not strict; set `skills.runtimeIsolation` to `strict` to require container isolation",
			})
		} else {
			if runtimeBin, err := skillruntime.StrictIsolationPreflight(); err != nil {
				report.Checks = append(report.Checks, SecurityCheck{
					Name:    "gap_runtime_isolation",
					Status:  SecurityFail,
					Message: fmt.Sprintf("strict mode configured but runtime preflight failed: %v", err),
				})
			} else {
				report.Checks = append(report.Checks, SecurityCheck{
					Name:    "gap_runtime_isolation",
					Status:  SecurityPass,
					Message: fmt.Sprintf("strict container isolation is enabled and runtime `%s` is usable", runtimeBin),
				})
			}
		}
	}
	if cfg.Skills.Enabled && cfg.Skills.AllowSystemRepoSkills && strings.EqualFold(strings.TrimSpace(cfg.Skills.Scope), "all") {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "gap_prompt_skill_scope",
			Status:  SecurityWarn,
			Message: "system repo skill prompt loading is set to scope=all; switch to scope=selected for least privilege",
		})
	} else if cfg.Skills.Enabled && cfg.Skills.AllowSystemRepoSkills {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "gap_prompt_skill_scope",
			Status:  SecurityPass,
			Message: "system repo skill prompt loading is constrained by selected-skill scope",
		})
	}
	tokenCount, encryptedCount := countOAuthTokenFiles()
	if tokenCount > 0 {
		if encryptedCount == tokenCount {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "gap_oauth_encryption",
				Status:  SecurityPass,
				Message: fmt.Sprintf("oauth token files are encrypted at rest (%d/%d)", encryptedCount, tokenCount),
			})
		} else {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "gap_oauth_encryption",
				Status:  SecurityWarn,
				Message: fmt.Sprintf("oauth token encryption coverage is partial (%d/%d encrypted); re-auth to rotate old plaintext files", encryptedCount, tokenCount),
			})
		}
	}
	pinnableCount, pinnedCount := countPinnedExternalSkills()
	if pinnableCount > 0 {
		if pinnedCount == pinnableCount {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "gap_skill_hash_pinning",
				Status:  SecurityPass,
				Message: fmt.Sprintf("external skill hash pinning coverage complete (%d/%d)", pinnedCount, pinnableCount),
			})
		} else {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "gap_skill_hash_pinning",
				Status:  SecurityWarn,
				Message: fmt.Sprintf("external skill hash pinning is partial (%d/%d); use source URLs with #sha256=<digest>", pinnedCount, pinnableCount),
			})
		}
	}
}

func appendDeepSkillAuditChecks(cfg *config.Config, report *SecurityReport) {
	if cfg == nil || report == nil {
		return
	}
	dirs, err := skillruntime.ResolveStateDirs()
	if err != nil {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "skills_verify_installed",
			Status:  SecurityFail,
			Message: fmt.Sprintf("failed to resolve state dirs: %v", err),
		})
		return
	}
	entries, err := os.ReadDir(dirs.Installed)
	if err != nil {
		if os.IsNotExist(err) {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_verify_installed",
				Status:  SecurityPass,
				Message: "no installed skills to audit",
			})
			return
		}
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "skills_verify_installed",
			Status:  SecurityFail,
			Message: fmt.Sprintf("failed reading installed skills: %v", err),
		})
		return
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		report.Checks = append(report.Checks, SecurityCheck{
			Name:    "skills_verify_installed",
			Status:  SecurityPass,
			Message: "no installed skills to audit",
		})
		return
	}

	for _, name := range names {
		p := filepath.Join(dirs.Installed, name)
		verify, err := skillruntime.VerifySkillSource(cfg, p)
		if err != nil {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_verify:" + name,
				Status:  SecurityFail,
				Message: fmt.Sprintf("verify failed: %v", err),
			})
			continue
		}
		if verify.CriticalCount() > 0 {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_verify:" + name,
				Status:  SecurityFail,
				Message: fmt.Sprintf("critical findings=%d warnings=%d", verify.CriticalCount(), verify.WarningCount()),
			})
		} else if verify.WarningCount() > 0 {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_verify:" + name,
				Status:  SecurityWarn,
				Message: fmt.Sprintf("warnings=%d", verify.WarningCount()),
			})
		} else {
			report.Checks = append(report.Checks, SecurityCheck{
				Name:    "skills_verify:" + name,
				Status:  SecurityPass,
				Message: "no findings",
			})
		}
	}
}

func countOAuthTokenFiles() (total int, encrypted int) {
	dirs, err := skillruntime.ResolveStateDirs()
	if err != nil {
		return 0, 0
	}
	authRoot := filepath.Join(dirs.ToolsDir, "auth")
	total = 0
	encrypted = 0
	_ = filepath.WalkDir(authRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, string(os.PathSeparator)+"token.json") {
			total++
			if isEncryptedOAuthBlobFile(path) {
				encrypted++
			}
		}
		return nil
	})
	return total, encrypted
}

func isEncryptedOAuthBlobFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return false
	}
	_, hasCiphertext := obj["ciphertext"]
	_, hasNonce := obj["nonce"]
	_, hasVersion := obj["version"]
	_, hasAccessToken := obj["access_token"]
	return hasCiphertext && hasNonce && hasVersion && !hasAccessToken
}

func MarshalSecurityReport(r SecurityReport) string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

func countPinnedExternalSkills() (pinnable int, pinned int) {
	dirs, err := skillruntime.ResolveStateDirs()
	if err != nil {
		return 0, 0
	}
	entries, err := os.ReadDir(dirs.Installed)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dirs.Installed, e.Name(), ".kafclaw-skill.json"))
		if err != nil {
			continue
		}
		var meta struct {
			Source     string `json:"source"`
			PinnedHash string `json:"pinnedHash"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		source := strings.ToLower(strings.TrimSpace(meta.Source))
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			pinnable++
			if strings.TrimSpace(meta.PinnedHash) != "" {
				pinned++
			}
		}
	}
	return pinnable, pinned
}
