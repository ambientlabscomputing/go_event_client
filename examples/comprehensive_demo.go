package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/ambientlabscomputing/go_event_client"
)

func main() {
	// Set up logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load environment variables
	apiURL := os.Getenv("EVENT_API_URL")
	socketsURL := os.Getenv("SOCKETS_URL")
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	oauthURL := os.Getenv("OAUTH_TOKEN_URL")

	if apiURL == "" || socketsURL == "" || clientID == "" || clientSecret == "" || oauthURL == "" {
		log.Fatal("Missing required environment variables. Please set AMBIENT_EVENT_API_URL, AMBIENT_SOCKETS_URL, AMBIENT_CLIENT_ID, AMBIENT_CLIENT_SECRET, and AMBIENT_OAUTH_URL")
	}

	// Create OAuth token callback
	getTokenCallback := func(ctx context.Context) (string, error) {
		return go_event_client.GetOAuthToken(ctx, oauthURL, clientID, clientSecret)
	}

	// Create client options
	options := go_event_client.EventClientOptions{
		EventAPIURL:  apiURL,
		SocketsURL:   socketsURL,
		PingInterval: 30,
	}

	// Create the event client
	ctx := context.Background()
	client := go_event_client.NewEventClient(ctx, options, getTokenCallback, logger)

	// Start the client
	if err := client.Start(); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	defer client.Stop()

	logger.Info("Event client started successfully")

	// Example 1: Basic subscription and message handling
	fmt.Println("\n=== Example 1: Basic Subscription ===")

	err := client.AddHandler("^demo\\.basic\\..*", func(message go_event_client.Message) {
		fmt.Printf("📨 Basic message received: %s\n", message.Content)
	})
	if err != nil {
		log.Fatalf("Failed to add basic handler: %v", err)
	}

	err = client.NewSubscription(ctx, "demo.basic.test")
	if err != nil {
		log.Fatalf("Failed to create basic subscription: %v", err)
	}

	// Wait for subscription to be established
	time.Sleep(2 * time.Second)

	// Publish a basic message
	basicMessage := map[string]interface{}{
		"type":      "notification",
		"message":   "Hello from basic demo!",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	err = client.Publish("demo.basic.test", basicMessage)
	if err != nil {
		log.Fatalf("Failed to publish basic message: %v", err)
	}

	// Example 2: Aggregate type subscription
	fmt.Println("\n=== Example 2: Aggregate Type Subscription ===")

	err = client.AddHandler("^demo\\.user\\..*", func(message go_event_client.Message) {
		fmt.Printf("👤 User aggregate message received: %s\n", message.Content)
	})
	if err != nil {
		log.Fatalf("Failed to add user handler: %v", err)
	}

	// Subscribe to all user events
	err = client.NewAggregateTypeSubscription(ctx, "demo.user.updated", "user", false)
	if err != nil {
		log.Fatalf("Failed to create user aggregate subscription: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Publish a message with aggregate information
	userMessage := map[string]interface{}{
		"action":    "profile_updated",
		"user_id":   123,
		"username":  "john_doe",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	err = client.PublishWithAggregate("demo.user.updated", userMessage, "user", nil)
	if err != nil {
		log.Fatalf("Failed to publish user message: %v", err)
	}

	// Example 3: Specific aggregate subscription
	fmt.Println("\n=== Example 3: Specific Aggregate Subscription ===")

	err = client.AddHandler("^demo\\.order\\..*", func(message go_event_client.Message) {
		fmt.Printf("🛒 Order aggregate message received: %s\n", message.Content)
	})
	if err != nil {
		log.Fatalf("Failed to add order handler: %v", err)
	}

	// Subscribe to a specific order
	specificOrderID := 456
	err = client.NewAggregateSubscription(ctx, "demo.order.status", "order", specificOrderID, false)
	if err != nil {
		log.Fatalf("Failed to create specific order subscription: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Publish a message for the specific order
	orderMessage := map[string]interface{}{
		"action":    "status_changed",
		"order_id":  456,
		"status":    "shipped",
		"tracking":  "1Z999AA1234567890",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	err = client.PublishWithAggregate("demo.order.status", orderMessage, "order", &specificOrderID)
	if err != nil {
		log.Fatalf("Failed to publish order message: %v", err)
	}

	// Example 4: Regex subscription
	fmt.Println("\n=== Example 4: Regex Subscription ===")

	err = client.AddHandler("^demo\\.system\\..*", func(message go_event_client.Message) {
		fmt.Printf("🔧 System message received: %s\n", message.Content)
	})
	if err != nil {
		log.Fatalf("Failed to add system handler: %v", err)
	}

	// Create a regex subscription that matches multiple topics
	err = client.NewSubscriptionWithOptions(ctx, "demo\\.system\\..*", "", nil, true)
	if err != nil {
		log.Fatalf("Failed to create regex subscription: %v", err)
	}

	time.Sleep(2 * time.Second)

	// Publish multiple messages that should match the regex
	systemTopics := []string{
		"demo.system.health",
		"demo.system.metrics",
		"demo.system.alerts",
	}

	for i, topic := range systemTopics {
		systemMessage := map[string]interface{}{
			"type":      "system_event",
			"topic":     topic,
			"sequence":  i + 1,
			"data":      fmt.Sprintf("System event #%d", i+1),
			"timestamp": time.Now().Format(time.RFC3339),
		}

		err = client.Publish(topic, systemMessage)
		if err != nil {
			log.Printf("Failed to publish system message to %s: %v", topic, err)
		} else {
			fmt.Printf("📤 Published message to %s\n", topic)
		}

		// Small delay between messages
		time.Sleep(500 * time.Millisecond)
	}

	// Example 5: HTTP API Publishing (if connection_id is available)
	fmt.Println("\n=== Example 5: HTTP API Publishing ===")

	// Wait a bit to ensure we have a connection_id from received messages
	time.Sleep(3 * time.Second)

	apiMessage := map[string]interface{}{
		"type":      "api_published",
		"message":   "This message was published via HTTP API",
		"method":    "HTTP",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	err = client.PublishViaAPI(ctx, "demo.basic.test", apiMessage, "", nil)
	if err != nil {
		fmt.Printf("⚠️  Failed to publish via API (this is expected if no messages were received yet): %v\n", err)
	} else {
		fmt.Println("📤 Successfully published message via HTTP API")
	}

	// Wait to see all messages
	fmt.Println("\n=== Waiting for messages (5 seconds) ===")
	time.Sleep(5 * time.Second)

	fmt.Println("\n=== Demo completed successfully! ===")
	fmt.Println("All event bus features have been demonstrated:")
	fmt.Println("✅ Basic subscriptions and publishing")
	fmt.Println("✅ Aggregate type subscriptions")
	fmt.Println("✅ Specific aggregate subscriptions")
	fmt.Println("✅ Regex pattern subscriptions")
	fmt.Println("✅ HTTP API publishing")
	fmt.Println("✅ Real-time message handling")
}
