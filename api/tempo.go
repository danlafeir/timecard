package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultStartTime = "09:00:00"
	secondsPerHour   = 3600
	MaxDaysPerWeek   = 5 // work days in a week
	HoursPerDay      = 8 // standard hours in a workday
	daysInWeek       = 7
)

// tempoAPIBaseURL and tempoAPIUserBaseURL are vars so tests can override them
// with an httptest.Server URL.
var (
	tempoAPIBaseURL     = "https://api.tempo.io/4/worklogs"
	tempoAPIUserBaseURL = "https://api.tempo.io/4/worklogs/user"
)

type WorkType struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WorklogRequest struct {
	AuthorAccountID  string     `json:"authorAccountId"`
	Description      string     `json:"description"`
	IssueID          string     `json:"issueId"`
	StartDate        string     `json:"startDate"`
	StartTime        string     `json:"startTime"`
	TimeSpentSeconds int        `json:"timeSpentSeconds"`
	Attributes       []WorkType `json:"attributes"`
}

// Issue represents the issue information in a worklog response.
type Issue struct {
	ID int `json:"id"`
}

// WorklogResponse represents a worklog entry returned from the Tempo API.
type WorklogResponse struct {
	Issue Issue `json:"issue"`
}

// UserWorklogsResponse represents the response from the user worklogs endpoint.
type UserWorklogsResponse struct {
	Results  []WorklogResponse `json:"results"`
	Metadata struct {
		Count  int `json:"count"`
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"metadata"`
}

var (
	CapitalizableWorkType = WorkType{
		Key:   "_WorkType_",
		Value: "14C",
	}
	PtoWorkType = WorkType{
		Key:   "_WorkType_",
		Value: "20E",
	}
	OtherWorkType = WorkType{
		Key:   "_WorkType_",
		Value: "12E",
	}
)

// createWorklogRequest builds a worklog request for a specific day.
func createWorklogRequest(workType WorkType, hours int, date time.Time, accountID, issueID string) *WorklogRequest {
	return &WorklogRequest{
		AuthorAccountID:  accountID,
		Description:      "timecard",
		IssueID:          issueID,
		StartDate:        date.Format(time.DateOnly),
		StartTime:        defaultStartTime,
		TimeSpentSeconds: hours * secondsPerHour,
		Attributes:       []WorkType{workType},
	}
}

// cleanBearerToken removes whitespace and newlines from the bearer token.
func cleanBearerToken(token string) string {
	return strings.TrimSpace(token)
}

// sendWorklogEntry sends a single worklog entry to the Tempo API.
func sendWorklogEntry(reqBody *WorklogRequest, bearerToken string) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", tempoAPIBaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	cleanedToken := cleanBearerToken(bearerToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanedToken))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(resp, reqBody)
	}

	return nil
}

// handleAPIError processes and returns detailed error information for API failures.
func handleAPIError(resp *http.Response, reqBody *WorklogRequest) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d error (unable to read response body): %w", resp.StatusCode, err)
	}

	// Handle 401 Unauthorized - suggest reconfiguring API token
	if resp.StatusCode == http.StatusUnauthorized {
		log.Printf("❌ Authentication Failed (HTTP 401)\n")
		log.Printf("🔑 Your Tempo API token appears to be invalid or expired.\n")
		return fmt.Errorf("Authentication failed: Please configure a new Tempo API token")
	}

	// Log detailed error information for debugging
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		log.Printf("❌ API Request Failed (HTTP %d)\n", resp.StatusCode)
		log.Printf("👤 Account ID: %s\n", reqBody.AuthorAccountID)
		log.Printf("🎫 Issue ID: %s\n", reqBody.IssueID)
		if len(reqBody.Attributes) > 0 {
			log.Printf("🏷️ Work Type: %s\n", reqBody.Attributes[0].Value)
		}
		log.Printf("📝 Response: %s\n", string(bodyBytes))

		reqJSON, _ := json.Marshal(reqBody)
		log.Printf("🔗 Request: %s\n", string(reqJSON))

		return fmt.Errorf("API request failed with HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Handle 500-level and other errors
	log.Printf("❌ Server Error (HTTP %d): %s\n", resp.StatusCode, string(bodyBytes))
	return fmt.Errorf("server error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
}

// SendPtoWorklog allocates PTO hours serially across consecutive weekdays
// starting from startDay, filling up to HoursPerDay (8) hours per day.
// Unlike SendWorklog, hours are not spread fractionally — each day is filled
// to 8h before moving to the next.
func SendPtoWorklog(hours int, startDay time.Time, bearerToken, accountID, issueID string) error {
	if hours <= 0 {
		return nil
	}
	remaining := hours
	for day := 0; remaining > 0 && day < MaxDaysPerWeek; day++ {
		hoursThisDay := remaining
		if hoursThisDay > HoursPerDay {
			hoursThisDay = HoursPerDay
		}
		logDate := startDay.AddDate(0, 0, day)
		fmt.Printf("Logging %d hours PTO for %s\n", hoursThisDay, logDate.Format(time.DateOnly))
		reqBody := createWorklogRequest(PtoWorkType, hoursThisDay, logDate, accountID, issueID)
		if err := sendWorklogEntry(reqBody, bearerToken); err != nil {
			return fmt.Errorf("failed to send PTO for %s: %w", logDate.Format(time.DateOnly), err)
		}
		remaining -= hoursThisDay
	}
	return nil
}

// AllocateHours returns how many hours to log on each of MaxDaysPerWeek days,
// given a budget and per-day capacity limits. Days are filled serially up to
// their capacity. Any hours that exceed the total capacity are spread across
// days that had non-zero capacity (days not fully covered by PTO).
func AllocateHours(hours int, dayCapacities [MaxDaysPerWeek]int) [MaxDaysPerWeek]int {
	var alloc [MaxDaysPerWeek]int
	if hours <= 0 {
		return alloc
	}
	remaining := hours

	// Serial fill: top each day up to its capacity before moving on.
	for i := 0; i < MaxDaysPerWeek && remaining > 0; i++ {
		if dayCapacities[i] <= 0 {
			continue
		}
		give := remaining
		if give > dayCapacities[i] {
			give = dayCapacities[i]
		}
		alloc[i] = give
		remaining -= give
	}

	// Excess beyond capacity: spread across days that had any capacity.
	if remaining > 0 {
		n := 0
		for _, c := range dayCapacities {
			if c > 0 {
				n++
			}
		}
		for i := 0; i < MaxDaysPerWeek && remaining > 0 && n > 0; i++ {
			if dayCapacities[i] <= 0 {
				continue
			}
			extra := (remaining + n - 1) / n
			alloc[i] += extra
			remaining -= extra
			n--
		}
	}

	return alloc
}

// SendWorklogAllocation sends pre-computed per-day hour allocations to the
// Tempo API. Days with zero hours are skipped.
func SendWorklogAllocation(workType WorkType, allocation [MaxDaysPerWeek]int, startDay time.Time, bearerToken, accountID, issueID string) error {
	for i, h := range allocation {
		if h <= 0 {
			continue
		}
		logDate := startDay.AddDate(0, 0, i)
		fmt.Printf("Logging %d hours for %s\n", h, logDate.Format(time.DateOnly))
		reqBody := createWorklogRequest(workType, h, logDate, accountID, issueID)
		if err := sendWorklogEntry(reqBody, bearerToken); err != nil {
			return fmt.Errorf("failed to send worklog for %s: %w", logDate.Format(time.DateOnly), err)
		}
	}
	return nil
}

// calculateWeekPriorDate calculates the date that is two weeks prior to the current date.
// Returns the date formatted as YYYY-MM-DD.
func calculateWeekPriorDate() string {
	twoWeeksAgo := time.Now().AddDate(0, 0, -daysInWeek*2)
	return twoWeeksAgo.Format(time.DateOnly)
}

// GetRecentIssueId fetches worklogs for a specific user account from the Tempo API.
// It queries worklogs updated from two weeks prior to the current date.
// Returns the issue ID from the last worklog entry in the results, or an error if the request fails or no results are found.
func GetRecentIssueId(accountID, bearerToken string) (int, error) {
	updatedFrom := calculateWeekPriorDate()

	// Build URL with account ID and query parameter
	baseURL := fmt.Sprintf("%s/%s", tempoAPIUserBaseURL, accountID)
	apiURL, err := url.Parse(baseURL)
	if err != nil {
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameter
	query := apiURL.Query()
	query.Set("updatedFrom", updatedFrom)
	apiURL.RawQuery = query.Encode()

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	cleanedToken := cleanBearerToken(bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cleanedToken))

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-OK responses
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("HTTP %d error (unable to read response body): %w", resp.StatusCode, err)
		}

		// Handle 401 Unauthorized - suggest reconfiguring API token
		if resp.StatusCode == http.StatusUnauthorized {
			log.Printf("❌ Authentication Failed (HTTP 401)\n")
			log.Printf("🔑 Your Tempo API token appears to be invalid or expired.\n")
			log.Printf("💡 Please configure a new Tempo API token by running:\n")
			log.Printf("   timecard configure --token <YOUR_NEW_TOKEN>\n")
			return 0, fmt.Errorf("authentication failed: please configure a new Tempo API token")
		}

		log.Printf("❌ GetRecentIssueId API Request Failed (HTTP %d)\n", resp.StatusCode)
		log.Printf("👤 Account ID: %s\n", accountID)
		log.Printf("📅 Updated From: %s\n", updatedFrom)
		log.Printf("📝 Response: %s\n", string(bodyBytes))

		return 0, fmt.Errorf("API request failed with HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode response
	var worklogsResponse UserWorklogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&worklogsResponse); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract issue ID from the last result
	if len(worklogsResponse.Results) == 0 {
		return 0, fmt.Errorf("no worklogs found in response")
	}

	lastResult := worklogsResponse.Results[len(worklogsResponse.Results)-1]
	return lastResult.Issue.ID, nil
}
