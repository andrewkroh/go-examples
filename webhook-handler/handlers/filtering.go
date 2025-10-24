package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Process filters incoming webhook events based on configurable criteria.
// Only events matching the filter criteria are published to Kafka.
// This example filters events based on a "level" field (e.g., "error", "warning").
func Process(ctx context.Context, payload []byte, publish func([]byte) error) error {
	// Parse the incoming JSON
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("failed to parse JSON payload: %w", err)
	}

	// Get filter configuration from environment
	var allowedLevels []string
	if env, ok := ctx.Value("env").(map[string]string); ok {
		if levels := env["ALLOWED_LEVELS"]; levels != "" {
			allowedLevels = strings.Split(levels, ",")
		}
	}

	// Default to error and warning if not configured
	if len(allowedLevels) == 0 {
		allowedLevels = []string{"error", "warning"}
	}

	// Check if event has a level field
	level, ok := data["level"].(string)
	if !ok {
		// If no level field, reject the event
		return fmt.Errorf("event missing required 'level' field")
	}

	// Check if level is in allowed list
	allowed := false
	for _, allowedLevel := range allowedLevels {
		if strings.EqualFold(strings.TrimSpace(allowedLevel), level) {
			allowed = true
			break
		}
	}

	if !allowed {
		// Skip this event (don't publish)
		return nil
	}

	// Marshal back to JSON
	filtered, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal filtered data: %w", err)
	}

	// Publish the filtered event
	return publish(filtered)
}
