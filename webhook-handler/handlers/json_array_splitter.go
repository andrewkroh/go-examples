package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Process handles webhooks that contain an array of events.
// It splits the array and publishes each event individually to Kafka.
func Process(ctx context.Context, payload []byte, publish func([]byte) error) error {
	// Parse the incoming JSON array
	var events []map[string]any
	if err := json.Unmarshal(payload, &events); err != nil {
		return fmt.Errorf("failed to parse JSON array payload: %w", err)
	}

	// Process each event in the array
	for i, event := range events {
		// Add metadata
		event["@timestamp"] = time.Now().UTC().Format(time.RFC3339)
		event["event_index"] = i
		event["total_events"] = len(events)

		// Marshal the individual event
		eventData, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %d: %w", i, err)
		}

		// Publish each event
		if err := publish(eventData); err != nil {
			return fmt.Errorf("failed to publish event %d: %w", i, err)
		}
	}

	return nil
}
