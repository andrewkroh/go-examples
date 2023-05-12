package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// https://backstage.forgerock.com/docs/idcloud/latest/tenants/audit-debug-logs.html
// https://backstage.forgerock.com/knowledge/kb/article/a37739488

// {
//  "errors": [
//    "Start Time (2022-10-08 00:00:00 +0000 UTC) Must Precede End Time (2022-10-07 00:00:00 +0000 UTC)"
//  ]
//}

// HTTP 400
// {
//  "errors": [
//    "1.00 Days Worth Of Data Requested (Query Start Date 2022-10-07 00:00:00 +0000 UTC, Query End Date 2022-10-08 00:00:01 +0000 UTC). Please Limit The Scope Of Your Query To Within A Day: Error: Cannot Request More Than One Days Worth Of Logs"
//  ]
//}

// HTTP 400
// {
//  "errors": [
//    "Error: Source Not Specified"
//  ]
//}

// HTTP 400
// {
//  "errors": [
//    "Error: BeginTime Must Be In RFC3339 Format"
//  ]
//}

// HTTP error messages
const (
	sourceNotSpecified          = "Error: Source Not Specified"
	moreThanADaysWorth          = "Please Limit The Scope Of Your Query To Within A Day: Error: Cannot Request More Than One Days Worth Of Logs"
	startTimeMustPreceedEndTime = "Start Time Must Precede End Time"
	badBeginTime                = "Error: BeginTime Must Be In RFC3339 Format"
	badEndTime                  = "Error: EndTime Must Be In RFC3339 Format"
	badPageSize                 = "Error Parsing Query Parameter _pageSize: Must Be Integer"
	badPageResultsCookie        = "Error Parsing Query Parameter _pagedResultsCookie: Invalid Format"
)

const (
	maxRequestDuration = 24 * time.Hour
	defaultBeginTime   = 15 * time.Minute // Unknown value, so this is a guess.
	contentType        = "Content-Type"
	applicationJSON    = "application/json; charset=utf-8"
)

var (
	addr     string // Listen address
	dataFile string // JSON file containing event data.
)

func init() {
	flag.StringVar(&addr, "http", "localhost:9888", "listen address")
	flag.StringVar(&dataFile, "data", "", "JSON data file containing events to serve (required)")
}

func main() {
	flag.Parse()

	if dataFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	logs, err := loadLogs(dataFile)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.NewRoute().Path("/monitoring/logs").Handler(newAPIKeyHandler(newMonitoringLogsHandler(logs)))
	h := handlers.CombinedLoggingHandler(os.Stderr, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		log.Fatal(http.ListenAndServe(addr, h))
	}()

	log.Printf("Listening on http://%s", addr)
	<-done
}

type LogsResponse struct {
	Result                  []Log   `json:"result"`
	ResultCount             int     `json:"resultCount"`
	PagedResultsCookie      *string `json:"pagedResultsCookie"`
	RemainingPagedResults   int     `json:"remainingPagedResults"`
	TotalPagedResults       int     `json:"totalPagedResults"`
	TotalPagedResultsPolicy string  `json:"totalPagedResultsPolicy"`
}

type Log struct {
	Payload   any       `json:"payload"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

type apiKeyHandler struct {
	next http.Handler
}

func newAPIKeyHandler(next http.Handler) apiKeyHandler { return apiKeyHandler{next: next} }

func (h apiKeyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("x-api-key") == "" || req.Header.Get("x-api-secret") == "" {
		http.Error(w, "Missing x-api-key or x-api-secret", http.StatusUnauthorized)
		return
	}
	h.next.ServeHTTP(w, req)
}

type monitoringLogsHandler struct {
	logs []Log
}

func newMonitoringLogsHandler(logs []Log) *monitoringLogsHandler {
	return &monitoringLogsHandler{logs: logs}
}

func (h *monitoringLogsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	now := time.Now()

	// Query params
	var (
		sources   []string
		beginTime = now.Add(-1 * defaultBeginTime)
		endTime   = now
		pageSize  = 1000
		page      = 0
	)

	if v := req.URL.Query().Get("source"); v != "" {
		if sources = strings.FieldsFunc(v, isComma); len(sources) == 0 {
			http.Error(w, sourceNotSpecified, http.StatusBadRequest)
			return
		}
	}

	if v := req.URL.Query().Get("beginTime"); v != "" {
		var err error
		if beginTime, err = time.Parse(time.RFC3339Nano, v); err != nil {
			http.Error(w, badBeginTime, http.StatusBadRequest)
			return
		}
	}

	if v := req.URL.Query().Get("endTime"); v != "" {
		var err error
		if endTime, err = time.Parse(time.RFC3339Nano, v); err != nil {
			http.Error(w, badEndTime, http.StatusBadRequest)
			return
		}
	}

	if v := req.URL.Query().Get("_pageSize"); v != "" {
		size, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, badPageSize, http.StatusBadRequest)
			return
		}
		if size > 0 {
			pageSize = size
		}
	}

	if v := req.URL.Query().Get("_pagedResultsCookie"); v != "" {
		c, err := decodeCookie(v)
		if err != nil {
			http.Error(w, badPageResultsCookie, http.StatusBadRequest)
			return
		}

		sources = c.SourceTypes
		beginTime = time.Time(c.BeginTime)
		endTime = time.Time(c.EndTime)
		pageSize = c.PageSize
		page = c.NextPage
	}

	if len(sources) == 0 {
		http.Error(w, sourceNotSpecified, http.StatusBadRequest)
		return
	}

	if endTime.Sub(beginTime) > maxRequestDuration {
		http.Error(w, moreThanADaysWorth, http.StatusBadRequest)
		return
	}

	if endTime.Before(beginTime) {
		http.Error(w, startTimeMustPreceedEndTime, http.StatusBadRequest)
		return
	}

	logs, more, err := getResults(h.logs, sources, page, pageSize, beginTime, endTime)
	if err != nil {
		http.Error(w, "Internal server failure", http.StatusInternalServerError)
		return
	}

	response := LogsResponse{
		Result:                  logs,
		ResultCount:             len(logs),
		PagedResultsCookie:      nil,
		RemainingPagedResults:   -1,
		TotalPagedResults:       -1,
		TotalPagedResultsPolicy: "NONE",
	}

	if more {
		c := newCookie(sources, pageSize, page+1, beginTime, endTime)
		base64Cookie := c.Encode()
		response.PagedResultsCookie = &base64Cookie
	}

	serveJSON(w, response, true)
}

func serveJSON(w http.ResponseWriter, value any, pretty bool) {
	w.Header().Set(contentType, applicationJSON)
	out := io.MultiWriter(w, os.Stderr)
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if pretty {
		enc.SetIndent("", "  ")
	}
	_ = enc.Encode(value)
}

func getResults(logs []Log, sourceTypes []string, page, pageSize int, beginTime, endTime time.Time) (pageData []Log, more bool, err error) {
	var startIdx, endIdx int

	for i, log := range logs {
		if log.Timestamp.After(beginTime) {
			startIdx = i
			break
		}
	}

	endIdx = startIdx
	for _, log := range logs[startIdx:] {
		if log.Timestamp.After(endTime) {
			break
		}
		endIdx++
	}

	// NOTE: This is not at all optimized.
	var matches []Log
	for _, log := range logs[startIdx:endIdx] {
		for _, wantedSource := range sourceTypes {
			if log.Source == wantedSource {
				matches = append(matches, log)
				break
			}
		}
	}

	recordIndex := page * pageSize
	if recordIndex < len(matches) {
		pageData = matches[recordIndex:min(recordIndex+pageSize, len(matches))]
	}
	if len(matches) > recordIndex+pageSize {
		more = true
	}

	if pageData == nil {
		// Always respond with empty slice instead of nil.
		pageData = []Log{}
	}

	return pageData, more, nil
}

func isComma(r rune) bool {
	return r == ','
}

type Cookie struct {
	SourceTypes []string `json:"sources"`
	PageSize    int      `json:"page_size"`
	BeginTime   Time     `json:"begin_time"`
	EndTime     Time     `json:"end_time"`
	NextPage    int      `json:"next_page"`
}

func (c Cookie) Encode() string {
	data, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(data)
}

func newCookie(sourceTypes []string, nextPage, pageSize int, beginTime, endTime time.Time) *Cookie {
	return &Cookie{
		SourceTypes: sourceTypes,
		PageSize:    pageSize,
		BeginTime:   Time(beginTime),
		EndTime:     Time(endTime),
		NextPage:    nextPage,
	}
}

func decodeCookie(in string) (*Cookie, error) {
	data, err := base64.RawStdEncoding.DecodeString(in)
	if err != nil {
		return nil, err
	}
	c := &Cookie{}
	if err = json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}

type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	v := time.Time(t).UTC().Format(time.RFC3339Nano)
	return json.Marshal(v)
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	s, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}

	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	*t = Time(v.UTC())
	return nil
}

func loadLogs(path string) ([]Log, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	dec.UseNumber()

	var logs []Log
	if err = dec.Decode(&logs); err != nil {
		return nil, err
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Nanosecond() < logs[j].Timestamp.Nanosecond()
	})

	return logs, nil
}

func min[T int](a, b T) T {
	if a <= b {
		return a
	}
	return b
}
