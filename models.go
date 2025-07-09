package go_event_client

type Message struct {
	ID            string `json:"id"`
	CreatedAt     string `json:"created_at"`
	Topic         string `json:"topic"`
	Content       string `json:"content"`
	SubscriberID  string `json:"subscriber_id"`
	ConnectionID  string `json:"connection_id"`
	SessionID     string `json:"session_id"`
	Timestamp     string `json:"timestamp"`
	Priority      string `json:"priority,omitempty"`
	AggregateType string `json:"aggregate_type,omitempty"`
	AggregateID   *int   `json:"aggregate_id,omitempty"`
}

type MessageCreate struct {
	Topic         string `json:"topic"`
	Message       string `json:"message"`
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
