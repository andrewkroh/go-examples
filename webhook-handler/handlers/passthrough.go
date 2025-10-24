package handler

import (
	"context"
)

// Process is a simple passthrough handler that forwards the webhook payload
// as-is to Kafka without any processing.
func Process(ctx context.Context, payload []byte, publish func([]byte) error) error {
	return publish(payload)
}
