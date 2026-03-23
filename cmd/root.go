package cmd

import (
	"os"

	"github.com/danlafeir/cli-go/pkg/secrets"
	"github.com/danlafeir/timecard/cmd/timecard"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Version:      "1.0",
	Use:          "timecard",
	Short:        "commands to manage your timecard",
	SilenceUsage: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var completionCmd = &cobra.Command{
	Use:    "completion [bash|zsh|fish|powershell]",
	Short:  "Generate shell completion scripts",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Println("Please specify a shell: bash, zsh, fish, or powershell")
			os.Exit(1)
		}
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			cmd.Println("Unsupported shell type.")
			os.Exit(1)
		}
	},
}

func init() {
	secrets.SetDefaultProvider("timecard")

	// Hide the help command
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	// Disable the completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Add commands
	rootCmd.AddCommand(timecard.AddEntryCmd())
	rootCmd.AddCommand(timecard.ConfigureCmd())
	rootCmd.AddCommand(timecard.GetWeekCmd())

	// Hide completion command if it was already registered
	if compCmd, _, _ := rootCmd.Find([]string{"completion"}); compCmd != nil {
		compCmd.Hidden = true
	}
}
