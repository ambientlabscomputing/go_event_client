package go_event_client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
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
	EventAPIURL          string
	SocketsURL           string
	PingInterval         int
	MaxReconnectAttempts int           // Maximum number of reconnection attempts (0 = infinite)
	ReconnectBackoff     time.Duration // Initial backoff duration between reconnection attempts
	MaxReconnectBackoff  time.Duration // Maximum backoff duration
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

	// reconnection state
	reconnectMu      sync.Mutex
	reconnectCount   int
	isReconnecting   bool
	reconnectTrigger chan struct{}

	// connection state management
	connMu     sync.Mutex
	pumpCtx    context.Context
	pumpCancel context.CancelFunc
}

func NewEventClient(ctx context.Context, options EventClientOptions, getTokenCallback GetTokenCallback, logger *slog.Logger) EventClient {
	// Set default reconnection options if not specified
	if options.MaxReconnectAttempts == 0 && options.ReconnectBackoff == 0 {
		options.MaxReconnectAttempts = 0 // Infinite retries by default
		options.ReconnectBackoff = 1 * time.Second
		options.MaxReconnectBackoff = 30 * time.Second
	}

	return &EventClientImpl{
		Ctx:              ctx,
		Options:          options,
		GetTokenCallback: getTokenCallback,
		Logger:           logger,
		HTTPClient:       &http.Client{},
		reconnectTrigger: make(chan struct{}, 1),
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

	// Start the reconnection manager
	go e.reconnectionManager()

	// Trigger initial connection
	e.triggerReconnect()

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
func (e *EventClientImpl) readPump(pumpCtx context.Context) {
	logger := e.Logger

	// Set limits and pong handler
	logger.Info("READPUMP: Starting read pump")
	logger.Debug("READPUMP: Setting read limits and pong handler")
	e.conn.SetReadLimit(65536) // 64KB limit

	// Set read deadline to 2x ping interval to allow for network latency
	readTimeout := time.Duration(e.Options.PingInterval*2) * time.Second
	e.conn.SetReadDeadline(time.Now().Add(readTimeout))
	logger.Debug("READPUMP: Read timeout configured", "ping_interval", e.Options.PingInterval, "read_timeout", readTimeout)

	e.conn.SetPongHandler(func(appData string) error {
		logger.Debug("READPUMP: Received pong", "app_data", appData)
		// Extend read deadline when we receive a pong
		e.conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})
	logger.Info("READPUMP: Read pump configured and started")

	messageCount := 0
	for {
		select {
		case <-pumpCtx.Done():
			logger.Info("READPUMP: Pump context cancelled, stopping read pump", "messages_processed", messageCount)
			return
		case <-e.ctx.Done():
			logger.Info("READPUMP: Client context cancelled, stopping read pump", "messages_processed", messageCount)
			return
		default:
			logger.Debug("READPUMP: Waiting for WebSocket message...")
			_, data, err := e.conn.ReadMessage()
			if err != nil {
				// Check if this is a graceful shutdown
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					logger.Info("READPUMP: WebSocket closed gracefully", "error", err, "messages_processed", messageCount)
				} else if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					logger.Error("READPUMP: WebSocket read timeout - server may not be responding to pings",
						"error", err,
						"messages_processed", messageCount,
						"ping_interval", e.Options.PingInterval,
						"read_timeout", time.Duration(e.Options.PingInterval*2)*time.Second)
				} else {
					logger.Error("READPUMP: Failed to read WebSocket message", "error", err, "messages_processed", messageCount)
				}

				// Trigger reconnection for all error types except context cancellation
				select {
				case <-pumpCtx.Done():
					// Don't trigger reconnect if pump context is cancelled
				case <-e.ctx.Done():
					// Don't trigger reconnect if client context is cancelled
				default:
					e.triggerReconnect()
				}
				return
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
func (e *EventClientImpl) writePump(pumpCtx context.Context) {
	pingInterval := time.Duration(e.Options.PingInterval) * time.Second
	ticker := time.NewTicker(pingInterval)
	e.Logger.Debug("WRITEPUMP: Starting write pump", "ping_interval_seconds", e.Options.PingInterval, "ping_interval_duration", pingInterval)
	defer func() {
		ticker.Stop()
		if e.conn != nil {
			e.conn.Close()
		}
	}()

	for {
		select {
		case msg := <-e.send:
			if e.conn == nil {
				e.Logger.Warn("WRITEPUMP: No connection available, dropping message")
				continue
			}
			e.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := e.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				e.Logger.Error("WRITEPUMP: Failed to send message", "error", err)
				// Only trigger reconnect if not shutting down
				select {
				case <-pumpCtx.Done():
					// Don't trigger reconnect if pump context is cancelled
				case <-e.ctx.Done():
					// Don't trigger reconnect if client context is cancelled
				default:
					e.triggerReconnect()
				}
				return
			}

		case <-ticker.C:
			if e.conn == nil {
				e.Logger.Debug("WRITEPUMP: No connection available, skipping ping")
				continue
			}
			e.Logger.Debug("WRITEPUMP: Sending ping message")
			e.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := e.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				e.Logger.Error("WRITEPUMP: Failed to send ping", "error", err)
				// Only trigger reconnect if not shutting down
				select {
				case <-pumpCtx.Done():
					// Don't trigger reconnect if pump context is cancelled
				case <-e.ctx.Done():
					// Don't trigger reconnect if client context is cancelled
				default:
					e.triggerReconnect()
				}
				return
			}
			e.Logger.Debug("WRITEPUMP: Ping message sent successfully")

		case <-pumpCtx.Done():
			e.Logger.Info("WRITEPUMP: Pump context cancelled, stopping write pump")
			return
		case <-e.ctx.Done():
			e.Logger.Info("WRITEPUMP: Client context cancelled, stopping write pump")
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

// triggerReconnect signals the reconnection manager to attempt reconnection
func (e *EventClientImpl) triggerReconnect() {
	e.reconnectMu.Lock()
	defer e.reconnectMu.Unlock()

	if e.isReconnecting {
		e.Logger.Debug("RECONNECT: Reconnection already in progress, ignoring trigger")
		return
	}

	e.Logger.Info("RECONNECT: Triggering reconnection attempt")
	select {
	case e.reconnectTrigger <- struct{}{}:
		e.Logger.Debug("RECONNECT: Reconnection trigger sent")
	default:
		e.Logger.Debug("RECONNECT: Reconnection trigger channel already full")
	}
}

// reconnectionManager handles WebSocket reconnection logic
func (e *EventClientImpl) reconnectionManager() {
	logger := e.Logger
	logger.Info("RECONNECT_MANAGER: Starting reconnection manager")

	for {
		select {
		case <-e.ctx.Done():
			logger.Info("RECONNECT_MANAGER: Context cancelled, stopping reconnection manager")
			return
		case <-e.reconnectTrigger:
			e.reconnectMu.Lock()
			if e.isReconnecting {
				e.reconnectMu.Unlock()
				continue
			}
			e.isReconnecting = true
			e.reconnectMu.Unlock()

			logger.Info("RECONNECT_MANAGER: Starting reconnection process", "current_attempt", e.reconnectCount)

			// Stop existing pumps and close connection
			e.connMu.Lock()
			if e.pumpCancel != nil {
				logger.Debug("RECONNECT_MANAGER: Stopping existing pumps")
				e.pumpCancel()
				e.pumpCancel = nil
			}
			if e.conn != nil {
				e.conn.Close()
				e.conn = nil
			}
			e.connMu.Unlock()

			// Small delay to ensure old goroutines have time to exit
			time.Sleep(100 * time.Millisecond)

			// Attempt reconnection with backoff
			success := e.attemptReconnection()

			e.reconnectMu.Lock()
			e.isReconnecting = false
			if success {
				e.reconnectCount = 0
				logger.Info("RECONNECT_MANAGER: Reconnection successful, resetting attempt counter")
			} else {
				logger.Error("RECONNECT_MANAGER: Reconnection failed, giving up")
			}
			e.reconnectMu.Unlock()
		}
	}
}

// attemptReconnection performs the actual reconnection with exponential backoff
func (e *EventClientImpl) attemptReconnection() bool {
	logger := e.Logger
	backoff := e.Options.ReconnectBackoff

	for attempt := 1; ; attempt++ {
		// Check if we should stop trying
		if e.Options.MaxReconnectAttempts > 0 && attempt > e.Options.MaxReconnectAttempts {
			logger.Error("RECONNECT: Maximum reconnection attempts reached",
				"max_attempts", e.Options.MaxReconnectAttempts,
				"total_attempts", attempt-1)
			return false
		}

		// Check if context is cancelled
		select {
		case <-e.ctx.Done():
			logger.Info("RECONNECT: Context cancelled during reconnection attempt", "attempt", attempt)
			return false
		default:
		}

		logger.Info("RECONNECT: Attempting to reconnect",
			"attempt", attempt,
			"max_attempts", e.Options.MaxReconnectAttempts,
			"backoff_duration", backoff)

		// Try to establish connection
		if err := e.establishConnection(); err != nil {
			logger.Error("RECONNECT: Failed to establish connection",
				"attempt", attempt,
				"error", err,
				"retry_in", backoff)

			// Wait before next attempt
			select {
			case <-time.After(backoff):
				// Exponential backoff with jitter
				backoff = time.Duration(float64(backoff) * 1.5)
				if backoff > e.Options.MaxReconnectBackoff {
					backoff = e.Options.MaxReconnectBackoff
				}
			case <-e.ctx.Done():
				logger.Info("RECONNECT: Context cancelled during backoff", "attempt", attempt)
				return false
			}
			continue
		}

		logger.Info("RECONNECT: Successfully reconnected",
			"attempt", attempt,
			"total_reconnections", e.reconnectCount+1)
		e.reconnectCount++
		return true
	}
}

// establishConnection creates a new WebSocket connection and starts the pumps
func (e *EventClientImpl) establishConnection() error {
	logger := e.Logger

	if e.Session == nil {
		return fmt.Errorf("session not available for connection")
	}

	connURI := e.Options.SocketsURL + "/ws/" + e.Session.ID
	logger.Info("RECONNECT: Connecting to WebSocket", "url", connURI)

	conn, _, err := websocket.DefaultDialer.Dial(connURI, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	e.connMu.Lock()
	e.conn = conn
	// Create a new context for the pumps that can be cancelled independently
	e.pumpCtx, e.pumpCancel = context.WithCancel(e.ctx)
	pumpCtx := e.pumpCtx
	e.connMu.Unlock()

	logger.Info("RECONNECT: WebSocket connection established")

	// Start the pumps with the pump-specific context
	go e.writePump(pumpCtx)
	go e.readPump(pumpCtx)

	logger.Info("RECONNECT: Read and write pumps started")
	return nil
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
	e.Logger.Info("STOP: Stopping event client")

	// Cancel pump contexts first to stop the goroutines
	e.connMu.Lock()
	if e.pumpCancel != nil {
		e.pumpCancel()
		e.pumpCancel = nil
	}
	e.connMu.Unlock()

	// Cancel main context to stop all goroutines
	if e.cancel != nil {
		e.cancel()
	}

	// Close WebSocket connection
	e.connMu.Lock()
	if e.conn != nil {
		e.conn.Close()
		e.conn = nil
	}
	e.connMu.Unlock()

	e.Logger.Info("STOP: Event client stopped")
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
