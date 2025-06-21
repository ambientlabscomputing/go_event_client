package go_event_client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// TestConfig holds configuration for tests
type TestConfig struct {
	OAuthClientID     string
	OAuthClientSecret string
	OAuthTokenURL     string
	EventAPIURL       string
	SocketsURL        string
	TestTimeout       time.Duration
	LogLevel          slog.Level
}

// LoadTestConfig loads test configuration from environment variables
func LoadTestConfig() (*TestConfig, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	config := &TestConfig{
		OAuthClientID:     getEnvOrDefault("OAUTH_CLIENT_ID", "api_tester"),
		OAuthClientSecret: getEnvOrDefault("OAUTH_CLIENT_SECRET", "tester_scrt"),
		OAuthTokenURL:     getEnvOrDefault("OAUTH_TOKEN_URL", "http://api.ambientlabsdev.io/oauth/token"),
		EventAPIURL:       getEnvOrDefault("EVENT_API_URL", "http://events.ambientlabsdev.io"),
		SocketsURL:        getEnvOrDefault("SOCKETS_URL", "wss://sockets.ambientlabsdev.io"),
		TestTimeout:       30 * time.Second,
		LogLevel:          slog.LevelDebug,
	}

	// Parse timeout if provided
	if timeoutStr := os.Getenv("TEST_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.TestTimeout = timeout
		}
	}

	// Parse log level if provided
	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		switch strings.ToLower(logLevelStr) {
		case "debug":
			config.LogLevel = slog.LevelDebug
		case "info":
			config.LogLevel = slog.LevelInfo
		case "warn":
			config.LogLevel = slog.LevelWarn
		case "error":
			config.LogLevel = slog.LevelError
		}
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// OAuthToken represents an OAuth token response
type OAuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// GetTestToken fetches an OAuth token for testing
func GetTestToken(ctx context.Context, config *TestConfig) (string, error) {
	data := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     config.OAuthClientID,
		"client_secret": config.OAuthClientSecret,
		"request_type":  "api",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.OAuthTokenURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// CreateTestLogger creates a logger for testing
func CreateTestLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

// CreateTestClient creates a test client with proper configuration
func CreateTestClient(ctx context.Context, config *TestConfig) (EventClient, error) {
	logger := CreateTestLogger(config.LogLevel)

	options := EventClientOptions{
		EventAPIURL:  config.EventAPIURL,
		SocketsURL:   config.SocketsURL,
		PingInterval: 1,
	}

	getToken := func(ctx context.Context) (string, error) {
		return GetTestToken(ctx, config)
	}

	return NewEventClient(ctx, options, getToken, logger), nil
}
