package go_event_client

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GetTokenCallback func(ctx context.Context) (string, error)

type EventClient interface {
	Start() error
	Stop() error
	AddHandler(expr string, handler func(string)) error
	Publish(topic string, v interface{}) error
	NewSubscription(ctx context.Context, topic string) error
}

type EventClientOptions struct {
	EventAPIURL  string
	SocketsURL   string
	PingInterval int
}

type handlerEntry struct {
	pattern *regexp.Regexp
	handler func(string)
}

type EventClientImpl struct {
	Ctx              context.Context
	Options          EventClientOptions
	GetTokenCallback GetTokenCallback
	Logger           *slog.Logger
	HTTPClient       *http.Client

	// internal values set during runtime
	Subscriber *Subscriber
	Session    *Session
	conn       *websocket.Conn
	send       chan []byte
	handlers   []handlerEntry
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
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
	connURI := e.Options.SocketsURL + "/ws/" + e.Session.Token
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
	req, err := http.NewRequestWithContext(e.Ctx, http.MethodPost, e.Options.EventAPIURL+"/subscribers", nil)
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
		e.Logger.Error("failed to register subscriber", "status_code", resp.StatusCode, "data", errData)
		return err
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
		return nil
	}
	token, err := e.GetTokenCallback(e.Ctx)
	if err != nil {
		logger.Error("failed to get token", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(e.Ctx, http.MethodPost, e.Options.EventAPIURL+"/sessions", nil)
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
		logger.Error("failed to request session", "status_code", resp.StatusCode, "data", errData)
		return err
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

	// Set limits and pong handler
	e.conn.SetReadLimit(512)
	e.conn.SetReadDeadline(time.Now().Add(time.Duration(e.Options.PingInterval) * time.Second))
	e.conn.SetPongHandler(func(appData string) error {
		e.conn.SetReadDeadline(time.Now().Add(time.Duration(e.Options.PingInterval) * time.Second))
		return nil
	})

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			_, data, err := e.conn.ReadMessage()
			if err != nil {
				return
			}
			// Dispatch matching handlers in their own goroutines
			go e.dispatch(data)
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
	var m Message
	if err := json.Unmarshal(raw, &m); err != nil {
		// invalid frame
		return
	}

	e.mu.RLock()
	entries := append([]handlerEntry(nil), e.handlers...)
	e.mu.RUnlock()

	for _, entry := range entries {
		if entry.pattern.MatchString(m.Topic) {
			go entry.handler(m.Message)
		}
	}
}

// AddHandler registers a callback for topics matching the given regex
// The expr should be a valid Go regex (e.g. "^user\\..*$" to match "user.*").
func (e *EventClientImpl) AddHandler(expr string, handler func(string)) error {
	re, err := regexp.Compile(expr)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.handlers = append(e.handlers, handlerEntry{pattern: re, handler: handler})
	e.mu.Unlock()
	return nil
}

// Publish sends a topic and payload. It blocks only if the send buffer is full.
func (e *EventClientImpl) Publish(topic string, v interface{}) error {
	msg := Message{Topic: topic}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	msg.Message = string(data)

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

func (e *EventClientImpl) Stop() error {
	e.cancel()
	if e.conn != nil {
		e.conn.Close()
	}
	return nil
}

func (e *EventClientImpl) NewSubscription(ctx context.Context, topic string) error {
	// call the API to create a new subscription
	token, err := e.GetTokenCallback(ctx)
	if err != nil {
		e.Logger.Error("failed to get token", "error", err)
		return err
	}

	url := e.Options.EventAPIURL + "/subscriptions/?topic=" + topic + "&subscriber_id=" + strconv.Itoa(e.Subscriber.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		e.Logger.Error("failed to create request", "error", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		errData, _ := io.ReadAll(resp.Body)
		e.Logger.Error("failed to send request", "error", err, "data", errData)
		return err
	}
	if resp.StatusCode >= 400 {
		errData, _ := io.ReadAll(resp.Body)
		e.Logger.Error("failed to create subscription", "status_code", resp.StatusCode, "data", errData)
		return err
	}
	var subscription Subscription
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		e.Logger.Error("failed to decode response", "error", err)
		return err
	}
	e.Logger.Info("subscription created", "subscription_id", subscription.ID)
	return nil
}
