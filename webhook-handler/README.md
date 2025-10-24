# Webhook Handler

A flexible webhook handler service that accepts HTTP webhook callbacks and processes them using user-defined Go handlers executed via Yaegi. Events are published to Warpstream (Kafka).

## Features

- **HTTP Webhook Endpoint**: Receives POST requests at `/webhook`
- **Payload Size Limits**: Configurable maximum payload size enforcement
- **User-Defined Handlers**: Write custom processing logic in Go, executed safely via Yaegi
- **Kafka/Warpstream Integration**: Publish processed events to Kafka topics
- **Security Sandboxing**: Optional Linux sandboxing using seccomp-bpf and landlock-lsm
- **Environment Variables**: Pass configuration to handlers via `WEBHOOK_*` environment variables

## Architecture

```
Webhook (HTTP POST) → Payload Size Check → Yaegi Handler → Kafka/Warpstream
```

The service:
1. Receives webhook payloads via HTTP POST
2. Enforces payload size limits
3. Executes user-defined handler via Yaegi
4. Handler processes payload and publishes events using the provided `publish` function
5. Events are sent to Kafka/Warpstream

## Handler Interface

User-defined handlers must implement a `Process` function with this signature:

```go
func Process(ctx context.Context, payload []byte, publish func([]byte) error) error
```

### Parameters

- `ctx`: Context with timeout and optional environment variables (accessible via `ctx.Value("env")`)
- `payload`: Raw webhook payload bytes
- `publish`: Function to publish events to Kafka (can be called multiple times)

### Handler Contract

- Handlers can call `publish()` zero or more times
- Each call to `publish()` sends one event to Kafka
- Returning an error will result in a 500 response to the webhook caller
- Returning `nil` results in a 200 OK response

## Usage

### Basic Usage

```bash
# Start the webhook handler with a simple passthrough handler
./webhook-handler \
  -handler=handlers/passthrough.go \
  -kafka-brokers=localhost:9092 \
  -kafka-topic=webhooks \
  -listen=:8080
```

### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `-handler` | Handler program file (required) | - |
| `-listen` | HTTP server listen address | `:8080` |
| `-max-payload-size` | Maximum payload size in bytes | `10485760` (10MB) |
| `-kafka-brokers` | Comma-separated Kafka broker addresses | `localhost:9092` |
| `-kafka-topic` | Kafka topic for events | `webhooks` |
| `-restrict` | Restrict stdlib access | `true` |
| `-seccomp` | Enable seccomp-bpf sandboxing | `false` |
| `-landlock-fs` | Enable filesystem sandboxing | `false` |
| `-landlock-net` | Enable network sandboxing | `false` |

### Environment Variables

Pass configuration to handlers using environment variables prefixed with `WEBHOOK_`:

```bash
export WEBHOOK_SOURCE="github"
export WEBHOOK_ALLOWED_LEVELS="error,warning,critical"

./webhook-handler -handler=handlers/filtering.go
```

Access in handler:
```go
if env, ok := ctx.Value("env").(map[string]string); ok {
    source := env["SOURCE"]  // "github"
    levels := env["ALLOWED_LEVELS"]  // "error,warning,critical"
}
```

## Example Handlers

### 1. Passthrough Handler

Forwards webhook payload directly to Kafka without modification.

```bash
./webhook-handler -handler=handlers/passthrough.go
```

### 2. JSON Enrichment Handler

Enriches JSON payloads with metadata (timestamp, processor info).

```bash
export WEBHOOK_SOURCE="my-application"
./webhook-handler -handler=handlers/json_enrichment.go
```

Input:
```json
{"event": "user.login", "user_id": "123"}
```

Output:
```json
{
  "event": "user.login",
  "user_id": "123",
  "@timestamp": "2024-01-15T10:30:00Z",
  "processor": {
    "name": "webhook-handler",
    "version": "1.0.0"
  },
  "source": "my-application"
}
```

### 3. JSON Array Splitter Handler

Splits webhook payloads containing arrays into individual events.

```bash
./webhook-handler -handler=handlers/json_array_splitter.go
```

Input (single webhook):
```json
[
  {"event": "user.login", "user_id": "123"},
  {"event": "user.logout", "user_id": "456"}
]
```

Output (two separate Kafka messages):
```json
{"event": "user.login", "user_id": "123", "@timestamp": "...", "event_index": 0, "total_events": 2}
{"event": "user.logout", "user_id": "456", "@timestamp": "...", "event_index": 1, "total_events": 2}
```

### 4. Filtering Handler

Filters events based on criteria (e.g., log level).

```bash
export WEBHOOK_ALLOWED_LEVELS="error,warning"
./webhook-handler -handler=handlers/filtering.go
```

Only events with `"level": "error"` or `"level": "warning"` will be published.

## Writing Custom Handlers

Create a new handler file:

```go
package handler

import (
    "context"
    "encoding/json"
    "fmt"
)

func Process(ctx context.Context, payload []byte, publish func([]byte) error) error {
    // 1. Parse payload
    var data map[string]any
    if err := json.Unmarshal(payload, &data); err != nil {
        return fmt.Errorf("parse error: %w", err)
    }

    // 2. Process/transform data
    data["processed"] = true

    // 3. Publish event(s)
    processed, _ := json.Marshal(data)
    return publish(processed)
}
```

### Handler Best Practices

1. **Validate Input**: Always validate the payload format
2. **Handle Errors**: Return descriptive errors for debugging
3. **Idempotency**: Design handlers to be idempotent when possible
4. **Multiple Events**: Call `publish()` multiple times to fan-out events
5. **Context**: Respect context cancellation for timeouts

## Testing

### Send a Test Webhook

```bash
# Passthrough handler
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'

# Array splitter handler
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '[{"event": "test1"}, {"event": "test2"}]'

# Filtering handler (with allowed level)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"level": "error", "message": "Something went wrong"}'
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Security

### Sandboxing Options

The service supports Linux kernel-level sandboxing:

#### Seccomp-BPF
Restricts system calls available to the handler process.

```bash
./webhook-handler -handler=handlers/passthrough.go -seccomp
```

#### Landlock-LSM
Restricts filesystem and network access.

```bash
# Restrict filesystem access
./webhook-handler -handler=handlers/passthrough.go -landlock-fs

# Restrict network access
./webhook-handler -handler=handlers/passthrough.go -landlock-net
```

#### Stdlib Restrictions
By default, handlers cannot access OS-level packages (os, net, etc.). Disable with:

```bash
./webhook-handler -handler=handlers/passthrough.go -restrict=false
```

## Kafka/Warpstream Integration

Events are published to Kafka using the [segmentio/kafka-go](https://github.com/segmentio/kafka-go) library with:

- **Compression**: Snappy compression enabled
- **Batching**: 10ms batch timeout for efficient throughput
- **Load Balancing**: LeastBytes balancing strategy

### Warpstream Configuration

To use with Warpstream (Kafka-compatible):

```bash
./webhook-handler \
  -handler=handlers/passthrough.go \
  -kafka-brokers=your-warpstream-cluster.com:9092 \
  -kafka-topic=webhooks
```

## Building

```bash
cd webhook-handler
go mod download
go build -o webhook-handler
```

## Example Deployment

### Docker Compose with Kafka

```yaml
version: '3.8'
services:
  kafka:
    image: apache/kafka:latest
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1

  webhook-handler:
    build: .
    ports:
      - "8080:8080"
    environment:
      - WEBHOOK_SOURCE=production
    command:
      - "-handler=/app/handlers/json_enrichment.go"
      - "-kafka-brokers=kafka:9092"
      - "-kafka-topic=webhooks"
    depends_on:
      - kafka
```

## Troubleshooting

### Handler Execution Fails

Check handler logs for errors. Common issues:
- Invalid function signature (must be `func Process(ctx context.Context, payload []byte, publish func([]byte) error) error`)
- Package name mismatch
- Syntax errors in handler code

### Payload Too Large

Increase the limit:
```bash
./webhook-handler -max-payload-size=52428800  # 50MB
```

### Kafka Connection Issues

Verify:
- Kafka brokers are accessible
- Topic exists or auto-creation is enabled
- Network connectivity

## License

See repository root for license information.
