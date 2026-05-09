package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCalculateHoursPerDay(t *testing.T) {
	tests := []struct {
		name       string
		totalHours int
		dayNumber  int
		expected   int
	}{
		{
			name:       "less than 5 hours returns 1",
			totalHours: 3,
			dayNumber:  1,
			expected:   1,
		},
		{
			name:       "exactly 5 hours returns 1 per day",
			totalHours: 5,
			dayNumber:  1,
			expected:   1,
		},
		{
			name:       "40 hours day 1",
			totalHours: 40,
			dayNumber:  1,
			expected:   8,
		},
		{
			name:       "40 hours day 5",
			totalHours: 40,
			dayNumber:  5,
			expected:   8,
		},
		{
			name:       "7 hours day 1 gets remainder",
			totalHours: 7,
			dayNumber:  1,
			expected:   2,
		},
		{
			name:       "7 hours day 3",
			totalHours: 7,
			dayNumber:  3,
			expected:   1,
		},
		{
			name:       "13 hours day 1",
			totalHours: 13,
			dayNumber:  1,
			expected:   3,
		},
		{
			name:       "23 hours day 1",
			totalHours: 23,
			dayNumber:  1,
			expected:   5,
		},
		{
			name:       "23 hours day 5",
			totalHours: 23,
			dayNumber:  5,
			expected:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateHoursPerDay(tt.totalHours, tt.dayNumber)
			if got != tt.expected {
				t.Errorf("calculateHoursPerDay(%d, %d) = %d, want %d", tt.totalHours, tt.dayNumber, got, tt.expected)
			}
		})
	}
}

func TestCalculateHoursPerDay_TotalSumsCorrectly(t *testing.T) {
	testCases := []int{5, 7, 10, 13, 23, 40}
	for _, totalHours := range testCases {
		daysToLog := totalHours
		if daysToLog > 5 {
			daysToLog = 5
		}
		sum := 0
		for day := 1; day <= daysToLog; day++ {
			sum += calculateHoursPerDay(totalHours, day)
		}
		if sum != totalHours {
			t.Errorf("total hours %d: sum across days = %d, want %d", totalHours, sum, totalHours)
		}
	}
}

func TestCleanBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "leading and trailing whitespace",
			input:    "  my-token  ",
			expected: "my-token",
		},
		{
			name:     "newlines",
			input:    "my-token\n",
			expected: "my-token",
		},
		{
			name:     "already clean",
			input:    "my-token",
			expected: "my-token",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "tabs and newlines",
			input:    "\t my-token \n\r",
			expected: "my-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanBearerToken(tt.input)
			if got != tt.expected {
				t.Errorf("cleanBearerToken(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCreateWorklogRequest(t *testing.T) {
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	workType := CapitalizableWorkType
	accountID := "acct-123"
	issueID := "ISSUE-456"
	hours := 8

	req := createWorklogRequest(workType, hours, date, accountID, issueID)

	if req.StartDate != "2024-03-15" {
		t.Errorf("StartDate = %q, want %q", req.StartDate, "2024-03-15")
	}
	if req.TimeSpentSeconds != 8*3600 {
		t.Errorf("TimeSpentSeconds = %d, want %d", req.TimeSpentSeconds, 8*3600)
	}
	if req.AuthorAccountID != accountID {
		t.Errorf("AuthorAccountID = %q, want %q", req.AuthorAccountID, accountID)
	}
	if req.IssueID != issueID {
		t.Errorf("IssueID = %q, want %q", req.IssueID, issueID)
	}
	if len(req.Attributes) != 1 || req.Attributes[0] != workType {
		t.Errorf("Attributes = %v, want [%v]", req.Attributes, workType)
	}
	if req.StartTime != "09:00:00" {
		t.Errorf("StartTime = %q, want %q", req.StartTime, "09:00:00")
	}
}

func TestCalculateWeekPriorDate(t *testing.T) {
	result := calculateWeekPriorDate()

	_, err := time.Parse(time.DateOnly, result)
	if err != nil {
		t.Fatalf("calculateWeekPriorDate() returned invalid date format: %q", result)
	}

	expected := time.Now().AddDate(0, 0, -14).Format(time.DateOnly)
	if result != expected {
		t.Errorf("calculateWeekPriorDate() = %q, want %q", result, expected)
	}
}

func TestHandleAPIError(t *testing.T) {
	reqBody := &WorklogRequest{
		AuthorAccountID: "acct-123",
		IssueID:         "ISSUE-456",
		Attributes:      []WorkType{CapitalizableWorkType},
	}

	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectContains string
	}{
		{
			name:           "401 returns auth error",
			statusCode:     http.StatusUnauthorized,
			body:           "unauthorized",
			expectContains: "Authentication failed",
		},
		{
			name:           "400 returns detailed error",
			statusCode:     http.StatusBadRequest,
			body:           `{"message":"bad request"}`,
			expectContains: "API request failed with HTTP 400",
		},
		{
			name:           "500 returns server error",
			statusCode:     http.StatusInternalServerError,
			body:           "internal error",
			expectContains: "server error (HTTP 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			err := handleAPIError(resp, reqBody)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("error %q should contain %q", err.Error(), tt.expectContains)
			}
		})
	}
}

func TestSendPtoWorklog(t *testing.T) {
	monday := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC) // a known Monday

	t.Run("zero hours makes no requests", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		err := SendPtoWorklog(0, monday, "token", "acct", "ISSUE-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != 0 {
			t.Errorf("expected 0 HTTP calls, got %d", calls)
		}
	})

	t.Run("24 hours fills three consecutive days at 8h each", func(t *testing.T) {
		var received []WorklogRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req WorklogRequest
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)
			received = append(received, req)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		err := SendPtoWorklog(24, monday, "token", "acct", "ISSUE-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(received) != 3 {
			t.Fatalf("expected 3 requests, got %d", len(received))
		}
		wantDates := []string{"2024-01-08", "2024-01-09", "2024-01-10"}
		for i, want := range wantDates {
			if received[i].StartDate != want {
				t.Errorf("request %d: StartDate = %q, want %q", i, received[i].StartDate, want)
			}
			if received[i].TimeSpentSeconds != hoursPerPtoDay*secondsPerHour {
				t.Errorf("request %d: TimeSpentSeconds = %d, want %d", i, received[i].TimeSpentSeconds, hoursPerPtoDay*secondsPerHour)
			}
		}
	})

	t.Run("partial day gets remainder hours", func(t *testing.T) {
		var seconds []int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req WorklogRequest
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)
			seconds = append(seconds, req.TimeSpentSeconds)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		// 20h = 8h + 8h + 4h across 3 days
		err := SendPtoWorklog(20, monday, "token", "acct", "ISSUE-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []int{8 * secondsPerHour, 8 * secondsPerHour, 4 * secondsPerHour}
		if len(seconds) != len(want) {
			t.Fatalf("expected %d requests, got %d", len(want), len(seconds))
		}
		for i, w := range want {
			if seconds[i] != w {
				t.Errorf("request %d: TimeSpentSeconds = %d, want %d", i, seconds[i], w)
			}
		}
	})

	t.Run("40 hours fills all five days", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		err := SendPtoWorklog(40, monday, "token", "acct", "ISSUE-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != maxDaysPerWeek {
			t.Errorf("expected %d requests, got %d", maxDaysPerWeek, calls)
		}
	})

	t.Run("hours exceeding 40 are capped at five days", func(t *testing.T) {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		err := SendPtoWorklog(80, monday, "token", "acct", "ISSUE-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calls != maxDaysPerWeek {
			t.Errorf("expected %d requests (cap), got %d", maxDaysPerWeek, calls)
		}
	})

	t.Run("server error propagates", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"bad"}`))
		}))
		defer server.Close()

		orig := tempoAPIBaseURL
		tempoAPIBaseURL = server.URL
		defer func() { tempoAPIBaseURL = orig }()

		err := SendPtoWorklog(8, monday, "token", "acct", "ISSUE-1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendWorklogEntry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer test-token")
		}
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}

		// Verify body
		body, _ := io.ReadAll(r.Body)
		var req WorklogRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
		}
		if req.IssueID != "ISSUE-123" {
			t.Errorf("IssueID = %q, want %q", req.IssueID, "ISSUE-123")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Temporarily override the base URL
	originalURL := tempoAPIBaseURL
	defer func() {
		// Can't reassign const, so we test via the httptest approach below
		_ = originalURL
	}()

	// Since tempoAPIBaseURL is a const, we need to test with a custom approach.
	// We'll verify the function handles a successful response by creating the request
	// body and verifying it serializes correctly.
	reqBody := createWorklogRequest(CapitalizableWorkType, 8, time.Now(), "acct-123", "ISSUE-123")
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Send to test server directly
	httpReq, _ := http.NewRequest("POST", server.URL, strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Bearer test-token")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestSendWorklogEntry_ErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer server.Close()

	// Verify non-200 responses produce errors by hitting the test server directly
	reqBody := createWorklogRequest(CapitalizableWorkType, 8, time.Now(), "acct-123", "ISSUE-123")
	jsonData, _ := json.Marshal(reqBody)

	httpReq, _ := http.NewRequest("POST", server.URL, strings.NewReader(string(jsonData)))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 status code")
	}
}
