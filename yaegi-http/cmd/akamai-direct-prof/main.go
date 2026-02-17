// Command akamai-direct-prof runs Akamai SIEM polling natively and records pprof data.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"runtime"
	"runtime/pprof"
)

const (
	akamaiAuthType = "EG1-HMAC-SHA256"

	defaultConfigID        = "1"
	defaultLimit           = 1000
	defaultPollTimeout     = 8 * time.Second
	defaultInitialInterval = 1 * time.Hour
	defaultToLag           = 5 * time.Second
	defaultOffsetTTL       = 2 * time.Minute
)

var (
	programPath        string
	runDuration        time.Duration
	cpuProfileDuration time.Duration
	outDir             string

	errOffsetInvalid = fmt.Errorf("offset is invalid or expired")
	spaceRE          = regexp.MustCompile(`\s{2,}`)
	nonceSeq         uint64
)

type akamaiConfig struct {
	baseURL      string
	configID     string
	clientToken  string
	clientSecret string
	accessToken  string

	limit           int
	pollTimeout     time.Duration
	pollInterval    time.Duration
	maxRequests     int
	offsetTTL       time.Duration
	initialInterval time.Duration
	toLag           time.Duration

	initialCursor string
	initialFrom   int64
	headersToSign []string
}

type pollState struct {
	cursor        string
	lastRequestAt time.Time
	lastEventUnix int64
}

type fetchMode string

const (
	modeOffset fetchMode = "offset"
	modeTime   fetchMode = "time"
)

type offsetContext struct {
	Offset string
	Total  int
	Limit  int
}

type offsetContextLine struct {
	Offset string          `json:"offset"`
	Total  json.RawMessage `json:"total"`
	Limit  json.RawMessage `json:"limit"`
}

type eventTimeLine struct {
	Timestamp      json.RawMessage `json:"timestamp"`
	Time           json.RawMessage `json:"time"`
	EventTime      json.RawMessage `json:"eventTime"`
	EventTimestamp json.RawMessage `json:"event_timestamp"`
	HTTPMessage    struct {
		TimeStamp json.RawMessage `json:"timeStamp"`
		Start     json.RawMessage `json:"start"`
		End       json.RawMessage `json:"end"`
	} `json:"httpMessage"`
}

type fetchResult struct {
	eventCount    int
	offsetContext offsetContext
	lastEventUnix int64
}

func init() {
	flag.StringVar(&programPath, "prog", "testdata/programs/akamai.go", "kept for parity; unused by direct runner")
	flag.DurationVar(&runDuration, "duration", 2*time.Minute, "total run duration")
	flag.DurationVar(&cpuProfileDuration, "cpu-profile-duration", 30*time.Second, "CPU profile duration")
	flag.StringVar(&outDir, "out", "profiles-direct", "directory where pprof files are written")
}

func main() {
	flag.Parse()

	if runDuration <= 0 {
		log.Fatal("-duration must be greater than zero")
	}
	if cpuProfileDuration <= 0 {
		log.Fatal("-cpu-profile-duration must be greater than zero")
	}
	if runDuration <= cpuProfileDuration {
		log.Fatal("-duration must be greater than -cpu-profile-duration")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory %q: %v", outDir, err)
	}

	env := loadProgramEnv()
	if _, ok := env["POLL_TIMEOUT"]; !ok {
		env["POLL_TIMEOUT"] = defaultRunnerPollTimeout(runDuration).String()
	}

	ctx, cancel := context.WithTimeout(context.Background(), runDuration)
	defer cancel()
	ctx = context.WithValue(ctx, "env", env)

	start := time.Now()
	var eventCount int64

	profileDone := make(chan error, 1)
	go func() {
		profileDone <- captureProfiles(ctx, runDuration, cpuProfileDuration, outDir)
	}()

	err := executeAkamai(ctx, http.DefaultClient, func(map[string]any) {
		atomic.AddInt64(&eventCount, 1)
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Fatalf("program execution failed: %v", err)
	}

	if profileErr := <-profileDone; profileErr != nil {
		log.Fatalf("profiling failed: %v", profileErr)
	}

	elapsed := time.Since(start)
	count := atomic.LoadInt64(&eventCount)
	rate := float64(count) / elapsed.Seconds()
	log.Printf("done elapsed=%s events=%d eps=%.2f out=%s", elapsed.Round(time.Millisecond), count, rate, outDir)
}

func executeAkamai(ctx context.Context, c *http.Client, callback func(event map[string]any)) error {
	cfg, err := readConfig(ctx)
	if err != nil {
		return err
	}

	state := pollState{
		cursor:        cfg.initialCursor,
		lastEventUnix: cfg.initialFrom,
	}

	deadline := time.Now().Add(cfg.pollTimeout)
	var requests, emitted int

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			break
		}
		if cfg.maxRequests > 0 && requests >= cfg.maxRequests {
			break
		}

		mode := chooseMode(cfg, state)
		if mode == modeTime {
			state.cursor = ""
		}
		result, err := fetchOnce(ctx, c, cfg, mode, state, callback)
		if err != nil {
			if err == errOffsetInvalid && mode == modeOffset {
				state.cursor = ""
				continue
			}
			return err
		}

		requests++
		state.lastRequestAt = time.Now()
		if result.offsetContext.Offset != "" {
			state.cursor = result.offsetContext.Offset
		}
		if result.lastEventUnix > state.lastEventUnix {
			state.lastEventUnix = result.lastEventUnix
		}

		emitted += result.eventCount
		if result.eventCount == 0 {
			break
		}
		if result.offsetContext.Offset != "" && result.offsetContext.Total == 0 {
			break
		}
		if cfg.pollInterval > 0 {
			wait := cfg.pollInterval
			if remaining := time.Until(deadline); wait > remaining {
				wait = remaining
			}
			t := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
		}
	}

	log.Printf("akamai_siem requests=%d events=%d cursor=%q last_event_unix=%d", requests, emitted, state.cursor, state.lastEventUnix)
	return nil
}

func readConfig(ctx context.Context) (akamaiConfig, error) {
	env, ok := ctx.Value("env").(map[string]string)
	if !ok {
		return akamaiConfig{}, fmt.Errorf("failed to get config from context")
	}

	cfg := akamaiConfig{
		baseURL:      env["URL"],
		configID:     withDefault(env["CONFIG_ID"], defaultConfigID),
		clientToken:  env["CLIENT_TOKEN"],
		clientSecret: env["CLIENT_SECRET"],
		accessToken:  env["ACCESS_TOKEN"],

		limit:           parseIntWithDefault(env["LIMIT"], defaultLimit),
		pollTimeout:     parseDurationWithDefault(env["POLL_TIMEOUT"], defaultPollTimeout),
		pollInterval:    parseDurationWithDefault(env["POLL_INTERVAL"], 0),
		maxRequests:     parseIntWithDefault(env["MAX_REQUESTS"], 0),
		offsetTTL:       parseDurationWithDefault(env["OFFSET_TTL"], defaultOffsetTTL),
		initialInterval: parseDurationWithDefault(env["INITIAL_INTERVAL"], defaultInitialInterval),
		toLag:           parseDurationWithDefault(env["TO_LAG"], defaultToLag),

		initialCursor: env["CURSOR"],
		headersToSign: splitCSV(env["HEADERS_TO_SIGN"]),
	}

	if cfg.baseURL == "" || cfg.configID == "" || cfg.clientToken == "" || cfg.clientSecret == "" || cfg.accessToken == "" {
		return akamaiConfig{}, fmt.Errorf("missing required Akamai configuration values")
	}
	if cfg.limit <= 0 {
		return akamaiConfig{}, fmt.Errorf("LIMIT must be greater than zero")
	}
	if cfg.pollTimeout <= 0 {
		return akamaiConfig{}, fmt.Errorf("POLL_TIMEOUT must be greater than zero")
	}
	if cfg.initialInterval <= 0 {
		return akamaiConfig{}, fmt.Errorf("INITIAL_INTERVAL must be greater than zero")
	}
	cfg.initialFrom = parseInt64WithDefault(env["FROM"], 0)
	if cfg.initialFrom == 0 {
		cfg.initialFrom = time.Now().Add(-cfg.initialInterval).Unix()
	}

	return cfg, nil
}

func chooseMode(cfg akamaiConfig, state pollState) fetchMode {
	if state.cursor == "" {
		return modeTime
	}
	if state.lastRequestAt.IsZero() {
		return modeOffset
	}
	if time.Since(state.lastRequestAt) > cfg.offsetTTL {
		return modeTime
	}
	return modeOffset
}

func fetchOnce(ctx context.Context, c *http.Client, cfg akamaiConfig, mode fetchMode, state pollState, callback func(map[string]any)) (fetchResult, error) {
	u, err := buildURL(cfg, mode, state)
	if err != nil {
		return fetchResult{}, err
	}

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return fetchResult{}, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		signRequest(req, cfg)

		resp, err := c.Do(req)
		if err != nil {
			return fetchResult{}, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			if shouldRetryNonceReplay(resp.StatusCode, body) && attempt == 0 {
				continue
			}
			if mode == modeOffset && looksLikeOffsetError(resp.StatusCode, body) {
				return fetchResult{}, errOffsetInvalid
			}
			return fetchResult{}, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		cursor, lastEventUnix, eventCount, err := parseResponse(ctx, resp.Body, callback)
		resp.Body.Close()
		if err != nil {
			return fetchResult{}, err
		}
		return fetchResult{
			eventCount:    eventCount,
			offsetContext: cursor,
			lastEventUnix: lastEventUnix,
		}, nil
	}
	return fetchResult{}, fmt.Errorf("request attempts exhausted")
}

func buildURL(cfg akamaiConfig, mode fetchMode, state pollState) (string, error) {
	base := strings.TrimSuffix(cfg.baseURL, "/")
	u, err := url.Parse(base + "/configs/" + url.PathEscape(cfg.configID))
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("limit", strconv.Itoa(cfg.limit))
	switch mode {
	case modeOffset:
		q.Set("offset", state.cursor)
	case modeTime:
		from := state.lastEventUnix
		if from == 0 {
			from = time.Now().Add(-cfg.initialInterval).Unix()
		}
		q.Set("from", strconv.FormatInt(from, 10))
		if cfg.toLag > 0 {
			to := time.Now().Add(-cfg.toLag).Unix()
			if to > from {
				q.Set("to", strconv.FormatInt(to, 10))
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseResponse(ctx context.Context, r io.Reader, callback func(map[string]any)) (offsetContext, int64, int, error) {
	s := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 16*1024*1024)

	var (
		cursor        offsetContext
		lastEventUnix int64
		eventCount    int
	)

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		if c, ok, err := decodeOffsetContextLine(line); err != nil {
			if ctx.Err() != nil && strings.Contains(err.Error(), "unexpected end of JSON input") {
				break
			}
			return offsetContext{}, 0, 0, fmt.Errorf("failed to decode response line as JSON: %w", err)
		} else if ok {
			cursor = c
			continue
		}

		ts, err := extractEventUnixLine(line)
		if err != nil {
			if ctx.Err() != nil && strings.Contains(err.Error(), "unexpected end of JSON input") {
				break
			}
			return offsetContext{}, 0, 0, fmt.Errorf("failed to decode response line as JSON: %w", err)
		}
		if ts > lastEventUnix {
			lastEventUnix = ts
		}
		callback(map[string]any{"message": line})
		eventCount++
	}
	if err := s.Err(); err != nil {
		if ctx.Err() != nil {
			return cursor, lastEventUnix, eventCount, nil
		}
		return offsetContext{}, 0, 0, fmt.Errorf("failed reading response: %w", err)
	}

	return cursor, lastEventUnix, eventCount, nil
}

func decodeOffsetContextLine(line string) (offsetContext, bool, error) {
	if !strings.Contains(line, "\"offset\"") {
		return offsetContext{}, false, nil
	}

	var item offsetContextLine
	if err := json.Unmarshal([]byte(line), &item); err != nil {
		return offsetContext{}, false, err
	}
	if item.Offset == "" {
		return offsetContext{}, false, nil
	}

	hasTotal := len(item.Total) > 0
	hasLimit := len(item.Limit) > 0
	if !hasTotal && !hasLimit {
		return offsetContext{}, false, nil
	}

	total, err := intFromRaw(item.Total)
	if err != nil {
		return offsetContext{}, false, err
	}
	limit, err := intFromRaw(item.Limit)
	if err != nil {
		return offsetContext{}, false, err
	}

	return offsetContext{
		Offset: item.Offset,
		Total:  total,
		Limit:  limit,
	}, true, nil
}

func extractEventUnixLine(line string) (int64, error) {
	var event eventTimeLine
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return 0, err
	}

	values := []json.RawMessage{
		event.Timestamp,
		event.Time,
		event.EventTime,
		event.EventTimestamp,
		event.HTTPMessage.TimeStamp,
		event.HTTPMessage.Start,
		event.HTTPMessage.End,
	}
	var latest int64
	for _, raw := range values {
		if ts := parseUnixRaw(raw); ts > latest {
			latest = ts
		}
	}
	return latest, nil
}

func parseUnixRaw(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	v := strings.TrimSpace(string(raw))
	if v == "" || v == "null" {
		return 0
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		s, err := strconv.Unquote(v)
		if err != nil {
			return 0
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if ts, err := time.Parse(time.RFC3339, s); err == nil {
			return ts.Unix()
		}
		return 0
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return int64(f)
	}
	return 0
}

func looksLikeOffsetError(statusCode int, body []byte) bool {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone, http.StatusUnprocessableEntity:
	default:
		return false
	}

	s := strings.ToLower(string(body))
	return strings.Contains(s, "offset") || strings.Contains(s, "cursor") || strings.Contains(s, "token")
}

func shouldRetryNonceReplay(statusCode int, body []byte) bool {
	if statusCode != http.StatusUnauthorized {
		return false
	}
	s := strings.ToLower(string(body))
	return strings.Contains(s, "nonce") && strings.Contains(s, "already used")
}

func signRequest(req *http.Request, cfg akamaiConfig) {
	timestamp := edgegridTimestamp(time.Now())
	nonce := randomNonce()
	authPrefix := fmt.Sprintf(
		"%s client_token=%s;access_token=%s;timestamp=%s;nonce=%s;",
		akamaiAuthType,
		cfg.clientToken,
		cfg.accessToken,
		timestamp,
		nonce,
	)

	msgPath := req.URL.EscapedPath()
	if req.URL.RawQuery != "" {
		msgPath = msgPath + "?" + req.URL.RawQuery
	}

	message := strings.Join([]string{
		req.Method,
		req.URL.Scheme,
		req.URL.Host,
		msgPath,
		canonicalizeHeaders(req.Header, cfg.headersToSign),
		createContentHash(req, 131072),
		authPrefix,
	}, "\t")

	signingKey := createSignature(timestamp, cfg.clientSecret)
	signature := createSignature(message, signingKey)
	req.Header.Set("Authorization", authPrefix+"signature="+signature)
}

func edgegridTimestamp(t time.Time) string {
	return t.UTC().Format("20060102T15:04:05-0700")
}

func randomNonce() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		n := atomic.AddUint64(&nonceSeq, 1)
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), n)
	}
	n := atomic.AddUint64(&nonceSeq, 1)
	b[16] = byte(n >> 56)
	b[17] = byte(n >> 48)
	b[18] = byte(n >> 40)
	b[19] = byte(n >> 32)
	b[20] = byte(n >> 24)
	b[21] = byte(n >> 16)
	b[22] = byte(n >> 8)
	b[23] = byte(n)
	return hex.EncodeToString(b)
}

func createSignature(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func canonicalizeHeaders(requestHeaders http.Header, headersToSign []string) string {
	if len(headersToSign) == 0 {
		return ""
	}

	want := make(map[string]struct{}, len(headersToSign))
	for _, h := range headersToSign {
		if h == "" {
			continue
		}
		want[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}

	var keys []string
	for k := range requestHeaders {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var out []string
	for _, k := range keys {
		if _, ok := want[strings.ToLower(k)]; !ok {
			continue
		}
		v := strings.TrimSpace(requestHeaders.Get(k))
		v = strings.ToLower(spaceRE.ReplaceAllString(v, " "))
		out = append(out, strings.ToLower(k)+":"+v)
	}
	return strings.Join(out, "\t")
}

func createContentHash(req *http.Request, maxBody int) string {
	if req.Method != http.MethodPost || req.Body == nil {
		return ""
	}
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return ""
	}
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if len(bodyBytes) > maxBody {
		bodyBytes = bodyBytes[:maxBody]
	}
	sum := sha256.Sum256(bodyBytes)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func intFromRaw(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, nil
	}
	v := strings.TrimSpace(string(raw))
	if v == "" || v == "null" {
		return 0, nil
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		s, err := strconv.Unquote(v)
		if err != nil {
			return 0, err
		}
		i, err := strconv.Atoi(s)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func withDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func parseIntWithDefault(v string, d int) int {
	if v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}

func parseInt64WithDefault(v string, d int64) int64 {
	if v == "" {
		return d
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return d
	}
	return i
}

func parseDurationWithDefault(v string, d time.Duration) time.Duration {
	if v == "" {
		return d
	}
	dur, err := time.ParseDuration(v)
	if err != nil {
		return d
	}
	return dur
}

func loadProgramEnv() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		after, found := strings.CutPrefix(kv, "YAEGI_HTTP_")
		if !found {
			continue
		}
		parts := strings.SplitN(after, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func defaultRunnerPollTimeout(runDuration time.Duration) time.Duration {
	if runDuration <= 2*time.Second {
		return runDuration
	}
	if runDuration <= 20*time.Second {
		return runDuration - 1*time.Second
	}
	return runDuration - 5*time.Second
}

func captureProfiles(ctx context.Context, total, cpuDur time.Duration, out string) error {
	midpoint := total / 2
	timer := time.NewTimer(midpoint)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return writeHeapProfile(filepath.Join(out, "heap_end.pprof"))
	case <-timer.C:
	}

	if err := writeHeapProfile(filepath.Join(out, "heap_mid.pprof")); err != nil {
		return err
	}
	if err := writeCPUProfile(filepath.Join(out, "cpu_mid.pprof"), cpuDur); err != nil {
		return err
	}
	return writeHeapProfile(filepath.Join(out, "heap_end.pprof"))
}

func writeCPUProfile(path string, d time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile %q: %w", path, err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	time.Sleep(d)
	pprof.StopCPUProfile()
	return nil
}

func writeHeapProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create heap profile %q: %w", path, err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}
	return nil
}
