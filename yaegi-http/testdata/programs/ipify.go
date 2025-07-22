package programs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	env, ok := ctx.Value("env").(map[string]string)
	if !ok {
		return fmt.Errorf("failed to get config")
	}
	url, found := env["URL"]
	if !found {
		return fmt.Errorf("failed to get URL")
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"?format=json", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(r)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}

	callback(data)
	return nil
}
