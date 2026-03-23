package timecard

import (
	"fmt"
	"os"

	"github.com/danlafeir/timecard/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ConfigureCmd() *cobra.Command {
	var apiToken string
	var accountId string

	configureCmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure integration with timesheet tool (currently just Tempo)",
		Run: func(cmd *cobra.Command, args []string) {
			initConfig()
			viper.ReadInConfig()

			configureApiToken(apiToken)
			configureAccountId(accountId)
			// Get accountId from viper after it's been set
			configuredAccountId := viper.GetString("tempo." + ACCOUNT_ID_CONFIG)
			if configuredAccountId == "" {
				configuredAccountId = accountId
			}
			configureIssueId(configuredAccountId)

			if err := viper.WriteConfig(); err != nil {
				fmt.Println("Failed to save config:", err)
				os.Exit(1)
			}
			fmt.Println("Configuration saved successfully.")
		},
	}
	configureCmd.Flags().StringVar(&apiToken, "token", "", "Tempo API token")
	configureCmd.Flags().StringVar(&accountId, "account-id", "", "Tempo account ID")
	return configureCmd
}

func AddEntryCmd() *cobra.Command {
	var capitalizableTime, ptoTime, otherTime int

	cmd := &cobra.Command{
		Use:     "add-week",
		Short:   "Add a timecard entry for a week of time",
		Example: "timecard add-week",
		RunE: func(cmd *cobra.Command, args []string) error {
			bearerToken := fetchBearerToken()
			accountId, issueId := fetchConfig()
			startOfWeek := requestDayOfWeek()

			// Use CLI flags if provided, otherwise prompt interactively
			hasAnyFlag := cmd.Flags().Changed("capitalizable-time") || cmd.Flags().Changed("pto-time") || cmd.Flags().Changed("other-time")

			if !hasAnyFlag {
				// No flags provided, use interactive prompts for all
				capitalizableTime, ptoTime, otherTime = requestTimeInput()
			} else {
				// Some or all flags provided - use flags for set values, prompt for missing ones
				if !cmd.Flags().Changed("capitalizable-time") {
					capitalizableTime = getTime(CapitalizableTime)
				}
				if !cmd.Flags().Changed("pto-time") {
					ptoTime = getTime(PtoTime)
				}
				if !cmd.Flags().Changed("other-time") {
					otherTime = getTime(OtherTime)
				}
			}

			if err := api.SendWorklog(api.CapitalizableWorkType, capitalizableTime, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}
			if err := api.SendWorklog(api.PtoWorkType, ptoTime, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}
			if err := api.SendWorklog(api.OtherWorkType, otherTime, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}

			fmt.Println("✅ All time entries submitted successfully!")
			return nil
		},
	}

	cmd.Flags().IntVarP(&capitalizableTime, "capitalizable-time", "c", 0, "Capitalizable time in hours")
	cmd.Flags().IntVarP(&ptoTime, "pto-time", "p", 0, "PTO time in hours")
	cmd.Flags().IntVarP(&otherTime, "other-time", "m", 0, "Other time in hours")

	return cmd
}
