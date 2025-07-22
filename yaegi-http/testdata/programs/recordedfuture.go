package programs

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

const (
	// Recorded Future API constants.
	rfTokenHeader = "X-RFToken"
	csvFormat     = "csv/splunk"
)

// Execute fetches a Recorded Future risk list, processes the response, and invokes a callback for each event.
// It uses an ETag to avoid refetching unchanged data.
func Execute(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	config, err := getConfig(ctx)
	if err != nil {
		return err
	}

	// Build request URL.
	reqURL, err := buildURL(config)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	// HEAD request to check ETag.
	newETag, err := checkETag(ctx, c, reqURL, config)
	if err != nil {
		return fmt.Errorf("failed to check ETag: %w", err)
	}
	if config.cursor != "" && newETag == config.cursor {
		return nil
	}

	// GET request to fetch data.
	resp, err := doRequest(ctx, c, http.MethodGet, reqURL, config)
	if err != nil {
		return fmt.Errorf("failed to execute GET request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Process response body.
	if err := processBody(resp.Body, callback); err != nil {
		return fmt.Errorf("failed to process response body: %w", err)
	}

	// TODO: Pass the new ETag into the state.
	log.Println("New ETag:", resp.Header.Get("ETag"))
	return nil
}

type rfConfig struct {
	url       string
	apiKey    string
	entity    string
	list      string
	cursor    string
	customURL string
}

func getConfig(ctx context.Context) (rfConfig, error) {
	env, ok := ctx.Value("env").(map[string]string)
	if !ok {
		return rfConfig{}, fmt.Errorf("failed to get config from context")
	}

	cfg := rfConfig{
		url:       env["URL"],
		apiKey:    env["API_KEY"],
		entity:    env["ENTITY"],
		list:      env["LIST"],
		cursor:    env["CURSOR"], // ETag from previous run.
		customURL: env["CUSTOM_URL"],
	}

	if cfg.apiKey == "" {
		return rfConfig{}, fmt.Errorf("API_KEY is required")
	}
	if cfg.customURL == "" && (cfg.url == "" || cfg.entity == "" || cfg.list == "") {
		return rfConfig{}, fmt.Errorf("URL, ENTITY, and LIST are required when CUSTOM_URL is not set")
	}

	return cfg, nil
}

func buildURL(config rfConfig) (string, error) {
	if config.customURL != "" {
		return config.customURL, nil
	}

	baseURL, err := url.Parse(config.url)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	baseURL = baseURL.JoinPath("v2", config.entity, "risklist")

	q := baseURL.Query()
	q.Set("format", csvFormat)
	q.Set("gzip", "true")
	q.Set("list", config.list)
	baseURL.RawQuery = q.Encode()

	return baseURL.String(), nil
}

func checkETag(ctx context.Context, c *http.Client, reqURL string, config rfConfig) (string, error) {
	if config.cursor == "" {
		return "", nil // No cursor, so we must fetch.
	}

	resp, err := doRequest(ctx, c, http.MethodHead, reqURL, config)
	if err != nil {
		// If HEAD fails, proceed to GET.
		return "", nil
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return resp.Header.Get("ETag"), nil
	}

	// If non-200, proceed to GET.
	return "", nil
}

func doRequest(ctx context.Context, c *http.Client, method, url string, config rfConfig) (*http.Response, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	r.Header.Set(rfTokenHeader, config.apiKey)
	r.Header.Set("Accept", "application/json")

	resp, err := c.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func processBody(body io.Reader, callback func(event map[string]any)) error {
	// The response is CSV.
	csvReader := csv.NewReader(body)
	csvReader.Comment = '#'
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		event := make(map[string]any, len(header))
		for i, value := range record {
			if i < len(header) {
				event[header[i]] = value
			}
		}

		// The original CEL encodes the whole object as a JSON string under a 'message' key.
		jsonMessage, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event to JSON: %w", err)
		}
		callback(map[string]any{"message": string(jsonMessage)})
	}

	return nil
}
