// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"compress/gzip"
	"encoding/csv"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskListHandler(t *testing.T) {
	for _, handler := range []*riskListHandler{
		newRiskListHandler("domain", domainData),
		newRiskListHandler("hash", hashData),
		newRiskListHandler("ip", ipData),
		newRiskListHandler("url", urlData),
	} {
		h := handler
		t.Run(h.entity, func(t *testing.T) {
			t.Run("with_gzip", func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/v2/hash/risklist?format=csv/splunk&gzip=true&list=default", nil)

				h.ServeHTTP(w, r)
				resp := w.Result()
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Contains(t, resp.Header.Get("Content-Type"), "application/gzip")

				reader, err := gzip.NewReader(resp.Body)
				require.NoError(t, err)

				readAllCSV(t, reader)
			})

			t.Run("without_gzip", func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/v2/hash/risklist?format=csv/splunk&gzip=false&list=default", nil)

				h.ServeHTTP(w, r)
				resp := w.Result()
				defer resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain")

				readAllCSV(t, resp.Body)
			})
		})
	}
}

func readAllCSV(t *testing.T, r io.Reader) {
	csv := csv.NewReader(r)
	csv.ReuseRecord = true
	var count int
	for {
		record, err := csv.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		assert.NotEmpty(t, record)
		count++
	}
	assert.NotZero(t, count, "No CSV records received.")
	t.Logf("Got %d CSV records.", count)
}
