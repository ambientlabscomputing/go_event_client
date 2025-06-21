package go_event_client

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestIntegration_BasicSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	client, err := CreateTestClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Test that we can get a token
	token, err := GetTestToken(ctx, config)
	if err != nil {
		t.Fatalf("Failed to get test token: %v", err)
	}
	if token == "" {
		t.Fatal("Expected non-empty token")
	}
	t.Logf("Successfully obtained token: %s...", token[:10])

	// Start the client
	err = client.Start()
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer client.Stop()

	// Set up a channel to receive messages
	received := make(chan string, 1)

	// Add a handler for test messages
	err = client.AddHandler("^test\\.integration\\..*", func(message string) {
		t.Logf("Received message: %s", message)
		received <- message
	})
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create a basic subscription
	err = client.NewSubscription(ctx, "test.integration.basic")
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}

	// Give some time for the subscription to be established
	time.Sleep(2 * time.Second)

	// Publish a test message
	testMessage := map[string]string{
		"test_id": "integration_test_1",
		"message": "Hello from integration test",
	}
	err = client.Publish("test.integration.basic", testMessage)
	if err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Wait for the message to be received
	select {
	case msg := <-received:
		t.Logf("Successfully received message: %s", msg)

		// Parse the message to ensure it's what we sent
		var receivedData map[string]string
		err = json.Unmarshal([]byte(msg), &receivedData)
		if err != nil {
			t.Fatalf("Failed to parse received message: %v", err)
		}

		if receivedData["test_id"] != testMessage["test_id"] {
			t.Errorf("Expected test_id %s, got %s", testMessage["test_id"], receivedData["test_id"])
		}

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestIntegration_AggregateTypeSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	client, err := CreateTestClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	err = client.Start()
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer client.Stop()

	// Set up a channel to receive messages
	received := make(chan string, 1)

	// Add a handler for aggregate messages
	err = client.AddHandler("^test\\.aggregate\\..*", func(message string) {
		t.Logf("Received aggregate message: %s", message)
		received <- message
	})
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create an aggregate type subscription
	err = client.NewAggregateTypeSubscription(ctx, "test.aggregate.node", "node", false)
	if err != nil {
		t.Fatalf("Failed to create aggregate type subscription: %v", err)
	}

	// Give some time for the subscription to be established
	time.Sleep(2 * time.Second)

	// Publish a test message with aggregate information via WebSocket
	testMessage := map[string]interface{}{
		"test_id": "aggregate_test_1",
		"action":  "created",
		"node_id": 123,
	}
	err = client.PublishWithAggregate("test.aggregate.node", testMessage, "node", nil)
	if err != nil {
		t.Fatalf("Failed to publish aggregate message: %v", err)
	}

	// Wait for the message to be received
	select {
	case msg := <-received:
		t.Logf("Successfully received aggregate message: %s", msg)

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for aggregate message")
	}
}

func TestIntegration_SpecificAggregateSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	client, err := CreateTestClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	err = client.Start()
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer client.Stop()

	// Set up a channel to receive messages
	received := make(chan string, 1)

	// Add a handler for specific aggregate messages
	err = client.AddHandler("^test\\.specific\\..*", func(message string) {
		t.Logf("Received specific aggregate message: %s", message)
		received <- message
	})
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create a specific aggregate subscription
	err = client.NewAggregateSubscription(ctx, "test.specific.user", "user", 456, false)
	if err != nil {
		t.Fatalf("Failed to create specific aggregate subscription: %v", err)
	}

	// Give some time for the subscription to be established
	time.Sleep(2 * time.Second)

	// Publish a test message with specific aggregate ID
	testMessage := map[string]interface{}{
		"test_id": "specific_aggregate_test_1",
		"action":  "updated",
		"user_id": 456,
	}
	aggregateID := 456
	err = client.PublishWithAggregate("test.specific.user", testMessage, "user", &aggregateID)
	if err != nil {
		t.Fatalf("Failed to publish specific aggregate message: %v", err)
	}

	// Wait for the message to be received
	select {
	case msg := <-received:
		t.Logf("Successfully received specific aggregate message: %s", msg)

	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for specific aggregate message")
	}
}

func TestIntegration_RegexSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestTimeout)
	defer cancel()

	client, err := CreateTestClient(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	err = client.Start()
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}
	defer client.Stop()

	// Set up a channel to receive messages
	received := make(chan string, 2)

	// Add a handler for regex pattern messages
	err = client.AddHandler("^test\\.regex\\..*", func(message string) {
		t.Logf("Received regex message: %s", message)
		received <- message
	})
	if err != nil {
		t.Fatalf("Failed to add handler: %v", err)
	}

	// Create a regex subscription - should match test.regex.something and test.regex.anything
	err = client.NewSubscriptionWithOptions(ctx, "test\\.regex\\..*", "", nil, true)
	if err != nil {
		t.Fatalf("Failed to create regex subscription: %v", err)
	}

	// Give some time for the subscription to be established
	time.Sleep(2 * time.Second)

	// Publish multiple test messages that should match the regex
	testMessages := []string{
		"test.regex.first",
		"test.regex.second",
	}

	for _, topic := range testMessages {
		testMessage := map[string]interface{}{
			"test_id": "regex_test",
			"topic":   topic,
		}
		err = client.Publish(topic, testMessage)
		if err != nil {
			t.Fatalf("Failed to publish regex message to %s: %v", topic, err)
		}
	}

	// Wait for messages to be received
	receivedCount := 0
	for receivedCount < len(testMessages) {
		select {
		case msg := <-received:
			t.Logf("Successfully received regex message %d: %s", receivedCount+1, msg)
			receivedCount++

		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for regex messages. Received %d out of %d", receivedCount, len(testMessages))
		}
	}
}

func TestOAuthTokenFetching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config, err := LoadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := GetTestToken(ctx, config)
	if err != nil {
		t.Fatalf("Failed to get OAuth token: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	t.Logf("Successfully obtained OAuth token: %s...", token[:min(len(token), 20)])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
