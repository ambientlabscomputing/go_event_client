package go_event_client

import (
	"regexp"
	"testing"
)

func TestHandlerSignature(t *testing.T) {
	// Test that we can create a handler entry with the new Message signature
	handler := func(m Message) {
		// Handler that expects a Message object
		if m.Topic == "" {
			t.Error("Expected topic to be set")
		}
	}

	// Create a regexp pattern
	pattern, err := regexp.Compile(`^test\..*$`)
	if err != nil {
		t.Fatalf("Failed to compile regex: %v", err)
	}

	// Create handler entry
	entry := handlerEntry{
		pattern: pattern,
		handler: handler,
	}

	// Create a test message
	testMessage := Message{
		ID:           "test-id",
		Topic:        "test.topic",
		Content:      NewMessageContentFromString(`{"test": "data"}`),
		ConnectionID: "conn-id",
		SessionID:    "sess-id",
		Timestamp:    "2023-01-01T00:00:00Z",
	}

	// Test that the handler can be called with a Message
	if entry.pattern.MatchString(testMessage.Topic) {
		entry.handler(testMessage)
	} else {
		t.Error("Pattern should match test.topic")
	}
}
