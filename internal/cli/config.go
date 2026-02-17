package cli

import (
	"encoding/json"
	"fmt"

	"github.com/KafClaw/KafClaw/internal/cliconfig"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage KafClaw configuration values",
}

var configGetCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get effective config value by dotted path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val, err := cliconfig.Get(args[0])
		if err != nil {
			return err
		}
		switch v := val.(type) {
		case map[string]any, []any:
			out, _ := json.MarshalIndent(v, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		default:
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <path> <value>",
	Short: "Set config value by dotted path (JSON or plain string)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cliconfig.Set(args[0], args[1])
	},
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <path>",
	Short: "Unset config value by dotted path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cliconfig.Unset(args[0])
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	rootCmd.AddCommand(configCmd)
}
