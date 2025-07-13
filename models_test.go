package go_event_client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessage_JSONMarshaling(t *testing.T) {
	// Test with all fields
	aggregateID := 123
	msg := Message{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt:     "2023-01-01T00:00:00Z",
		Topic:         "test.topic",
		Content:       NewMessageContentFromString(`{"key": "value"}`),
		SubscriberID:  "550e8400-e29b-41d4-a716-446655440001",
		ConnectionID:  "550e8400-e29b-41d4-a716-446655440002",
		SessionID:     "550e8400-e29b-41d4-a716-446655440003",
		Timestamp:     "2023-01-01T00:00:00Z",
		Priority:      "normal",
		AggregateType: "node",
		AggregateID:   &aggregateID,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Unmarshal back
	var unmarshaledMsg Message
	err = json.Unmarshal(jsonData, &unmarshaledMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Compare fields
	if unmarshaledMsg.ID != msg.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaledMsg.ID, msg.ID)
	}
	if unmarshaledMsg.Topic != msg.Topic {
		t.Errorf("Topic mismatch: got %s, want %s", unmarshaledMsg.Topic, msg.Topic)
	}
	if unmarshaledMsg.AggregateType != msg.AggregateType {
		t.Errorf("AggregateType mismatch: got %s, want %s", unmarshaledMsg.AggregateType, msg.AggregateType)
	}
	if unmarshaledMsg.AggregateID == nil || *unmarshaledMsg.AggregateID != *msg.AggregateID {
		t.Errorf("AggregateID mismatch: got %v, want %v", unmarshaledMsg.AggregateID, msg.AggregateID)
	}
}

func TestMessage_JSONMarshalingWithoutAggregateFields(t *testing.T) {
	// Test without aggregate fields
	msg := Message{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt:    "2023-01-01T00:00:00Z",
		Topic:        "test.topic",
		Content:      NewMessageContentFromString(`{"key": "value"}`),
		SubscriberID: "550e8400-e29b-41d4-a716-446655440001",
		ConnectionID: "550e8400-e29b-41d4-a716-446655440002",
		SessionID:    "550e8400-e29b-41d4-a716-446655440003",
		Timestamp:    "2023-01-01T00:00:00Z",
		Priority:     "normal",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Check that aggregate fields are omitted when empty
	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "aggregate_type") {
		t.Errorf("Expected aggregate_type to be omitted when empty, but found in JSON: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "aggregate_id") {
		t.Errorf("Expected aggregate_id to be omitted when nil, but found in JSON: %s", jsonStr)
	}
}

func TestSubscription_JSONMarshaling(t *testing.T) {
	aggregateID := 456
	sub := Subscription{
		ID:            "550e8400-e29b-41d4-a716-446655440003",
		CreatedAt:     "2023-01-01T00:00:00Z",
		Topic:         "test.*",
		SubscriberID:  "550e8400-e29b-41d4-a716-446655440004",
		AggregateType: "user",
		AggregateID:   &aggregateID,
		IsRegex:       true,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(sub)
	if err != nil {
		t.Fatalf("Failed to marshal subscription: %v", err)
	}

	// Unmarshal back
	var unmarshaledSub Subscription
	err = json.Unmarshal(jsonData, &unmarshaledSub)
	if err != nil {
		t.Fatalf("Failed to unmarshal subscription: %v", err)
	}

	// Compare fields
	if unmarshaledSub.ID != sub.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaledSub.ID, sub.ID)
	}
	if unmarshaledSub.IsRegex != sub.IsRegex {
		t.Errorf("IsRegex mismatch: got %t, want %t", unmarshaledSub.IsRegex, sub.IsRegex)
	}
	if unmarshaledSub.AggregateType != sub.AggregateType {
		t.Errorf("AggregateType mismatch: got %s, want %s", unmarshaledSub.AggregateType, sub.AggregateType)
	}
	if unmarshaledSub.AggregateID == nil || *unmarshaledSub.AggregateID != *sub.AggregateID {
		t.Errorf("AggregateID mismatch: got %v, want %v", unmarshaledSub.AggregateID, sub.AggregateID)
	}
}

func TestSubscriptionCreate_JSONMarshaling(t *testing.T) {
	aggregateID := 789
	subCreate := SubscriptionCreate{
		Topic:         "node.created",
		SubscriberID:  "550e8400-e29b-41d4-a716-446655440005",
		AggregateType: "node",
		AggregateID:   &aggregateID,
		IsRegex:       false,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(subCreate)
	if err != nil {
		t.Fatalf("Failed to marshal subscription create: %v", err)
	}

	// Unmarshal back
	var unmarshaledSubCreate SubscriptionCreate
	err = json.Unmarshal(jsonData, &unmarshaledSubCreate)
	if err != nil {
		t.Fatalf("Failed to unmarshal subscription create: %v", err)
	}

	// Compare fields
	if unmarshaledSubCreate.Topic != subCreate.Topic {
		t.Errorf("Topic mismatch: got %s, want %s", unmarshaledSubCreate.Topic, subCreate.Topic)
	}
	if unmarshaledSubCreate.SubscriberID != subCreate.SubscriberID {
		t.Errorf("SubscriberID mismatch: got %s, want %s", unmarshaledSubCreate.SubscriberID, subCreate.SubscriberID)
	}
	if unmarshaledSubCreate.AggregateType != subCreate.AggregateType {
		t.Errorf("AggregateType mismatch: got %s, want %s", unmarshaledSubCreate.AggregateType, subCreate.AggregateType)
	}
	if unmarshaledSubCreate.AggregateID == nil || *unmarshaledSubCreate.AggregateID != *subCreate.AggregateID {
		t.Errorf("AggregateID mismatch: got %v, want %v", unmarshaledSubCreate.AggregateID, subCreate.AggregateID)
	}
	if unmarshaledSubCreate.IsRegex != subCreate.IsRegex {
		t.Errorf("IsRegex mismatch: got %t, want %t", unmarshaledSubCreate.IsRegex, subCreate.IsRegex)
	}
}

func TestSubscriber_UUIDFields(t *testing.T) {
	subscriber := Subscriber{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt: "2023-01-01T00:00:00Z",
		UserID:    "user123",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(subscriber)
	if err != nil {
		t.Fatalf("Failed to marshal subscriber: %v", err)
	}

	// Unmarshal back
	var unmarshaledSubscriber Subscriber
	err = json.Unmarshal(jsonData, &unmarshaledSubscriber)
	if err != nil {
		t.Fatalf("Failed to unmarshal subscriber: %v", err)
	}

	// Check that ID is still a string (UUID)
	if unmarshaledSubscriber.ID != subscriber.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaledSubscriber.ID, subscriber.ID)
	}
}

func TestSession_UUIDFields(t *testing.T) {
	session := Session{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		CreatedAt:     "2023-01-01T00:00:00Z",
		SubscriberID:  "550e8400-e29b-41d4-a716-446655440001",
		Status:        "active",
		LastConnected: "2023-01-01T00:00:00Z",
		Token:         "abc123token",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	// Unmarshal back
	var unmarshaledSession Session
	err = json.Unmarshal(jsonData, &unmarshaledSession)
	if err != nil {
		t.Fatalf("Failed to unmarshal session: %v", err)
	}

	// Check that IDs are still strings (UUIDs)
	if unmarshaledSession.ID != session.ID {
		t.Errorf("ID mismatch: got %s, want %s", unmarshaledSession.ID, session.ID)
	}
	if unmarshaledSession.SubscriberID != session.SubscriberID {
		t.Errorf("SubscriberID mismatch: got %s, want %s", unmarshaledSession.SubscriberID, session.SubscriberID)
	}
	if unmarshaledSession.Status != session.Status {
		t.Errorf("Status mismatch: got %s, want %s", unmarshaledSession.Status, session.Status)
	}
}

func TestMessageContent_FlexibleParsing(t *testing.T) {
	// Test parsing string content
	stringContentJSON := `{"id":"test-1","topic":"test.topic","content":"Hello World","subscriber_id":"test-sub","connection_id":"test-conn","session_id":"test-sess","timestamp":"2023-01-01T00:00:00Z","priority":"normal","created_at":"2023-01-01T00:00:00Z"}`

	var stringMsg Message
	err := json.Unmarshal([]byte(stringContentJSON), &stringMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal string content: %v", err)
	}

	if !stringMsg.Content.IsString() {
		t.Error("Expected content to be recognized as string")
	}

	if stringMsg.Content.String() != "Hello World" {
		t.Errorf("Expected content string to be 'Hello World', got '%s'", stringMsg.Content.String())
	}

	// Test parsing key-value pairs content
	kvContentJSON := `{"id":"test-2","topic":"test.topic","content":[{"Key":"message","Value":"Hello from WebSocket tester!"},{"Key":"timestamp","Value":"2025-07-13T15:40:06.471Z"}],"subscriber_id":"test-sub","connection_id":"test-conn","session_id":"test-sess","timestamp":"2023-01-01T00:00:00Z","priority":"normal","created_at":"2023-01-01T00:00:00Z"}`

	var kvMsg Message
	err = json.Unmarshal([]byte(kvContentJSON), &kvMsg)
	if err != nil {
		t.Fatalf("Failed to unmarshal key-value content: %v", err)
	}

	if !kvMsg.Content.IsMap() {
		t.Error("Expected content to be recognized as map")
	}

	contentMap, ok := kvMsg.Content.AsMap()
	if !ok {
		t.Fatal("Expected content to be accessible as map")
	}

	expectedMessage := "Hello from WebSocket tester!"
	if contentMap["message"] != expectedMessage {
		t.Errorf("Expected message value to be '%s', got '%s'", expectedMessage, contentMap["message"])
	}

	expectedTimestamp := "2025-07-13T15:40:06.471Z"
	if contentMap["timestamp"] != expectedTimestamp {
		t.Errorf("Expected timestamp value to be '%s', got '%s'", expectedTimestamp, contentMap["timestamp"])
	}

	// Test GetValue method
	if kvMsg.Content.GetValue("message") != expectedMessage {
		t.Errorf("GetValue('message') returned '%s', expected '%s'", kvMsg.Content.GetValue("message"), expectedMessage)
	}

	// Test that non-existent keys return empty string
	if kvMsg.Content.GetValue("nonexistent") != "" {
		t.Errorf("GetValue('nonexistent') should return empty string, got '%s'", kvMsg.Content.GetValue("nonexistent"))
	}

	// Test marshaling back
	stringMsgBytes, err := json.Marshal(stringMsg)
	if err != nil {
		t.Fatalf("Failed to marshal string message: %v", err)
	}

	kvMsgBytes, err := json.Marshal(kvMsg)
	if err != nil {
		t.Fatalf("Failed to marshal key-value message: %v", err)
	}

	// Should be able to unmarshal back successfully
	var remarshaled1, remarshaled2 Message
	if err := json.Unmarshal(stringMsgBytes, &remarshaled1); err != nil {
		t.Fatalf("Failed to remarshal string message: %v", err)
	}

	if err := json.Unmarshal(kvMsgBytes, &remarshaled2); err != nil {
		t.Fatalf("Failed to remarshal key-value message: %v", err)
	}

	if remarshaled1.Content.String() != stringMsg.Content.String() {
		t.Error("Remarshaled string content doesn't match original")
	}

	if !remarshaled2.Content.IsMap() {
		t.Error("Remarshaled key-value content should still be a map")
	}
}
