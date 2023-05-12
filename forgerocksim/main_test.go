package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadLogs(t *testing.T) {
	logs, err := loadLogs("testdata/logs.json")
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, logs, 2)
}

func TestCookie(t *testing.T) {
	c1 := newCookie([]string{"am-activity"}, 2, 1, time.Now(), time.Now())
	base64Cookie := c1.Encode()

	c2, err := decodeCookie(base64Cookie)
	if err != nil {
		t.Fatal(err)
	}

	// Comparing time.Time does not work as expected so use JSON.
	a, _ := json.Marshal(c1)
	b, _ := json.Marshal(c2)
	assert.JSONEq(t, string(a), string(b))
}

func TestMonitoringLogsHandler(t *testing.T) {
	logs, err := loadLogs("testdata/logs.json")
	if err != nil {
		t.Fatal(err)
	}
	h := newMonitoringLogsHandler(logs)

	const cookie = `{"sources":["am-activity"],"page_size":1,"begin_time":"2022-12-07T16:00:02.945435858Z","end_time":"2022-12-07T16:33:00Z","next_page":1}`
	base64Cookie := base64.StdEncoding.EncodeToString([]byte(cookie))

	t.Run("range", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/monitoring/logs?source=am-activity&beginTime=2022-12-07T16:00:02.945435858Z&endTime=2022-12-07T16:33:00Z&_pageSize=1", nil)

		h.ServeHTTP(w, r)
		resp := w.Result()
		defer resp.Body.Close()

		var data map[string]interface{}
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&data); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, data["result"], 1)
		assert.Equal(t, base64Cookie, data["pagedResultsCookie"])
	})

	t.Run("paging", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/monitoring/logs?_pagedResultsCookie="+base64Cookie, nil)

		h.ServeHTTP(w, r)
		resp := w.Result()
		defer resp.Body.Close()

		var data map[string]interface{}
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&data); err != nil {
			t.Fatal(err)
		}

		assert.Len(t, data["result"], 1)
		t.Logf("%#v", data)
	})
}
