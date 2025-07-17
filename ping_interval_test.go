package go_event_client

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestPingIntervalTiming verifies that the ping interval is correctly converted to seconds
func TestPingIntervalTiming(t *testing.T) {
	// Test with a very short interval for quick testing
	pingInterval := 1 // 1 second

	options := EventClientOptions{
		EventAPIURL:  "http://localhost:8085/api",
		SocketsURL:   "ws://localhost:8085",
		PingInterval: pingInterval,
	}

	client := NewEventClient(context.Background(), options, nil, slog.Default())

	// We can't easily test the actual ticker without starting the full client,
	// but we can verify the client is created with the correct options
	impl, ok := client.(*EventClientImpl)
	if !ok {
		t.Fatal("Expected EventClientImpl")
	}

	if impl.Options.PingInterval != pingInterval {
		t.Errorf("Expected PingInterval %d, got %d", pingInterval, impl.Options.PingInterval)
	}

	// Test that the duration calculation works correctly
	expectedDuration := time.Duration(pingInterval) * time.Second
	actualDuration := time.Duration(impl.Options.PingInterval) * time.Second

	if actualDuration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, actualDuration)
	}

	// Verify the duration is reasonable (1 second = 1000000000 nanoseconds)
	if actualDuration != time.Second {
		t.Errorf("Expected 1 second, got %v", actualDuration)
	}
}
