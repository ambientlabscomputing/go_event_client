package go_event_client

type Message struct {
	ID           int    `json:"id"`
	CreatedAt    string `json:"created_at"`
	Topic        string `json:"topic"`
	Message      string `json:"message"`
	ConnectionID int    `json:"connection_id"`
	SessionID    int    `json:"session_id"`
	Timestamp    string `json:"timestamp"`
}

type MessageCreate struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

type Subscriber struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
	UserID    string `json:"user_id"`
}

type SubscriptionCreate struct {
	Topic        string `json:"topic"`
	SubscriberID int    `json:"subscriber_id"`
}

type Subscription struct {
	ID           int    `json:"id"`
	CreatedAt    string `json:"created_at"`
	Topic        string `json:"topic"`
	SubscriberID int    `json:"subscriber_id"`
}

type Session struct {
	ID            int    `json:"id"`
	CreatedAt     string `json:"created_at"`
	SubscriberID  int    `json:"subscriber_id"`
	Status        string `json:"status"`
	LastConnected string `json:"last_connected"`
	Token         string `json:"token"`
}
