# Go Event Client

A Go client library for the Ambient Event Bus that provides real-time event messaging with WebSocket connections, OAuth authentication, and advanced subscription features.

## Features

- **Real-time messaging** via WebSocket connections
- **OAuth 2.0 authentication** with automatic token management
- **Regex pattern matching** for flexible topic subscriptions
- **Aggregate-based subscriptions** for type-specific and resource-specific events
- **HTTP API publishing** in addition to WebSocket publishing
- **UUID support** for all identifiers
- **Comprehensive testing** with unit and integration tests
- **Type-safe** Go structs for all event data
- **Connection resilience** with ping/pong heartbeat mechanism

## Installation

```bash
go get github.com/ambientlabscomputing/go_event_client
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    
    "github.com/ambientlabscomputing/go_event_client"
)

func main() {
    // Create logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    
    // Configure client options
    options := go_event_client.EventClientOptions{
        EventAPIURL:  "http://events.ambientlabsdev.io",
        SocketsURL:   "wss://sockets.ambientlabsdev.io",
        PingInterval: 30, // seconds
    }
    
    // OAuth token callback function
    getToken := func(ctx context.Context) (string, error) {
        return go_event_client.GetOAuthToken(ctx, 
            "https://oauth.ambientlabsdev.io/oauth/token",
            "your-client-id", 
            "your-client-secret")
    }
    
    // Create client
    client := go_event_client.NewEventClient(context.Background(), options, getToken, logger)
    
    // Add message handler
    err := client.AddHandler("^user\\..*", func(message string) {
        fmt.Printf("Received user event: %s\n", message)
    })
    if err != nil {
        panic(err)
    }
    
    // Start client
    if err := client.Start(); err != nil {
        panic(err)
    }
    defer client.Stop()
    
    // Create subscription
    err = client.NewSubscription(context.Background(), "user.created")
    if err != nil {
        panic(err)
    }
    
    // Publish message
    err = client.Publish("user.created", map[string]interface{}{
        "user_id": 123,
        "email":   "user@example.com",
    })
    if err != nil {
        panic(err)
    }
}
```

## Advanced Features

### Aggregate-Based Subscriptions

Subscribe to events based on resource types and specific resource IDs:

```go
// Subscribe to all events for a specific aggregate type
err := client.NewAggregateTypeSubscription(ctx, "node.events", "node", false)

// Subscribe to events for a specific aggregate instance
err := client.NewAggregateSubscription(ctx, "user.events", "user", 123, false)

// Publish with aggregate information
err := client.PublishWithAggregate("user.updated", userData, "user", &userID)
```

### Regex Topic Matching

Use regex patterns for flexible topic matching:

```go
// Regex subscription for all user events
err := client.NewSubscriptionWithOptions(ctx, "user\\..*", "", nil, true)

// Regex with aggregate type
err := client.NewAggregateTypeSubscription(ctx, "node\\.(created|updated)", "node", true)
```

### HTTP API Publishing

In addition to WebSocket publishing, you can publish messages via HTTP API:

```go
// Publish via HTTP API (requires established WebSocket connection for connection_id)
err := client.PublishViaAPI(ctx, "user.notification", messageData, "user", &userID)
```

### Message Structure

Messages include comprehensive metadata and optional aggregate information:

```go
type Message struct {
    ID            string `json:"id"`              // UUID
    CreatedAt     string `json:"created_at"`     // ISO timestamp
    Topic         string `json:"topic"`          // Event topic
    Message       string `json:"message"`        // JSON payload
    SubscriberID  string `json:"subscriber_id"`  // Subscriber UUID
    SessionID     string `json:"session_id"`     // Session UUID
    ConnectionID  string `json:"connection_id"`  // Connection UUID
    Timestamp     string `json:"timestamp"`      // Processing timestamp
    AggregateType string `json:"aggregate_type,omitempty"` // Resource type
    AggregateID   *int   `json:"aggregate_id,omitempty"`   // Resource ID
}
```

## Configuration

### Environment Variables

Create a `.env` file with your configuration:

```bash
AMBIENT_EVENT_API_URL=http://events.ambientlabsdev.io
AMBIENT_SOCKETS_URL=wss://sockets.ambientlabsdev.io
AMBIENT_OAUTH_URL=https://oauth.ambientlabsdev.io/oauth/token
AMBIENT_CLIENT_ID=your-client-id
AMBIENT_CLIENT_SECRET=your-client-secret
```

### OAuth Token Management

The client includes a built-in OAuth token helper:

```go
token, err := go_event_client.GetOAuthToken(ctx, oauthURL, clientID, clientSecret)
```

## API Reference

### Client Interface

```go
type EventClient interface {
    Start() error
    Stop() error
    AddHandler(expr string, handler func(string)) error
    Publish(topic string, v interface{}) error
    PublishWithAggregate(topic string, v interface{}, aggregateType string, aggregateID *int) error
    PublishViaAPI(ctx context.Context, topic string, v interface{}, aggregateType string, aggregateID *int) error
    NewSubscription(ctx context.Context, topic string) error
    NewSubscriptionWithOptions(ctx context.Context, topic string, aggregateType string, aggregateID *int, isRegex bool) error
    NewAggregateTypeSubscription(ctx context.Context, topic string, aggregateType string, isRegex bool) error
    NewAggregateSubscription(ctx context.Context, topic string, aggregateType string, aggregateID int, isRegex bool) error
}
```

### Subscription Methods

| Method | Description |
|--------|-------------|
| `NewSubscription()` | Basic topic subscription |
| `NewSubscriptionWithOptions()` | Full control over subscription options |
| `NewAggregateTypeSubscription()` | Subscribe to all events of an aggregate type |
| `NewAggregateSubscription()` | Subscribe to events for a specific aggregate instance |

### Publishing Methods

| Method | Description |
|--------|-------------|
| `Publish()` | Basic message publishing via WebSocket |
| `PublishWithAggregate()` | Publish with aggregate information via WebSocket |
| `PublishViaAPI()` | Publish via HTTP API |

## Make Commands

The project includes a comprehensive Makefile for easy development and testing:

```bash
# Show all available commands
make help

# Run all tests
make test

# Run specific test types
make test-unit              # Unit tests only
make test-integration       # Integration tests only
make test-models           # Model/structure tests
make test-oauth            # OAuth authentication tests

# Run tests by functionality
make test-basic            # Basic subscription test
make test-aggregate        # Aggregate subscription tests
make test-regex            # Regex subscription test

# Build and examples
make build                 # Build library and examples
make run-demo             # Run comprehensive demo
make examples             # Build all examples

# Development helpers
make dev-setup            # Set up development environment
make check-env            # Check environment configuration
make format               # Format Go code
make lint                 # Run linter
make clean                # Clean build artifacts

# Verbose output (add V=1 to any test command)
make test-unit V=1        # Run unit tests with verbose output
```

For a complete list of commands, run `make help`.

## Examples

See the [`examples/`](./examples/) directory for comprehensive examples demonstrating all features:

- **Basic subscriptions and publishing**
- **Aggregate type subscriptions**  
- **Specific aggregate subscriptions**
- **Regex pattern subscriptions**
- **HTTP API publishing**
- **Real-time message handling**

Run the comprehensive demo:

```bash
cd examples
cp .env.example .env
# Edit .env with your credentials
source .env && go run comprehensive_demo.go
```

## Testing

The library includes comprehensive test coverage:

```bash
# Run unit tests
go test ./...

# Run integration tests (requires .env configuration)
go test -v -run "TestIntegration"

# Run OAuth token test
go test -v -run "TestOAuth"
```

### Integration Test Requirements

Integration tests require a `.env` file with valid credentials:

```bash
cp .env.example .env
# Edit .env with your actual API credentials
```

## API Endpoints

The client uses the following API v2 endpoints:

- `POST /v2/subscribers` - Register event subscriber
- `POST /v2/sessions` - Create WebSocket session
- `POST /v2/subscriptions` - Create event subscriptions
- `POST /v2/messages` - Publish messages via HTTP API
- `GET /ws/{token}` - WebSocket connection endpoint

## Error Handling

The client provides detailed error logging and handles various failure scenarios:

- **Connection failures** with automatic cleanup
- **Authentication errors** with clear error messages
- **API errors** with detailed response information
- **WebSocket disconnections** with graceful shutdown

## Changelog

### Latest Version

- ✅ **OAuth Authentication**: Built-in OAuth 2.0 client credentials flow
- ✅ **UUID Support**: All identifiers use UUID format  
- ✅ **Aggregate Subscriptions**: Type-based and instance-based event filtering
- ✅ **Regex Topics**: Pattern matching for flexible subscriptions
- ✅ **HTTP API Publishing**: Alternative to WebSocket publishing
- ✅ **Enhanced Message Structure**: Complete metadata and aggregate information
- ✅ **v2 API Endpoints**: Updated to use latest API version
- ✅ **Integration Testing**: Comprehensive end-to-end testing
- ✅ **Connection Management**: Improved WebSocket connection handling
- ✅ **Error Handling**: Enhanced error messages and logging

## License

[MIT License](LICENSE)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

For bugs and feature requests, please create an issue.
