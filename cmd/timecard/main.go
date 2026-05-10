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
	var capitalizableTime, ptoTime, otherTime int

	cmd := &cobra.Command{
		Use:     "add-week",
		Short:   "Add a timecard entry for a week of time",
		Example: "timecard add-week",
		RunE: func(cmd *cobra.Command, args []string) error {
			bearerToken := fetchBearerToken()
			accountId, issueId := fetchConfig()
			startOfWeek := requestDayOfWeek()

			hasAnyFlag := cmd.Flags().Changed("capitalizable-time") || cmd.Flags().Changed("pto-time") || cmd.Flags().Changed("other-time")

			if !hasAnyFlag {
				capitalizableTime, ptoTime, otherTime = requestTimeInput()
			} else {
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

			total := capitalizableTime + ptoTime + otherTime
			if total < minWeeklyHours {
				return fmt.Errorf("total hours (%d) must be at least %d (capitalizable: %d, PTO: %d, other: %d)",
					total, minWeeklyHours, capitalizableTime, ptoTime, otherTime)
			}

			// Compute PTO day allocation to determine remaining capacity per day.
			var fullWeek [api.MaxDaysPerWeek]int
			for i := range fullWeek {
				fullWeek[i] = api.HoursPerDay
			}
			ptoDayAlloc := api.AllocateHours(ptoTime, fullWeek)

			remainingCap := fullWeek
			for i, h := range ptoDayAlloc {
				remainingCap[i] -= h
			}

			capAlloc := api.AllocateHours(capitalizableTime, remainingCap)
			for i, h := range capAlloc {
				remainingCap[i] -= h
			}

			otherAlloc := api.AllocateHours(otherTime, remainingCap)

			if err := api.SendPtoWorklog(ptoTime, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}
			if err := api.SendWorklogAllocation(api.CapitalizableWorkType, capAlloc, startOfWeek, bearerToken, accountId, issueId); err != nil {
				return err
			}
			if err := api.SendWorklogAllocation(api.OtherWorkType, otherAlloc, startOfWeek, bearerToken, accountId, issueId); err != nil {
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
