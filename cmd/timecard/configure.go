package timecard

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/danlafeir/timecard/api"
	"github.com/danlafeir/cli-go/pkg/secrets"
	"github.com/spf13/viper"
)

const SECRETS_NAMESPACE = "timecard"
const API_TOKEN_NAME = "jira-api-token"
const ACCOUNT_ID_CONFIG = "tempo.accountId"
const ISSUE_ID_CONFIG = "tempo.issueId"

var configPath string

func configureApiToken(apiToken string) string {
	existing, _ := secrets.Read(SECRETS_NAMESPACE, API_TOKEN_NAME)
	if existing != "" && apiToken == "" {
		fmt.Print("A Tempo API token is already configured. Replace it? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if answer := strings.TrimSpace(scanner.Text()); answer != "y" && answer != "Y" {
			fmt.Println("Keeping existing token.")
			return existing
		}
	}

	token := strings.TrimSpace(apiToken)
	if token == "" {
		fmt.Print("Enter your Tempo API token: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		token = strings.TrimSpace(scanner.Text())
	}

	if token == "" {
		fmt.Println("Token cannot be empty. Re-run the configure command")
		os.Exit(1)
	}

	if err := secrets.Write(SECRETS_NAMESPACE, API_TOKEN_NAME, token); err != nil {
		fmt.Println("Failed to write token to keychain:", err)
		os.Exit(1)
	}
	fmt.Println("Tempo API token saved securely to keychain.")
	return token
}

func configureAccountId(accountId string) {
	existing := viper.GetString(ACCOUNT_ID_CONFIG)
	if existing != "" && accountId == "" {
		fmt.Printf("Account ID is already configured (%s). Replace it? (y/N): ", existing)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if answer := strings.TrimSpace(scanner.Text()); answer != "y" && answer != "Y" {
			fmt.Println("Keeping existing account ID.")
			return
		}
	}

	if accountId == "" {
		fmt.Print("Add Tempo Account Id here: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		accountId = strings.TrimSpace(scanner.Text())
		fmt.Print("\n")
	}
	if accountId == "" {
		fmt.Println("Account ID cannot be empty.")
		os.Exit(1)
	}
	viper.Set(ACCOUNT_ID_CONFIG, accountId)
}

func configureIssueId(accountId string) {
	if accountId == "" {
		fmt.Println("Account ID is required to fetch recent issue ID. Please configure account ID first.")
		os.Exit(1)
	}

	bearerToken := fetchBearerToken()
	fmt.Print("Fetching recent issue ID from Tempo API...\n")
	recentIssueId, err := api.GetRecentIssueId(accountId, bearerToken)
	if err != nil {
		fmt.Printf("Failed to fetch recent issue ID: %v\n", err)
		fmt.Print("Enter your default Issue ID manually: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		id := strings.TrimSpace(scanner.Text())
		if id == "" {
			fmt.Println("Issue ID cannot be empty.")
			os.Exit(1)
		}
		viper.Set(ISSUE_ID_CONFIG, id)
		return
	}

	id := strconv.Itoa(recentIssueId)
	fmt.Printf("Found recent issue ID: %s\n", id)
	viper.Set(ISSUE_ID_CONFIG, id)
}

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get user home directory:", err)
	}

	timecardConfigDir := filepath.Join(homeDir, ".timecard")
	if err := os.MkdirAll(timecardConfigDir, 0755); err != nil {
		log.Fatal("Failed to create .timecard config directory:", err)
	}
	return filepath.Join(timecardConfigDir, "config.yaml")
}

func initConfig() {
	configFilePath := getConfigPath()
	configDir := filepath.Dir(configFilePath)
	configName := strings.TrimSuffix(filepath.Base(configFilePath), filepath.Ext(configFilePath))

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatal("Failed to create config directory:", err)
	}

	// Create config file if it doesn't exist
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		emptyConfig := []byte("tempo:\n")
		if err := os.WriteFile(configFilePath, emptyConfig, 0644); err != nil {
			log.Fatal("Failed to create config file:", err)
		}
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
}

func fetchConfig() (accountId string, issueId string) {
	initConfig()
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	if !viper.IsSet(ACCOUNT_ID_CONFIG) {
		configureAccountId("")
	} else {
		accountId = viper.GetString(ACCOUNT_ID_CONFIG)
	}

	if !viper.IsSet(ISSUE_ID_CONFIG) {
		configureIssueId(accountId)
	} else {
		issueId = viper.GetString(ISSUE_ID_CONFIG)
	}

	return
}

func fetchBearerToken() string {
	bearerToken, err := secrets.Read(SECRETS_NAMESPACE, API_TOKEN_NAME)

	if bearerToken == "" || err != nil {
		return configureApiToken("")
	}
	return bearerToken
}
