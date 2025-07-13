package go_event_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GetTokenCallback func(ctx context.Context) (string, error)

type EventClient interface {
	Start() error
	Stop() error
	AddHandler(expr string, handler func(Message)) error
	Publish(topic string, v interface{}) error
	PublishWithAggregate(topic string, v interface{}, aggregateType string, aggregateID *int) error
	PublishViaAPI(ctx context.Context, topic string, v interface{}, aggregateType string, aggregateID *int) error
	NewSubscription(ctx context.Context, topic string) error
	NewSubscriptionWithOptions(ctx context.Context, topic string, aggregateType string, aggregateID *int, isRegex bool) error
	NewAggregateTypeSubscription(ctx context.Context, topic string, aggregateType string, isRegex bool) error
	NewAggregateSubscription(ctx context.Context, topic string, aggregateType string, aggregateID int, isRegex bool) error
}

type EventClientOptions struct {
	EventAPIURL  string
	SocketsURL   string
	PingInterval int
}

type handlerEntry struct {
	pattern *regexp.Regexp
	handler func(Message)
}

type EventClientImpl struct {
	Ctx              context.Context
	Options          EventClientOptions
	GetTokenCallback GetTokenCallback
	Logger           *slog.Logger
	HTTPClient       *http.Client

	// internal values set during runtime
	Subscriber   *Subscriber
	Session      *Session
	ConnectionID string
	conn         *websocket.Conn
	send         chan []byte
	handlers     []handlerEntry
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewEventClient(ctx context.Context, options EventClientOptions, getTokenCallback GetTokenCallback, logger *slog.Logger) EventClient {
	return &EventClientImpl{
		Ctx:              ctx,
		Options:          options,
		GetTokenCallback: getTokenCallback,
		Logger:           logger,
		HTTPClient:       &http.Client{},
	}
}

func (e *EventClientImpl) Start() error {
	logger := e.Logger

	logger.Debug("starting event client", "event_api_url", e.Options.EventAPIURL, "sockets_url", e.Options.SocketsURL)
	if err := e.RegisterSubscriber(); err != nil {
		logger.Error("failed to register subscriber", "error", err)
		return err
	}
	logger.Info("subscriber registered", "subscriber_id", e.Subscriber.ID)

	if err := e.RequestSession(); err != nil {
		logger.Error("failed to request session", "error", err)
		return err
	}
	logger.Info("session requested", "session_id", e.Session.ID)

	e.ctx, e.cancel = context.WithCancel(e.Ctx)
	e.send = make(chan []byte, 256)

	var err error
	connURI := e.Options.SocketsURL + "/ws/" + e.Session.ID
	logger.Info("connecting to websocket", "url", connURI)
	e.conn, _, err = websocket.DefaultDialer.Dial(connURI, nil)
	if err != nil {
		logger.Error("failed to connect to websocket", "error", err)
		return err
	}

	go e.writePump()
	go e.readPump()
	logger.Info("connected to websocket", "url", e.Options.SocketsURL)

	return nil
}

func (e *EventClientImpl) RegisterSubscriber() error {
	token, err := e.GetTokenCallback(e.Ctx)
	if err != nil {
		e.Logger.Error("failed to get token", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(e.Ctx, http.MethodPost, e.Options.EventAPIURL+"/v2/subscribers", nil)
	if err != nil {
		e.Logger.Error("failed to create request", "error", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		errData, _ := io.ReadAll(resp.Body)
		e.Logger.Error("failed to send request", "error", err, "data", errData)
		return err
	}

	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		e.Logger.Error("failed to register subscriber", "status_code", resp.StatusCode, "data", string(errData))
		return fmt.Errorf("failed to register subscriber: status %d, body: %s", resp.StatusCode, string(errData))
	}

	var subscriber Subscriber
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&subscriber); err != nil {
		e.Logger.Error("failed to decode response", "error", err)
		return err
	}
	e.Subscriber = &subscriber
	e.Logger.Info("subscriber registered", "subscriber_id", subscriber.ID)
	return nil
}

func (e *EventClientImpl) RequestSession() error {
	logger := e.Logger
	if e.Subscriber == nil {
		logger.Error("subscriber not registered")
		return fmt.Errorf("subscriber not registered")
	}
	token, err := e.GetTokenCallback(e.Ctx)
	if err != nil {
		logger.Error("failed to get token", "error", err)
		return err
	}

	sessionCreate := SessionCreate{
		SubscriberID: e.Subscriber.ID,
	}

	jsonData, err := json.Marshal(sessionCreate)
	if err != nil {
		logger.Error("failed to marshal session create", "error", err)
		return err
	}

	req, err := http.NewRequestWithContext(e.Ctx, http.MethodPost, e.Options.EventAPIURL+"/v2/sessions", bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("failed to create request", "error", err)
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		errData, _ := io.ReadAll(resp.Body)
		logger.Error("failed to send request", "error", err, "data", errData)
		return err
	}

	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		logger.Error("failed to request session", "status_code", resp.StatusCode, "data", string(errData))
		return fmt.Errorf("failed to request session: status %d, body: %s", resp.StatusCode, string(errData))
	}

	var session Session
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		logger.Error("failed to decode response", "error", err)
		return err
	}
	e.Session = &session
	logger.Info("session requested", "session_id", session.ID)
	return nil
}

// readPump blocks on ReadMessage, parsing incoming JSON frames and dispatching
func (e *EventClientImpl) readPump() {
	defer e.cancel()
	logger := e.Logger

	// Set limits and pong handler
	logger.Info("READPUMP: Starting read pump")
	logger.Debug("READPUMP: Setting read limits and pong handler")
	e.conn.SetReadLimit(65536) // 64KB limit
	e.conn.SetReadDeadline(time.Now().Add(time.Duration(e.Options.PingInterval) * time.Second))
	e.conn.SetPongHandler(func(appData string) error {
		logger.Debug("READPUMP: Received pong", "app_data", appData)
		e.conn.SetReadDeadline(time.Now().Add(time.Duration(e.Options.PingInterval) * time.Second))
		return nil
	})
	logger.Info("READPUMP: Read pump configured and started")

	messageCount := 0
	for {
		select {
		case <-e.ctx.Done():
			logger.Info("READPUMP: Context cancelled, stopping read pump", "messages_processed", messageCount)
			return
		default:
			logger.Debug("READPUMP: Waiting for WebSocket message...")
			_, data, err := e.conn.ReadMessage()
			if err != nil {
				// Check if this is a graceful shutdown
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					logger.Info("READPUMP: WebSocket closed gracefully", "error", err, "messages_processed", messageCount)
					return
				} else {
					logger.Error("READPUMP: Failed to read WebSocket message", "error", err, "messages_processed", messageCount)
					return
				}
			}

			messageCount++
			logger.Info("READPUMP: Received WebSocket message",
				"message_number", messageCount,
				"data_size", len(data),
				"data_preview", func() string {
					if len(data) > 200 {
						return string(data[:200]) + "..."
					}
					return string(data)
				}())

			// Dispatch matching handlers in their own goroutines
			logger.Debug("READPUMP: Dispatching message to handlers")
			go e.dispatch(data)
			logger.Debug("READPUMP: Message dispatched (async)")
		}
	}
}

// writePump blocks on the send channel and ticker for pings
func (e *EventClientImpl) writePump() {
	ticker := time.NewTicker(time.Duration(e.Options.PingInterval))
	defer func() {
		ticker.Stop()
		e.conn.Close()
		e.cancel()
	}()

	for {
		select {
		case msg := <-e.send:
			e.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := e.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			e.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := e.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-e.ctx.Done():
			return
		}
	}
}

// dispatch unmarshals the frame and calls all handlers whose regex matches the topic
func (e *EventClientImpl) dispatch(raw []byte) {
	logger := e.Logger

	logger.Info("DISPATCH: Starting message dispatch", "message_size", len(raw))
	logger.Debug("DISPATCH: Raw message data", "raw_data", string(raw))

	if len(raw) == 0 {
		logger.Warn("DISPATCH: Received empty frame, ignoring")
		return
	}

	var m Message
	if err := json.Unmarshal(raw, &m); err != nil {
		logger.Error("DISPATCH: Failed to unmarshal message", "error", err, "raw_data", string(raw))
		return
	}

	logger.Info("DISPATCH: Message unmarshaled successfully",
		"topic", m.Topic,
		"message_id", m.ID,
		"subscriber_id", m.SubscriberID,
		"session_id", m.SessionID,
		"connection_id", m.ConnectionID,
		"content_preview", func() string {
			content := m.Content.String()
			if len(content) > 100 {
				return content[:100] + "..."
			}
			return content
		}())

	// Capture connection ID from the first message for API publishing
	if e.ConnectionID == "" && m.ConnectionID != "" {
		e.ConnectionID = m.ConnectionID
		logger.Info("DISPATCH: Captured connection ID for API publishing", "connection_id", e.ConnectionID)
	}

	logger.Debug("DISPATCH: Acquiring read lock for handlers")
	e.mu.RLock()
	handlerCount := len(e.handlers)
	logger.Info("DISPATCH: Handler lookup", "num_handlers", handlerCount, "topic", m.Topic)

	if handlerCount == 0 {
		logger.Warn("DISPATCH: No handlers registered, ignoring message", "topic", m.Topic)
		e.mu.RUnlock()
		return
	}

	// Copy handlers to avoid holding the lock while calling them
	entries := append([]handlerEntry(nil), e.handlers...)
	e.mu.RUnlock()
	logger.Debug("DISPATCH: Released read lock, copied handlers", "handler_count", len(entries))

	found_handler := false
	matching_handlers := 0

	for i, entry := range entries {
		pattern := entry.pattern.String()
		logger.Debug("DISPATCH: Testing handler",
			"handler_index", i,
			"pattern", pattern,
			"topic", m.Topic)

		matches := entry.pattern.MatchString(m.Topic)
		logger.Debug("DISPATCH: Pattern match result",
			"handler_index", i,
			"pattern", pattern,
			"topic", m.Topic,
			"matches", matches)

		if matches {
			logger.Info("DISPATCH: Handler matched!",
				"handler_index", i,
				"pattern", pattern,
				"topic", m.Topic)
			found_handler = true
			matching_handlers++

			// Wrap handler execution with logging and panic recovery
			go func(handlerIndex int, handlerPattern string, message Message, handlerFunc func(Message)) {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("HANDLER: Handler panicked",
							"handler_index", handlerIndex,
							"pattern", handlerPattern,
							"topic", message.Topic,
							"panic", r)
					}
				}()

				logger.Info("HANDLER: Executing handler",
					"handler_index", handlerIndex,
					"pattern", handlerPattern,
					"topic", message.Topic,
					"message_id", message.ID)

				handlerFunc(message)

				logger.Info("HANDLER: Handler completed",
					"handler_index", handlerIndex,
					"pattern", handlerPattern,
					"topic", message.Topic,
					"message_id", message.ID)
			}(i, pattern, m, entry.handler)
		} else {
			logger.Debug("DISPATCH: Handler did not match",
				"handler_index", i,
				"pattern", pattern,
				"topic", m.Topic)
		}
	}

	if !found_handler {
		logger.Warn("DISPATCH: No handlers matched topic",
			"topic", m.Topic,
			"total_handlers", len(entries),
			"message_id", m.ID)
		// Log all handler patterns for debugging
		for i, entry := range entries {
			logger.Debug("DISPATCH: Available handler pattern",
				"handler_index", i,
				"pattern", entry.pattern.String())
		}
	} else {
		logger.Info("DISPATCH: Message dispatched successfully",
			"topic", m.Topic,
			"message_id", m.ID,
			"matching_handlers", matching_handlers,
			"total_handlers", len(entries))
	}
}

// AddHandler registers a callback for topics matching the given regex
// The expr should be a valid Go regex (e.g. "^user\\..*$" to match "user.*").
func (e *EventClientImpl) AddHandler(expr string, handler func(Message)) error {
	logger := e.Logger
	logger.Info("ADDHANDLER: Registering new handler", "pattern", expr)

	re, err := regexp.Compile(expr)
	if err != nil {
		logger.Error("ADDHANDLER: Failed to compile regex pattern", "pattern", expr, "error", err)
		return err
	}

	e.mu.Lock()
	handlerIndex := len(e.handlers)
	e.handlers = append(e.handlers, handlerEntry{pattern: re, handler: handler})
	totalHandlers := len(e.handlers)
	e.mu.Unlock()

	logger.Info("ADDHANDLER: Handler registered successfully",
		"pattern", expr,
		"handler_index", handlerIndex,
		"total_handlers", totalHandlers)

	return nil
}

// Publish sends a topic and payload. It blocks only if the send buffer is full.
func (e *EventClientImpl) Publish(topic string, v interface{}) error {
	return e.PublishWithAggregate(topic, v, "", nil)
}

// PublishWithAggregate sends a topic and payload with aggregate information
func (e *EventClientImpl) PublishWithAggregate(topic string, v interface{}, aggregateType string, aggregateID *int) error {
	msg := Message{
		Topic:         topic,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	msg.Content = NewMessageContentFromString(string(data))

	frame, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case e.send <- frame:
		return nil
	case <-e.ctx.Done():
		return e.ctx.Err()
	}
}

// PublishViaAPI publishes a message via HTTP API instead of WebSocket (useful for testing)
func (e *EventClientImpl) PublishViaAPI(ctx context.Context, topic string, v interface{}, aggregateType string, aggregateID *int) error {
	if e.Session == nil {
		return fmt.Errorf("session not available for API publishing")
	}
	if e.ConnectionID == "" {
		return fmt.Errorf("connection ID not available for API publishing - ensure WebSocket connection is established and at least one message has been received")
	}

	token, err := e.GetTokenCallback(ctx)
	if err != nil {
		return err
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	msgCreate := MessageCreate{
		Topic:         topic,
		Content:       string(data),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
	}

	jsonData, err := json.Marshal(msgCreate)
	if err != nil {
		return err
	}

	// Add required query parameters
	url := fmt.Sprintf("%s/v2/messages?session_id=%s&connection_id=%s", e.Options.EventAPIURL, e.Session.ID, e.ConnectionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to publish message: status %d, body: %s", resp.StatusCode, string(errData))
	}

	return nil
}

func (e *EventClientImpl) Stop() error {
	e.cancel()
	if e.conn != nil {
		e.conn.Close()
	}
	return nil
}

func (e *EventClientImpl) NewSubscription(ctx context.Context, topic string) error {
	return e.NewSubscriptionWithOptions(ctx, topic, "", nil, false)
}

func (e *EventClientImpl) NewSubscriptionWithOptions(ctx context.Context, topic string, aggregateType string, aggregateID *int, isRegex bool) error {
	// call the API to create a new subscription
	e.Logger.Info("creating new subscription", "topic", topic, "aggregate_type", aggregateType, "aggregate_id", aggregateID, "is_regex", isRegex)
	token, err := e.GetTokenCallback(ctx)
	if err != nil {
		e.Logger.Error("failed to get token", "error", err)
		return err
	}
	e.Logger.Debug("got token for subscription")

	subscription := SubscriptionCreate{
		Topic:         topic,
		SubscriberID:  e.Subscriber.ID,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		IsRegex:       isRegex,
	}

	jsonData, err := json.Marshal(subscription)
	if err != nil {
		e.Logger.Error("failed to marshal subscription", "error", err)
		return err
	}

	url := e.Options.EventAPIURL + "/v2/subscriptions"
	e.Logger.Debug("subscription URL", "url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		e.Logger.Error("failed to create request", "error", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		e.Logger.Error("failed to send request", "error", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		e.Logger.Error("failed to create subscription", "status_code", resp.StatusCode, "data", string(errData))
		return fmt.Errorf("failed to create subscription: %s", string(errData))
	}

	var createdSubscription Subscription
	if err := json.NewDecoder(resp.Body).Decode(&createdSubscription); err != nil {
		e.Logger.Error("failed to decode response", "error", err)
		return err
	}
	e.Logger.Info("subscription created", "subscription_id", createdSubscription.ID)
	return nil
}

// NewAggregateTypeSubscription creates a subscription for all messages of a specific aggregate type
func (e *EventClientImpl) NewAggregateTypeSubscription(ctx context.Context, topic string, aggregateType string, isRegex bool) error {
	return e.NewSubscriptionWithOptions(ctx, topic, aggregateType, nil, isRegex)
}

// NewAggregateSubscription creates a subscription for a specific aggregate type and ID
func (e *EventClientImpl) NewAggregateSubscription(ctx context.Context, topic string, aggregateType string, aggregateID int, isRegex bool) error {
	return e.NewSubscriptionWithOptions(ctx, topic, aggregateType, &aggregateID, isRegex)
}

// GetOAuthToken is a public convenience function for obtaining OAuth tokens
func GetOAuthToken(ctx context.Context, oauthURL, clientID, clientSecret string) (string, error) {
	requestBody := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"grant_type":    "client_credentials",
		"request_type":  "api",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get token: status %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}

// ListHandlers returns information about currently registered handlers (for debugging)
func (e *EventClientImpl) ListHandlers() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	patterns := make([]string, len(e.handlers))
	for i, handler := range e.handlers {
		patterns[i] = handler.pattern.String()
	}

	e.Logger.Info("LISTHANDLERS: Current registered handlers",
		"total_handlers", len(patterns),
		"patterns", patterns)

	return patterns
}

// LogHandlerState logs the current state of all handlers (for debugging)
func (e *EventClientImpl) LogHandlerState() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	logger := e.Logger
	logger.Info("HANDLER_STATE: Current handler registry", "total_handlers", len(e.handlers))

	for i, handler := range e.handlers {
		logger.Info("HANDLER_STATE: Handler details",
			"handler_index", i,
			"pattern", handler.pattern.String())
	}
}
