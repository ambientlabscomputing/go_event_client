package go_event_client

import (
	"encoding/json"
	"fmt"
)

// KeyValuePair represents a key-value pair from the server
type KeyValuePair struct {
	Key   string      `json:"Key"`
	Value interface{} `json:"Value"`
}

// MessageContent represents flexible content that can be either a string or key-value pairs
type MessageContent struct {
	raw    json.RawMessage
	parsed interface{}
}

// UnmarshalJSON implements custom unmarshaling for MessageContent
func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	mc.raw = data

	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		mc.parsed = str
		return nil
	}

	// Try to unmarshal as key-value pairs array
	var kvPairs []KeyValuePair
	if err := json.Unmarshal(data, &kvPairs); err == nil {
		// Convert to map for easier access, converting all values to strings
		contentMap := make(map[string]string)
		for _, pair := range kvPairs {
			// Convert value to string regardless of original type
			valueStr := fmt.Sprintf("%v", pair.Value)
			contentMap[pair.Key] = valueStr
		}
		mc.parsed = contentMap
		return nil
	}

	// If neither works, store as raw JSON
	mc.parsed = string(data)
	return nil
}

// MarshalJSON implements custom marshaling for MessageContent
func (mc MessageContent) MarshalJSON() ([]byte, error) {
	if mc.raw != nil {
		return mc.raw, nil
	}
	return json.Marshal(mc.parsed)
}

// String returns the content as a string
func (mc *MessageContent) String() string {
	switch v := mc.parsed.(type) {
	case string:
		return v
	case map[string]string:
		// Convert map back to JSON string for backward compatibility
		if jsonBytes, err := json.Marshal(v); err == nil {
			return string(jsonBytes)
		}
		return fmt.Sprintf("%+v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// AsString returns the content as a string (same as String() but more explicit)
func (mc *MessageContent) AsString() string {
	return mc.String()
}

// AsMap returns the content as a map if it was parsed from key-value pairs
func (mc *MessageContent) AsMap() (map[string]string, bool) {
	if m, ok := mc.parsed.(map[string]string); ok {
		return m, true
	}
	return nil, false
}

// GetValue gets a value by key if content is a map, otherwise returns empty string
func (mc *MessageContent) GetValue(key string) string {
	if m, ok := mc.AsMap(); ok {
		return m[key]
	}
	return ""
}

// IsMap returns true if the content was parsed as key-value pairs
func (mc *MessageContent) IsMap() bool {
	_, ok := mc.parsed.(map[string]string)
	return ok
}

// IsString returns true if the content is a simple string
func (mc *MessageContent) IsString() bool {
	_, ok := mc.parsed.(string)
	return ok
}

// NewMessageContentFromString creates a MessageContent from a string
func NewMessageContentFromString(content string) MessageContent {
	// Marshal the string properly to ensure valid JSON
	jsonBytes, _ := json.Marshal(content)
	return MessageContent{
		raw:    jsonBytes,
		parsed: content,
	}
}

// NewMessageContentFromMap creates a MessageContent from a map
func NewMessageContentFromMap(content map[string]string) MessageContent {
	jsonBytes, _ := json.Marshal(content)
	return MessageContent{
		raw:    jsonBytes,
		parsed: content,
	}
}

type Message struct {
	ID            string         `json:"id"`
	CreatedAt     string         `json:"created_at"`
	Topic         string         `json:"topic"`
	Content       MessageContent `json:"content"`
	SubscriberID  string         `json:"subscriber_id"`
	ConnectionID  string         `json:"connection_id"`
	SessionID     string         `json:"session_id"`
	Timestamp     string         `json:"timestamp"`
	Priority      string         `json:"priority,omitempty"`
	AggregateType string         `json:"aggregate_type,omitempty"`
	AggregateID   *int           `json:"aggregate_id,omitempty"`
}

type MessageCreate struct {
	Topic         string `json:"topic"`
	Content       string `json:"content"`
	AggregateType string `json:"aggregate_type,omitempty"`
	AggregateID   *int   `json:"aggregate_id,omitempty"`
}

type Subscriber struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	UserID    string `json:"user_id"`
}

type SubscriptionCreate struct {
	Topic         string `json:"topic"`
	SubscriberID  string `json:"subscriber_id"`
	AggregateType string `json:"aggregate_type,omitempty"`
	AggregateID   *int   `json:"aggregate_id,omitempty"`
	IsRegex       bool   `json:"is_regex,omitempty"`
}

type SessionCreate struct {
	SubscriberID string `json:"subscriber_id"`
}

type Subscription struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	Topic         string `json:"topic"`
	SubscriberID  string `json:"subscriber_id"`
	AggregateType string `json:"aggregate_type,omitempty"`
	AggregateID   *int   `json:"aggregate_id,omitempty"`
	IsRegex       bool   `json:"is_regex,omitempty"`
}

type Session struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	SubscriberID  string `json:"subscriber_id"`
	Status        string `json:"status"`
	LastConnected string `json:"last_connected"`
	Token         string `json:"token"`
}
