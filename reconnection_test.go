package go_event_client

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestReconnectionBehavior demonstrates the reconnection functionality
func TestReconnectionBehavior(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	
	// Create a client with custom reconnection settings
	options := EventClientOptions{
		EventAPIURL:          "http://localhost:8085/api",
		SocketsURL:           "ws://localhost:8085",
		PingInterval:         1,
		MaxReconnectAttempts: 3,              // Limited attempts for testing
		ReconnectBackoff:     500 * time.Millisecond, // Faster backoff for testing
		MaxReconnectBackoff:  2 * time.Second,
	}
	
	ctx := context.Background()
	
	// This will fail to connect since there's likely no server on port 9999
	invalidOptions := options
	invalidOptions.SocketsURL = "ws://localhost:9999"
	
	client := NewEventClient(ctx, invalidOptions, func(ctx context.Context) (string, error) {
		return "fake-token", nil
	}, logger)
	
	// This should demonstrate reconnection attempts and eventual failure
	impl := client.(*EventClientImpl)
	
	// Test the reconnection logic directly
	t.Run("ReconnectionAttempts", func(t *testing.T) {
		// Set up a session (this would normally be done in Start())
		impl.Session = &Session{ID: "test-session-id"}
		impl.ctx, impl.cancel = context.WithCancel(ctx)
		
		// Try to establish connection - this should fail and trigger reconnection attempts
		success := impl.attemptReconnection()
		
		if success {
			t.Error("Expected reconnection to fail with invalid server")
		}
		
		// Check that reconnection attempts were made
		if impl.reconnectCount == 0 {
			t.Error("Expected at least one reconnection attempt")
		}
		
		t.Logf("Made %d reconnection attempts as expected", impl.reconnectCount)
	})
}

// TestReconnectionConfiguration tests different reconnection configurations
func TestReconnectionConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	
	t.Run("DefaultConfiguration", func(t *testing.T) {
		options := EventClientOptions{
			EventAPIURL:  "http://localhost:8085/api",
			SocketsURL:   "ws://localhost:8085",
			PingInterval: 1,
			// No reconnection settings - should use defaults
		}
		
		client := NewEventClient(context.Background(), options, func(ctx context.Context) (string, error) {
			return "fake-token", nil
		}, logger)
		
		impl := client.(*EventClientImpl)
		
		// Check that defaults were set
		if impl.Options.MaxReconnectAttempts != 0 {
			t.Errorf("Expected infinite retries (0), got %d", impl.Options.MaxReconnectAttempts)
		}
		if impl.Options.ReconnectBackoff != 1*time.Second {
			t.Errorf("Expected 1s backoff, got %v", impl.Options.ReconnectBackoff)
		}
		if impl.Options.MaxReconnectBackoff != 30*time.Second {
			t.Errorf("Expected 30s max backoff, got %v", impl.Options.MaxReconnectBackoff)
		}
	})
	
	t.Run("CustomConfiguration", func(t *testing.T) {
		options := EventClientOptions{
			EventAPIURL:          "http://localhost:8085/api",
			SocketsURL:           "ws://localhost:8085",
			PingInterval:         1,
			MaxReconnectAttempts: 5,
			ReconnectBackoff:     2 * time.Second,
			MaxReconnectBackoff:  60 * time.Second,
		}
		
		client := NewEventClient(context.Background(), options, func(ctx context.Context) (string, error) {
			return "fake-token", nil
		}, logger)
		
		impl := client.(*EventClientImpl)
		
		// Check that custom values were preserved
		if impl.Options.MaxReconnectAttempts != 5 {
			t.Errorf("Expected 5 max attempts, got %d", impl.Options.MaxReconnectAttempts)
		}
		if impl.Options.ReconnectBackoff != 2*time.Second {
			t.Errorf("Expected 2s backoff, got %v", impl.Options.ReconnectBackoff)
		}
		if impl.Options.MaxReconnectBackoff != 60*time.Second {
			t.Errorf("Expected 60s max backoff, got %v", impl.Options.MaxReconnectBackoff)
		}
	})
}
