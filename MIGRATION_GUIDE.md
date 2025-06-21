# Migration Guide: Go Event Client v2.0

This guide will help you migrate from the previous version of the Go Event Client to the new version that supports UUID IDs, aggregate fields, and regex topic matching.

## Breaking Changes

### 1. ID Fields Changed from `int` to `string` (UUID)

**Before:**
```go
type Message struct {
    ID           int    `json:"id"`
    ConnectionID int    `json:"connection_id"`
    SessionID    int    `json:"session_id"`
    // ...
}

type Subscriber struct {
    ID int `json:"id"`
    // ...
}
```

**After:**
```go
type Message struct {
    ID           string `json:"id"`
    ConnectionID string `json:"connection_id"`
    SessionID    string `json:"session_id"`
    // ...
}

type Subscriber struct {
    ID string `json:"id"`
    // ...
}
```

**Migration Steps:**
- Update any code that expects integer IDs to handle string UUIDs
- Remove any integer parsing/conversion logic for IDs
- Update logging and debugging code that displays IDs

### 2. NewSubscription Method Parameters

**Before:**
```go
func (client *EventClient) NewSubscription(ctx context.Context, topic string) error
```

**After:**
```go
// For backward compatibility, the original method still works for exact matching
func (client *EventClient) NewSubscription(ctx context.Context, topic string) error

// New method with full options
func (client *EventClient) NewSubscriptionWithOptions(ctx context.Context, topic string, aggregateType string, aggregateID *int, isRegex bool) error
```

**Migration Steps:**
- No immediate changes required - existing calls to `NewSubscription` will continue to work
- Consider migrating to new subscription methods for enhanced functionality

## New Features

### 1. Aggregate-Based Subscriptions

You can now subscribe to events based on aggregate types and specific aggregate IDs:

```go
// Subscribe to all events for a specific aggregate type
err := client.NewAggregateTypeSubscription(ctx, "node.events", "node", false)

// Subscribe to events for a specific aggregate instance
err := client.NewAggregateSubscription(ctx, "node.specific", "node", 123, false)
```

### 2. Regex Topic Matching

You can now use regex patterns for topic matching:

```go
// Subscribe with regex pattern matching
err := client.NewSubscriptionWithOptions(ctx, "user\\..*", "", nil, true)

// Subscribe to aggregate with regex
err := client.NewAggregateTypeSubscription(ctx, "node\\..*", "node", true)
```

### 3. Enhanced Message Structure

Messages now include optional aggregate information:

```go
type Message struct {
    // ... existing fields ...
    AggregateType string `json:"aggregate_type,omitempty"`
    AggregateID   *int   `json:"aggregate_id,omitempty"`
}
```

## Updated Usage Examples

### Basic Usage (Backward Compatible)

```go
// This code continues to work without changes
client := go_event_client.NewEventClient(ctx, options, getToken, logger)
err := client.AddHandler("^example\\..*$", handleMessage)
err := client.Start()
err := client.NewSubscription(ctx, "example.topic")
```

### New Advanced Usage

```go
// Using the new aggregate features
client := go_event_client.NewEventClient(ctx, options, getToken, logger)

// Handler for all user events
err := client.AddHandler("^user\\..*$", func(message string) {
    fmt.Printf("User event: %s\n", message)
})

// Handler for all node events
err := client.AddHandler("^node\\..*$", func(message string) {
    fmt.Printf("Node event: %s\n", message)
})

err := client.Start()

// Subscribe to all user events with regex
err := client.NewAggregateTypeSubscription(ctx, "user\\..*", "user", true)

// Subscribe to specific node events
err := client.NewAggregateSubscription(ctx, "node.updated", "node", 456, false)

// Subscribe with full options
err := client.NewSubscriptionWithOptions(ctx, "system\\..*", "system", nil, true)
```

## Testing Your Migration

### Environment Setup

1. Copy `.env.example` to `.env`
2. Update the OAuth credentials if needed
3. Run unit tests: `go test -v ./... -short`
4. Run integration tests: `go test -v ./...`

### Verification Steps

1. **Check ID Handling**: Ensure your code properly handles string UUIDs instead of integers
2. **Test Subscriptions**: Verify that your existing subscriptions still work
3. **Test Message Parsing**: Ensure your message handlers can parse the new message format
4. **Test New Features**: Experiment with aggregate subscriptions and regex matching

## Common Migration Issues

### Issue 1: Integer ID Conversion

**Problem:**
```go
// This will now fail
subscriberIDInt, err := strconv.Atoi(subscriber.ID)
```

**Solution:**
```go
// IDs are now UUIDs, use them as strings
subscriberID := subscriber.ID
```

### Issue 2: String Concatenation with IDs

**Problem:**
```go
// This still works but was updated internally
url := baseURL + "/subscriptions/?subscriber_id=" + strconv.Itoa(subscriber.ID)
```

**Solution:**
```go
// Now handled properly internally, but if you're doing this manually:
url := baseURL + "/subscriptions/?subscriber_id=" + subscriber.ID
```

### Issue 3: Message ID Comparisons

**Problem:**
```go
// This will no longer work
if message.ID > lastProcessedID {
    // process message
}
```

**Solution:**
```go
// Use string comparison or maintain a separate tracking mechanism
if message.ID != lastProcessedID {
    // process message
    lastProcessedID = message.ID
}
```

## Support

If you encounter issues during migration:

1. Check the unit tests for examples of correct usage
2. Review the updated sample code in `samples/sample.go`
3. Run the integration tests to verify your setup
4. Consult the updated API documentation

## Summary

The migration primarily involves:
1. Updating ID handling from integers to UUID strings
2. Optionally adopting new aggregate subscription features
3. Optionally using regex topic matching for more flexible subscriptions

Most existing code will continue to work with minimal changes, as we've maintained backward compatibility where possible.
