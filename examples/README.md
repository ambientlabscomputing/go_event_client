# Examples

This directory contains examples demonstrating how to use the Go Event Client with the Ambient Labs Event API.

## Prerequisites

Before running the examples, you need to set up your environment variables. Copy the `.env.example` file to `.env` and fill in your credentials:

```bash
cp ../.env.example .env
```

Required environment variables:
- `AMBIENT_EVENT_API_URL` - The Event API URL
- `AMBIENT_SOCKETS_URL` - The WebSocket URL  
- `AMBIENT_CLIENT_ID` - Your OAuth client ID
- `AMBIENT_CLIENT_SECRET` - Your OAuth client secret
- `AMBIENT_OAUTH_URL` - The OAuth token endpoint URL

## Running Examples

### Comprehensive Demo

The comprehensive demo showcases all the main features of the event client:

```bash
# Load environment variables and run the demo
source .env && go run comprehensive_demo.go
```

This example demonstrates:
- ✅ Basic subscriptions and publishing
- ✅ Aggregate type subscriptions  
- ✅ Specific aggregate subscriptions
- ✅ Regex pattern subscriptions
- ✅ HTTP API publishing
- ✅ Real-time message handling

## Example Features Demonstrated

### 1. Basic Subscription
```go
// Subscribe to a specific topic
client.NewSubscription(ctx, "demo.basic.test")

// Publish a message  
client.Publish("demo.basic.test", message)
```

### 2. Aggregate Type Subscription
```go
// Subscribe to all events of a specific aggregate type
client.NewAggregateTypeSubscription(ctx, "demo.user.updated", "user", false)

// Publish with aggregate information
client.PublishWithAggregate("demo.user.updated", message, "user", nil)
```

### 3. Specific Aggregate Subscription  
```go
// Subscribe to events for a specific aggregate instance
client.NewAggregateSubscription(ctx, "demo.order.status", "order", 456, false)

// Publish with specific aggregate ID
aggregateID := 456
client.PublishWithAggregate("demo.order.status", message, "order", &aggregateID)
```

### 4. Regex Subscription
```go  
// Subscribe using regex pattern
client.NewSubscriptionWithOptions(ctx, "demo\\.system\\..*", "", nil, true)

// Publishes to demo.system.health, demo.system.metrics, etc. will all match
```

### 5. HTTP API Publishing
```go
// Publish via HTTP API instead of WebSocket
client.PublishViaAPI(ctx, "demo.basic.test", message, "", nil)
```

## Message Handling

All examples use message handlers to receive and process events:

```go
client.AddHandler("^demo\\.basic\\..*", func(message string) {
    fmt.Printf("Received: %s\n", message)
})
```

Handlers use regex patterns to match topics they want to process.
