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
			configuredAccountId := viper.GetString(ACCOUNT_ID_CONFIG)
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
	var capitalizableTime, ptoDays, otherTime int

	cmd := &cobra.Command{
		Use:     "add-week",
		Short:   "Add a timecard entry for a week of time",
		Example: "timecard add-week",
		RunE: func(cmd *cobra.Command, args []string) error {
			bearerToken := fetchBearerToken()
			accountId, issueId := fetchConfig()
			startOfWeek := requestDayOfWeek()

			hasAnyFlag := cmd.Flags().Changed("capitalizable-time") || cmd.Flags().Changed("pto-days") || cmd.Flags().Changed("other-time")

			if !hasAnyFlag {
				capitalizableTime, ptoDays, otherTime = requestTimeInput()
			} else {
				if !cmd.Flags().Changed("capitalizable-time") {
					capitalizableTime = getTime(CapitalizableTime)
				}
				if !cmd.Flags().Changed("pto-days") {
					ptoDays = getPtoDays()
				}
				if !cmd.Flags().Changed("other-time") {
					otherTime = getTime(OtherTime)
				}
			}

			ptoHours := ptoDays * hoursPerPtoDay
			total := capitalizableTime + ptoHours + otherTime
			if total < minWeeklyHours {
				return fmt.Errorf("total hours (%d) must be at least %d (capitalizable: %d, PTO: %d day(s) / %d hours, other: %d)",
					total, minWeeklyHours, capitalizableTime, ptoDays, ptoHours, otherTime)
			}

			if err := api.SendWorklog(api.CapitalizableWorkType, capitalizableTime, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}
			if err := api.SendPtoWorklog(ptoDays, startOfWeek, bearerToken, accountId, issueId); err != nil {
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
	cmd.Flags().IntVarP(&ptoDays, "pto-days", "p", 0, "PTO days (0-5; each day = 8 hours)")
	cmd.Flags().IntVarP(&otherTime, "other-time", "m", 0, "Other time in hours")

	return cmd
}
