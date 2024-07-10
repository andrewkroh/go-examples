package main

import (
	"bytes"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestExportActivityHandler_ServeHTTP(t *testing.T) {
	t.Run("last_24h", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/2.0/fo/activity_log/?action=list", nil)
		req.Header.Set("X-Requested-With", "Go")

		resp := httptest.NewRecorder()
		h := newExportActivityHandler()
		h.ServeHTTP(resp, req)

		body, err := io.ReadAll(resp.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s", body)

		if !bytes.HasPrefix(body, []byte("----BEGIN_RESPONSE_BODY_CSV\n")) {
			t.Error("Missing header in response body")
		}
		if !bytes.HasSuffix(body, []byte("----END_RESPONSE_BODY_CSV\n")) {
			t.Error("Missing trailer in response body")
		}

		const eventCount = 24 * 60 / 5
		const headerFooterCount = 3
		const lineCount = headerFooterCount + eventCount

		if c := bytes.Count(body, []byte("\n")); c != lineCount {
			t.Errorf("body contained %d lines, expected %d", c, lineCount)
		}
	})

	t.Run("truncated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/2.0/fo/activity_log/?action=list&truncation_limit=1", nil)
		req.Header.Set("X-Requested-With", "Go")

		resp := httptest.NewRecorder()
		h := newExportActivityHandler()
		h.ServeHTTP(resp, req)

		body, err := io.ReadAll(resp.Result().Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s", body)

		if !bytes.Contains(body, []byte("----BEGIN_RESPONSE_FOOTER_CSV\n")) {
			t.Error("Missing footer start in response body")
		}
		if !bytes.HasSuffix(body, []byte("----END_RESPONSE_FOOTER_CSV\n")) {
			t.Error("Missing footer end in response body")
		}

		const eventCount = 1
		const headerFooterCount = 8
		const lineCount = headerFooterCount + eventCount

		if c := bytes.Count(body, []byte("\n")); c != lineCount {
			t.Errorf("body contained %d lines, expected %d", c, lineCount)
		}
	})

	t.Run("id_max", func(t *testing.T) {
		idMax := time.Now().Add(-1 * time.Minute).Unix()

		req := httptest.NewRequest("GET", "/api/2.0/fo/activity_log/?action=list&id_max="+strconv.FormatInt(idMax, 10), nil)
		req.Header.Set("X-Requested-With", "Go")

		resp := httptest.NewRecorder()
		h := newExportActivityHandler()
		h.ServeHTTP(resp, req)

		body, err := io.ReadAll(resp.Result().Body)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.HasPrefix(body, []byte("----BEGIN_RESPONSE_BODY_CSV\n")) {
			t.Error("Missing header in response body")
		}
		if !bytes.HasSuffix(body, []byte("----END_RESPONSE_BODY_CSV\n")) {
			t.Error("Missing trailer in response body")
		}

		const eventCount = 1
		const headerFooterCount = 3
		const lineCount = headerFooterCount + eventCount

		if c := bytes.Count(body, []byte("\n")); c != lineCount {
			t.Errorf("body contained %d lines, expected %d", c, lineCount)
		}
	})
}
