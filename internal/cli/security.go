package cli

import (
	"fmt"
	"strings"

	"github.com/KafClaw/KafClaw/internal/cliconfig"
	"github.com/spf13/cobra"
)

var securityJSON bool
var securityAuditDeep bool
var securityFixYes bool

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Run security checks, deep audits, and safe remediations",
}

var securityCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Run fast security checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := cliconfig.RunSecurityCheck()
		if err != nil {
			return formatSecurityError("CHECK_FAILED", err, "rerun with `kafclaw security check --json` for details")
		}
		return printSecurityReport(cmd, report)
	},
}

var securityAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Run security audit (optionally deep)",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := cliconfig.RunSecurityAudit(cliconfig.SecurityAuditOptions{Deep: securityAuditDeep})
		if err != nil {
			return formatSecurityError("AUDIT_FAILED", err, "rerun with `kafclaw security audit --deep --json`")
		}
		return printSecurityReport(cmd, report)
	},
}

var securityFixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Apply safe security remediations",
	RunE: func(cmd *cobra.Command, args []string) error {
		report, err := cliconfig.RunSecurityFix(cliconfig.SecurityFixOptions{Yes: securityFixYes})
		if err != nil {
			return formatSecurityError("FIX_FAILED", err, "rerun with `kafclaw security fix --yes` to confirm remediation")
		}
		return printSecurityReport(cmd, report)
	},
}

func init() {
	securityCmd.AddCommand(securityCheckCmd)
	securityCmd.AddCommand(securityAuditCmd)
	securityCmd.AddCommand(securityFixCmd)

	securityCheckCmd.Flags().BoolVar(&securityJSON, "json", false, "Output JSON")
	securityAuditCmd.Flags().BoolVar(&securityJSON, "json", false, "Output JSON")
	securityAuditCmd.Flags().BoolVar(&securityAuditDeep, "deep", false, "Run deep audit (verifies installed skills)")
	securityFixCmd.Flags().BoolVar(&securityJSON, "json", false, "Output JSON")
	securityFixCmd.Flags().BoolVar(&securityFixYes, "yes", false, "Confirm applying security remediations")

	rootCmd.AddCommand(securityCmd)
}

func printSecurityReport(cmd *cobra.Command, report cliconfig.SecurityReport) error {
	if securityJSON {
		fmt.Fprintln(cmd.OutOrStdout(), cliconfig.MarshalSecurityReport(report))
	} else {
		failures := 0
		for _, check := range report.Checks {
			symbol := "PASS"
			if check.Status == cliconfig.SecurityWarn {
				symbol = "WARN"
			}
			if check.Status == cliconfig.SecurityFail {
				symbol = "FAIL"
				failures++
			}
			fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", symbol, check.Name, check.Message)
		}
		if failures > 0 {
			return fmt.Errorf("security reported %d failing check(s)", failures)
		}
	}
	if report.HasFailures() {
		return fmt.Errorf("security reported failing checks")
	}
	return nil
}

func formatSecurityError(code string, err error, remediation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("[%s] %v. remediation: %s", strings.ToUpper(strings.TrimSpace(code)), err, remediation)
}
