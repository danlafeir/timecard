package timecard

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

const (
	CapitalizableTime = "How much time did you spend developing, designing or testing software? This is considered capitalizable time (in hours): "
	PtoTime           = "How much time did you spend with PTO (vacation or sick) (in hours): "
	OtherTime         = "How much time did you spend on other activities i.e. meetings, etc. (in hours): "

	minWeeklyHours = 40
)

func requestTimeInput() (capitalizableTime, ptoTime, otherTime int) {
	fmt.Printf("Answer the following questions to estimate how you spent your time this week.\n\n")

	for {
		capitalizableTime = getTime(CapitalizableTime)
		ptoTime = getTime(PtoTime)
		otherTime = getTime(OtherTime)

		total := capitalizableTime + ptoTime + otherTime
		fmt.Printf("Total hours this week: %d (capitalizable: %d, PTO: %d, other: %d)\n",
			total, capitalizableTime, ptoTime, otherTime)

		if total >= minWeeklyHours {
			break
		}
		fmt.Printf("\nTotal is %dh — %dh short of the required %dh. Please re-enter.\n\n",
			total, minWeeklyHours-total, minWeeklyHours)
	}
	return
}

func getTime(printString string) int {
	fmt.Print(printString)
	var timeInput string
	if _, err := fmt.Scan(&timeInput); err != nil {
		log.Fatal(err)
	}
	return stringToInt(timeInput)
}

func stringToInt(input string) int {
	var convertedValue, err = strconv.Atoi(input)
	if err != nil {
		log.Fatal(err)
	}
	return convertedValue
}

func requestDayOfWeek() time.Time {
	mondayOfThisWeek := determineWeekforTimeSheet()

	fmt.Printf("Would you like to fill out time for %s (Y/N)? ", mondayOfThisWeek.Format(time.DateOnly))
	var confirmTime string
	if _, err := fmt.Scan(&confirmTime); err != nil {
		log.Fatal(err)
	}

	if confirmTime == "y" || confirmTime == "Y" {
		return mondayOfThisWeek
	}

	fmt.Printf("\nHow many weeks back would you like to fill out (ex. 1 means last week): ")
	var timeInput string
	if _, err := fmt.Scan(&timeInput); err != nil {
		log.Fatal(err)
	}

	weeksBack := stringToInt(timeInput)
	fmt.Printf("Now we are filling out a timesheet for %s\n", mondayOfThisWeek.AddDate(0, 0, -7*weeksBack).Format(time.DateOnly))

	return mondayOfThisWeek.AddDate(0, 0, -7*weeksBack)
}

func determineWeekforTimeSheet() time.Time {
	currentDay := time.Now()
	dayOfTheWeek := int(currentDay.Weekday())
	var distanceToMonday int
	if dayOfTheWeek-1 == -1 {
		distanceToMonday = -6
	} else {
		distanceToMonday = -(dayOfTheWeek - 1)
	}

	monday := currentDay.AddDate(0, 0, distanceToMonday)
	print(fmt.Sprintf("This will fill out the timesheet for the week of %s\n\n", monday.Format(time.DateOnly)))
	return monday
}
