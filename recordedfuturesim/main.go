// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"compress/gzip"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//go:embed testdata/domain.csv
var domainData []byte

//go:embed testdata/hash.csv
var hashData []byte

//go:embed testdata/ip.csv
var ipData []byte

//go:embed testdata/url.csv
var urlData []byte

var addr string // Listen address

func init() {
	flag.StringVar(&addr, "http", "localhost:9903", "listen address")
}

func main() {
	flag.Parse()

	r := mux.NewRouter()
	r.NewRoute().Path("/v2/domain/risklist").Handler(newAPIKeyHandler(newRiskListHandler("domain", domainData)))
	r.NewRoute().Path("/v2/hash/risklist").Handler(newAPIKeyHandler(newRiskListHandler("hash", hashData)))
	r.NewRoute().Path("/v2/ip/risklist").Handler(newAPIKeyHandler(newRiskListHandler("ip", ipData)))
	r.NewRoute().Path("/v2/url/risklist").Handler(newAPIKeyHandler(newRiskListHandler("url", urlData)))
	h := handlers.CombinedLoggingHandler(os.Stderr, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		log.Fatal(http.ListenAndServe(addr, h))
	}()

	log.Printf("Listening on http://%s", addr)
	<-done
}

type apiKeyHandler struct {
	next http.Handler
}

func newAPIKeyHandler(next http.Handler) apiKeyHandler { return apiKeyHandler{next: next} }

func (h apiKeyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("X-RFToken") == "" {
		http.Error(w, "Missing X-RFToken", http.StatusUnauthorized)
		return
	}
	h.next.ServeHTTP(w, req)
}

type riskListHandler struct {
	entity          string
	responseDataCSV []byte
}

func newRiskListHandler(entity string, responseDataGzip []byte) *riskListHandler {
	return &riskListHandler{entity: entity, responseDataCSV: responseDataGzip}
}

func (h *riskListHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Query params
	var (
		format     = req.URL.Query().Get("format")
		returnGzip bool
	)

	switch format {
	case "csv/splunk":
	default:
		http.Error(w, "invalid format", http.StatusBadRequest)
		return
	}

	if v := req.URL.Query().Get("gzip"); v != "" {
		var err error
		if returnGzip, err = strconv.ParseBool(v); err != nil {
			http.Error(w, "gzip param must be a boolean", http.StatusBadRequest)
			return
		}
	}

	if returnGzip {
		w.Header().Set("Content-Type", "application/gzip;charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=rf_%s_threatfeed_default_csv.csv.gz", h.entity))

		gzipWriter := gzip.NewWriter(w)
		if _, err := gzipWriter.Write(h.responseDataCSV); err != nil {
			log.Printf("WARN: failed writing response to client: %v", err)
		}
		if err := gzipWriter.Close(); err != nil {
			log.Printf("WARN: failed writing response to client: %v", err)
		}
	} else {
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=rf_%s_threatfeed_default_csv.csv", h.entity))
		if _, err := w.Write(h.responseDataCSV); err != nil {
			log.Printf("WARN: failed writing response to client: %v", err)
		}
	}
}
