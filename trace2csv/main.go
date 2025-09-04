package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), `
Usage: trace2csv [OPTIONS]

Converts request trace logs from Elastic Agent's CEL and HTTP JSON inputs
to CSV format. Reads NDJSON (newline-delimited JSON) from stdin and writes
CSV to stdout.

The tool processes trace logs containing HTTP request/response data and
URL information, adding calculated fields like delta_sec between consecutive
requests. Certain sensitive headers are automatically redacted in the output
and replaced with sha256:<first 8-bytes of base64 encoded sha256>, but in
general you should not rely on this tool for privacy protection.

Example:
  cat trace.ndjson | trace2csv > output.csv

Options:
`[1:])
		flag.PrintDefaults()
	}
	flag.Parse()

	traces, err := readTraces()
	if err != nil {
		log.Fatal("ERROR:", err)
	}

	if err := writeCSV(traces); err != nil {
		log.Fatal("ERROR writing CSV:", err)
	}
}

// BeatTime is a time.Time that handles additional time formats resulting
// from different versions of elastic/beats.
type BeatTime time.Time

// TraceLog represents a trace log entry with HTTP and URL metadata.
type TraceLog struct {
	Timestamp                 BeatTime            `json:"@timestamp,omitempty"`
	HTTPRequestBodyBytes      float64             `json:"http.request.body.bytes,omitempty"`
	HTTPRequestBodyContent    string              `json:"http.request.body.content,omitempty"`
	HTTPRequestBodyTruncated  bool                `json:"http.request.body.truncated,omitempty"`
	HTTPRequestHeader         map[string][]string `json:"http.request.header,omitempty"`
	HTTPRequestMethod         string              `json:"http.request.method,omitempty"`
	HTTPRequestMimeType       string              `json:"http.request.mime_type,omitempty"`
	HTTPResponseBodyBytes     float64             `json:"http.response.body.bytes,omitempty"`
	HTTPResponseBodyContent   string              `json:"http.response.body.content,omitempty"` // Not used in output.
	HTTPResponseBodyTruncated bool                `json:"http.response.body.truncated,omitempty"`
	HTTPResponseHeader        map[string][]string `json:"http.response.header,omitempty"`
	HTTPResponseMimeType      string              `json:"http.response.mime_type,omitempty"`
	HTTPResponseStatusCode    float64             `json:"http.response.status_code,omitempty"`
	LogLevel                  string              `json:"log.level,omitempty"`
	Message                   string              `json:"message,omitempty"`
	TransactionID             string              `json:"transaction.id,omitempty"`
	URLDomain                 string              `json:"url.domain,omitempty"`
	URLOriginal               string              `json:"url.original,omitempty"`
	URLPath                   string              `json:"url.path,omitempty"`
	URLPort                   string              `json:"url.port,omitempty"`
	URLQuery                  string              `json:"url.query,omitempty"`
	URLScheme                 string              `json:"url.scheme,omitempty"`
}

// IsZero reports whether bt represents the zero time instant.
func (bt *BeatTime) IsZero() bool {
	return time.Time(*bt).IsZero()
}

// Time returns the underlying time.Time.
func (bt *BeatTime) Time() time.Time {
	return time.Time(*bt)
}

// UTC returns bt with the location set to UTC.
func (bt *BeatTime) UTC() time.Time {
	return time.Time(*bt).UTC()
}

// Unix returns the Unix time in seconds.
func (bt *BeatTime) Unix() int64 {
	return time.Time(*bt).Unix()
}

// UnmarshalJSON implements json.Unmarshaler to handle multiple time formats.
func (bt *BeatTime) UnmarshalJSON(data []byte) error {
	var t time.Time
	if err := t.UnmarshalJSON(data); err == nil {
		*bt = BeatTime(t)
		return nil
	}

	// If that fails, try our custom format.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Try more variations.
	formats := []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05Z0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			*bt = BeatTime(t)
			return nil
		}
	}

	return fmt.Errorf("unable to parse time %q", s)
}

// readTraces reads TraceLog entries from stdin.
func readTraces() ([]*TraceLog, error) {
	dec := json.NewDecoder(os.Stdin)

	var traces []*TraceLog
	for dec.More() {
		var l TraceLog
		if err := dec.Decode(&l); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		traces = append(traces, &l)
	}

	return traces, nil
}

// toMap converts a TraceLog to a map for CSV export.
func toMap(t *TraceLog) map[string]string {
	m := make(map[string]string)

	m["@timestamp"] = t.Timestamp.UTC().Format(time.RFC3339Nano)
	m["unix_sec"] = strconv.FormatInt(t.Timestamp.Unix(), 10)

	if t.HTTPRequestMethod != "" {
		m["http.request.method"] = t.HTTPRequestMethod
	}
	if t.HTTPRequestMimeType != "" {
		m["http.request.mime_type"] = t.HTTPRequestMimeType
	}
	if t.HTTPResponseMimeType != "" {
		m["http.response.mime_type"] = t.HTTPResponseMimeType
	}
	if t.LogLevel != "" {
		m["log.level"] = t.LogLevel
	}
	if t.Message != "" {
		m["message"] = t.Message
	}
	if t.TransactionID != "" {
		m["transaction.id"] = t.TransactionID
	}
	if t.URLDomain != "" {
		m["url.domain"] = t.URLDomain
	}
	if t.URLOriginal != "" {
		m["url.original"] = t.URLOriginal
	}
	if t.URLPath != "" {
		m["url.path"] = t.URLPath
	}
	if t.URLPort != "" {
		m["url.port"] = t.URLPort
	}
	if t.URLQuery != "" {
		m["url.query"] = t.URLQuery
	}
	if t.URLScheme != "" {
		m["url.scheme"] = t.URLScheme
	}

	// Add numeric fields (only if non-zero).
	if t.HTTPRequestBodyBytes != 0 {
		m["http.request.body.bytes"] = strconv.FormatFloat(t.HTTPRequestBodyBytes, 'f', -1, 64)
	}
	if t.HTTPRequestBodyContent != "" {
		m["http.request.body.content"] = t.HTTPRequestBodyContent
	}
	if t.HTTPResponseBodyBytes != 0 {
		m["http.response.body.bytes"] = strconv.FormatFloat(t.HTTPResponseBodyBytes, 'f', -1, 64)
	}
	if t.HTTPResponseStatusCode != 0 {
		m["http.response.status_code"] = strconv.FormatFloat(t.HTTPResponseStatusCode, 'f', -1, 64)
	}

	// Add boolean fields (only if true).
	if t.HTTPRequestBodyTruncated {
		m["http.request.body.truncated"] = "true"
	}
	if t.HTTPResponseBodyTruncated {
		m["http.response.body.truncated"] = "true"
	}

	// Add HTTP request headers.
	for name, values := range t.HTTPRequestHeader {
		key := fmt.Sprintf("http.request.header.%s", name)
		m[key] = redact(name, values)
	}

	// Add HTTP response headers.
	for name, values := range t.HTTPResponseHeader {
		key := fmt.Sprintf("http.response.header.%s", name)
		m[key] = redact(name, values)
	}

	// Add URL query parameters.
	query, err := url.ParseQuery(t.URLQuery)
	if err == nil {
		for name, values := range query {
			key := fmt.Sprintf("url.query.%s", name)
			m[key] = strings.Join(values, "|")
		}
	}

	return m
}

// writeCSV writes trace logs to stdout in CSV format.
func writeCSV(traces []*TraceLog) error {
	if len(traces) == 0 {
		return nil
	}

	// Convert all traces to maps and compute delta_sec.
	traceMaps := make([]map[string]string, len(traces))
	allKeys := make(map[string]bool)
	var prevTime *time.Time

	for i, trace := range traces {
		m := toMap(trace)
		traceMaps[i] = m

		// Compute delta_sec.
		currentTime := trace.Timestamp.Time()
		if prevTime != nil {
			delta := currentTime.Sub(*prevTime)
			m["delta_sec"] = strconv.FormatInt(int64(delta.Seconds()), 10)
		}
		prevTime = &currentTime

		for k := range m {
			allKeys[k] = true
		}
	}

	// Build ordered header list.
	var headers []string

	// First add timestamps if present.
	if allKeys["@timestamp"] {
		headers = append(headers, "@timestamp")
		delete(allKeys, "@timestamp")
	}
	if allKeys["unix_sec"] {
		headers = append(headers, "unix_sec")
		delete(allKeys, "unix_sec")
	}
	if allKeys["delta_sec"] {
		headers = append(headers, "delta_sec")
		delete(allKeys, "delta_sec")
	}

	// Then add message if present.
	if allKeys["message"] {
		headers = append(headers, "message")
		delete(allKeys, "message")
	}

	// Sort remaining keys and append.
	var remainingKeys []string
	for k := range allKeys {
		remainingKeys = append(remainingKeys, k)
	}
	sort.Strings(remainingKeys)
	headers = append(headers, remainingKeys...)

	// Write CSV.
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header row.
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write data rows.
	for _, traceMap := range traceMaps {
		row := make([]string, len(headers))
		for i, header := range headers {
			row[i] = traceMap[header]
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func isSecret(key string) bool {
	k := strings.ToLower(key)
	switch k {
	case "authorization", "cookie", "set-cookie", "proxy-authorization", "x-redlock-auth":
		return true
	}

	search := []string{
		"apikey", "authkey", "token",
	}
	k = strings.ReplaceAll(k, "-", "")
	for _, s := range search {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

func redact(key string, values []string) string {
	if !isSecret(key) {
		return strings.Join(values, "|")
	}

	var redacted []string
	for _, value := range values {
		h := sha256.New().Sum([]byte(value))
		sha := base64.RawStdEncoding.EncodeToString(h)
		sha = "sha256:" + sha[:8]
		redacted = append(redacted, sha)
	}
	return strings.Join(redacted, "|")
}
