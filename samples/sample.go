package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"log/slog"

	"github.com/ambientlabscomputing/go_event_client" // Replace with the actual module path
)

func main() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Define options for the EventClient
	options := go_event_client.EventClientOptions{
		EventAPIURL:  "http://events.ambientlabsdev.io",
		SocketsURL:   "wss://sockets.ambientlabsdev.io",
		PingInterval: 30,
	}

	// Define a token callback function
	getToken := func(ctx context.Context) (string, error) {
		// Replace this with your logic to fetch a token
		return "TOKEN", nil
	}

	// Create a new EventClient
	client := go_event_client.NewEventClient(context.Background(), options, getToken, logger)

	// Add a simple handler for a topic
	err := client.AddHandler("^example\\.topic$", func(message string) {
		fmt.Printf("Received message on 'example.topic': %s\n", message)
	})
	if err != nil {
		logger.Error("failed to subscribe to topic", "error", err)
		return
	}

	// Start the client
	if err := client.Start(); err != nil {
		logger.Error("failed to start client", "error", err)
		return
	}

	// subscribe to example.topic
	err = client.NewSubscription(context.Background(), "example.topic")
	if err != nil {
		logger.Error("failed to subscribe to topic", "error", err)
		return
	}
	defer client.Stop()

	// Publish a test message
	err = client.Publish("example.topic", map[string]string{"key": "value"})
	if err != nil {
		logger.Error("failed to publish message", "error", err)
		return
	}

	// Keep the client running for a while to receive messages
	fmt.Println("Client is running. Press Ctrl+C to exit.")
	time.Sleep(1 * time.Minute)
}
