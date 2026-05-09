package cmd

import (
	"fmt"
	"os"

	"github.com/danlafeir/cli-go/pkg/secrets"
	"github.com/danlafeir/timecard/cmd/timecard"
	"github.com/danlafeir/timecard/internal/selfupdate"
	"github.com/spf13/cobra"
)

// Build metadata, populated by main.go from -ldflags-injected values.
var (
	BuildVersion string
	BuildGitHash string
	BuildDate    string
)

var rootCmd = &cobra.Command{
	Use:          "timecard",
	Short:        "commands to manage your timecard",
	SilenceUsage: true,
}

// updateCmd downloads and installs the latest GitHub Release of timecard.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update timecard to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		selfupdate.Run(BuildVersion, cmd)
	},
}

// versionCmd prints build metadata.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print timecard version, commit, and build date",
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Println(versionString())
	},
}

func versionString() string {
	return fmt.Sprintf("timecard %s (commit %s, built %s)", BuildVersion, BuildGitHash, BuildDate)
}

func Execute() {
	rootCmd.Version = versionString()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	secrets.SetDefaultProvider("timecard")

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	rootCmd.AddCommand(timecard.AddEntryCmd())
	rootCmd.AddCommand(timecard.ConfigureCmd())
	rootCmd.AddCommand(timecard.GetWeekCmd())
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}
