package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/ambientlabscomputing/go_event_client"
)

func main() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Define options for the EventClient
	options := go_event_client.EventClientOptions{
		EventAPIURL:  "http://events.ambientlabsdev.io",
		SocketsURL:   "wss://sockets.ambientlabsdev.io",
		PingInterval: 1,
	}

	// Define a token callback function - in a real application,
	// you'd implement proper OAuth token fetching here
	getToken := func(ctx context.Context) (string, error) {
		// This example shows how you could use the test helper for development
		// In production, replace this with your own OAuth implementation
		config, err := go_event_client.LoadTestConfig()
		if err != nil {
			return "", err
		}
		return go_event_client.GetTestToken(ctx, config)
	}

	// Create a new EventClient
	client := go_event_client.NewEventClient(context.Background(), options, getToken, logger)

	// Add handlers for different types of events

	// Handler for exact topic matching
	err := client.AddHandler("^example-1\\.topic$", func(message string) {
		fmt.Printf("Received message on exact match 'example-1.topic': %s\n", message)
	})
	if err != nil {
		logger.Error("failed to add exact match handler", "error", err)
		return
	}

	// Handler for regex pattern matching
	err = client.AddHandler("^user\\..*$", func(message string) {
		fmt.Printf("Received user event: %s\n", message)
	})
	if err != nil {
		logger.Error("failed to add regex handler", "error", err)
		return
	}

	// Handler for node events
	err = client.AddHandler("^node\\..*$", func(message string) {
		fmt.Printf("Received node event: %s\n", message)
	})
	if err != nil {
		logger.Error("failed to add node handler", "error", err)
		return
	}

	// Start the client
	if err := client.Start(); err != nil {
		logger.Error("failed to start client", "error", err)
		return
	}
	defer client.Stop()

	// Example 1: Basic subscription with exact matching
	err = client.NewSubscription(context.Background(), "example-1.topic")
	if err != nil {
		logger.Error("failed to create basic subscription", "error", err)
		return
	}

	// Example 2: Subscription with regex pattern matching
	err = client.NewSubscriptionWithOptions(context.Background(), "user\\..*", "", nil, true)
	if err != nil {
		logger.Error("failed to create regex subscription", "error", err)
		return
	}

	// Example 3: Aggregate type subscription (all events for 'node' type)
	err = client.NewAggregateTypeSubscription(context.Background(), "node.events", "node", false)
	if err != nil {
		logger.Error("failed to create aggregate type subscription", "error", err)
		return
	}

	// Example 4: Specific aggregate subscription (events for node ID 123)
	err = client.NewAggregateSubscription(context.Background(), "node.specific", "node", 123, false)
	if err != nil {
		logger.Error("failed to create specific aggregate subscription", "error", err)
		return
	}

	// Give subscriptions time to be established
	fmt.Println("Subscriptions created. Waiting for them to be established...")
	time.Sleep(3 * time.Second)

	// Publish test messages to demonstrate different subscription types

	// Test basic subscription
	err = client.Publish("example-1.topic", map[string]string{
		"message": "Hello from basic subscription",
		"type":    "basic",
	})
	if err != nil {
		logger.Error("failed to publish basic message", "error", err)
	}

	// Test regex subscription
	err = client.Publish("user.created", map[string]interface{}{
		"message": "User created event",
		"user_id": 456,
		"type":    "user_event",
	})
	if err != nil {
		logger.Error("failed to publish user message", "error", err)
	}

	// Test aggregate type subscription
	err = client.Publish("node.events", map[string]interface{}{
		"message":    "Node event for all nodes",
		"event_type": "status_update",
		"type":       "aggregate_type",
	})
	if err != nil {
		logger.Error("failed to publish aggregate type message", "error", err)
	}

	// Test specific aggregate subscription
	err = client.Publish("node.specific", map[string]interface{}{
		"message": "Specific node event",
		"node_id": 123,
		"action":  "updated",
		"type":    "specific_aggregate",
	})
	if err != nil {
		logger.Error("failed to publish specific aggregate message", "error", err)
	}

	// Keep the client running for a while to receive messages
	fmt.Println("Client is running. Press Ctrl+C to exit.")
	fmt.Println("Waiting for messages...")
	time.Sleep(10 * time.Second)

	fmt.Println("Demo completed!")
}
