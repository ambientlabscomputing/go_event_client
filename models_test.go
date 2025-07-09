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
		Content:       `{"key": "value"}`,
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
		Content:      `{"key": "value"}`,
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
}
