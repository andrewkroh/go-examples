package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Process enriches incoming JSON webhooks with metadata before publishing.
// It adds a timestamp and processing metadata to each event.
func Process(ctx context.Context, payload []byte, publish func([]byte) error) error {
	// Parse the incoming JSON
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to parse JSON payload: %w", err)
	}

	// Enrich with metadata
	data["@timestamp"] = time.Now().UTC().Format(time.RFC3339)
	data["processor"] = map[string]any{
		"name":    "webhook-handler",
		"version": "1.0.0",
	}

	// Get environment variables if available
	if env, ok := ctx.Value("env").(map[string]string); ok {
		if source := env["SOURCE"]; source != "" {
			data["source"] = source
		}
	}

	// Marshal back to JSON
	enriched, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal enriched data: %w", err)
	}

	// Publish the enriched event
	return publish(enriched)
}
