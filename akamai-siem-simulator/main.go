// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This simulates the Akamai SIEM API. It provides realistic validation of
// request signatures. It emulates the time and offset based queries by
// simulating a world in which a new event happens every 5 minutes. The
// offset cursor is simply the unix time in sec of the last returned event.
//
// References
//
//	https://techdocs.akamai.com/siem-integration/reference/get-configid
//	https://techdocs.akamai.com/developer/docs/authenticate-with-edgegrid
//	https://github.com/akamai/AkamaiOPEN-edgegrid-golang/blob/d417bd104d59eb9bf668da20c35f9bf899b65f90/pkg/edgegrid/signer.go#L44-L54
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// newDataInterval controls how often new events happen in the simulator.
const newDataInterval = 5 * time.Minute

var (
	addr         string // Listen address
	accessToken  string
	clientSecret string
	clientToken  string
)

func init() {
	flag.StringVar(&addr, "http", "localhost:9903", "listen address")
	flag.StringVar(&accessToken, "access-token", "", "access token (required)")
	flag.StringVar(&clientSecret, "client-secret", "", "client secret (required)")
	flag.StringVar(&clientToken, "client-token", "", "client token (required)")
}

func main() {
	flag.Parse()

	if accessToken == "" || clientSecret == "" || clientToken == "" {
		flag.Usage()
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.NewRoute().Path("/siem/v1/configs/{configId}").Handler(
		newSignatureHandler(accessToken, clientSecret, clientToken, newSecurityEventsHandler()))
	h := handlers.CombinedLoggingHandler(os.Stderr, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		log.Fatal(http.ListenAndServe(addr, h))
	}()

	log.Printf("Listening on http://%s/siem/v1/configs/{configId}", addr)
	<-done
}

type signatureHandler struct {
	verifier *signatureChecker
	next     http.Handler
}

func newSignatureHandler(accessToken, clientSecret, clientToken string, next http.Handler) signatureHandler {
	return signatureHandler{
		verifier: &signatureChecker{
			AccessToken:  accessToken,
			ClientSecret: clientSecret,
			ClientToken:  clientToken,
		},
		next: next,
	}
}

func (h signatureHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := h.verifier.Verify(req); err != nil {
		log.Printf("ERROR %s", err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	h.next.ServeHTTP(w, req)
}

type signatureEventsHandler struct{}

func newSecurityEventsHandler() *signatureEventsHandler {
	return &signatureEventsHandler{}
}

// https://techdocs.akamai.com/siem-integration/reference/get-configid
func (*signatureEventsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Query params
	var (
		offset string
		limit  = 10000
		from   time.Time
		to     time.Time
	)

	var err error
	for query, values := range req.URL.Query() {
		if len(values) == 0 {
			continue
		}
		val := values[0]

		switch query {
		case "offset":
			offset = val
		case "limit":
			limit, err = strconv.Atoi(val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
		case "from":
			v, err := strconv.Atoi(val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
			from = time.Unix(int64(v), 0)
		case "to":
			v, err := strconv.Atoi(val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
			to = time.Unix(int64(v), 0)
		default:
			http.Error(w, "invalid query param "+query, http.StatusBadRequest)
			return
		}
	}

	// API actually returns application/x-ndjson.
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Server", "Jetty(9.4.43.v20210629)")
	w.Header().Set("Vary", "Accept-Encoding, User-Agent")

	if offset == "" && to.IsZero() {
		http.Error(w, "need either and offset or a 'to'", http.StatusBadRequest)
		return
	}

	var events []any
	var newOffset string
	var limited bool

	if offset != "" {
		events, newOffset, limited, err = generateSampleDataFromOffset(offset, limit)
	} else {
		events, newOffset, limited, err = generateSampleData(from, to, limit)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, e := range events {
		if err = writeJSON(e, w); err != nil {
			log.Printf("WARN: failed writing response to client: %v", err)
		}
	}
	oc := offsetContext{
		Total:  len(events),
		Offset: newOffset,
	}
	if limited {
		oc.Limit = limit
	}
	if err = writeJSON(oc, w); err != nil {
		log.Printf("WARN: failed writing response to client: %v", err)
	}
	log.Printf("DEBUG Return offset context of %#v", oc)
}

func writeJSON(v any, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// https://techdocs.akamai.com/siem-integration/reference/offsetcontext
type offsetContext struct {
	Total  int    `json:"total"`           // The number of security events included in the response.
	Offset string `json:"offset"`          // Identifies the last processed security event in a response.
	Limit  int    `json:"limit,omitempty"` // Appears if the size limit was reached during data fetch.
}

func generateSampleData(from, to time.Time, limit int) (events []any, offset string, limited bool, err error) {
	if !from.Before(to) {
		return nil, "", false, fmt.Errorf("'from' must be earlier than 'to'")
	}

	t := from
	offset = strconv.FormatInt(t.Unix(), 10)
	for {
		t = t.Add(newDataInterval)
		if !t.Before(to) {
			break
		}
		if limit > 0 && len(events) >= limit {
			limited = true
			break
		}
		events = append(events, newEvent(t))
		offset = strconv.FormatInt(t.Unix(), 10)
	}

	return events, offset, limited, nil
}

func generateSampleDataFromOffset(offset string, limit int) (events []any, newOffset string, limited bool, err error) {
	v, err := strconv.Atoi(offset)
	if err != nil {
		return nil, "", false, err
	}
	from := time.Unix(int64(v), 0)
	to := time.Now()
	return generateSampleData(from, to, limit)
}

//go:embed assets/event.json
var eventJSON []byte

// newEvent returns a new event where the httpMessage.start
// time is set to the given timestamp.
func newEvent(timestamp time.Time) any {
	var e map[string]any
	if err := json.Unmarshal(eventJSON, &e); err != nil {
		panic(err)
	}

	e["httpMessage"].(map[string]any)["start"] = timestamp.Unix()
	e["httpMessage"].(map[string]any)["requestId"] = strconv.FormatInt(timestamp.UnixNano(), 16)
	return e
}

// ----------------------------
// Request signature validation
// ----------------------------

const eg1AuthType = "EG1-HMAC-SHA256"

var authHeaderRegex = regexp.MustCompile(`(?m)^(?P<auth_type>[\w-]+) client_token=(?P<client_token>[^;]+);access_token=(?P<access_token>[^;]+);timestamp=(?P<timestamp>[^;]+);nonce=(?P<nonce>[^;]+);signature=(?P<signature>[^;]+)$`)

type signatureChecker struct {
	ClientToken  string
	ClientSecret string
	AccessToken  string
}

func (c *signatureChecker) Verify(req *http.Request) error {
	auth := req.Header.Get("Authorization")
	if auth == "" {
		return fmt.Errorf("missing Authorization header")
	}

	match := authHeaderRegex.FindStringSubmatch(auth)
	if len(match) == 0 {
		return fmt.Errorf("invalid Authorization header format")
	}

	authType := match[authHeaderRegex.SubexpIndex("auth_type")]
	clientToken := match[authHeaderRegex.SubexpIndex("client_token")]
	accessToken := match[authHeaderRegex.SubexpIndex("access_token")]
	timestamp := match[authHeaderRegex.SubexpIndex("timestamp")]
	nonce := match[authHeaderRegex.SubexpIndex("nonce")]
	signature := match[authHeaderRegex.SubexpIndex("signature")]

	if authType != eg1AuthType {
		return fmt.Errorf("invalid auth type %q", authType)
	}

	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	msgData := []string{
		req.Method,
		scheme,
		req.Host,
		req.URL.RequestURI(),
		"", // No headers.
		"", // No body.
		fmt.Sprintf("%s client_token=%s;access_token=%s;timestamp=%s;nonce=%s;",
			authType, clientToken, accessToken, timestamp, nonce),
	}
	msg := strings.Join(msgData, "\t")

	key := createSignature(timestamp, c.ClientSecret)
	calculatedSignature := createSignature(msg, key)

	if signature != calculatedSignature {
		return errors.New("the signature does not match")
	}
	return nil
}

func createSignature(message, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
