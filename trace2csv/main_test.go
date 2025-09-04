package main

import "testing"

func TestIsSecret(t *testing.T) {
	// Assortment of headers grepped from elastic/integrations system tests.
	testHeaders := map[string]bool{
		"Accept":          false,
		"Auth-Key":        true,
		"Authorization":   true,
		"Content-Type":    false,
		"Cookie":          true,
		"Set-Cookie":      true,
		"TMV1-Query":      false,
		"X-Amz-Target":    false,
		"X-Api-Key":       true,
		"X-ApiKeys":       true,
		"X-Auth-Token":    true,
		"X-RFToken":       true,
		"_marker":         false,
		"accept":          false,
		"after":           false,
		"api-key":         true,
		"api-version":     false,
		"apikey":          true,
		"appId":           false,
		"authorization":   true,
		"body":            false,
		"confidence":      false,
		"cursor":          false,
		"days":            false,
		"deltaTime":       false,
		"endDate":         false,
		"enddate":         false,
		"fields":          false,
		"filterField":     false,
		"format":          false,
		"from":            false,
		"grant_type":      false,
		"gzip":            false,
		"headers":         false,
		"itype":           false,
		"key":             false,
		"lastUpdatedFrom": false,
		"last_seen":       false,
		"limit":           false,
		"methods":         false,
		"min_timestamp":   false,
		"modified_ts__lt": false,
		"next":            false,
		"offset":          false,
		"oldest":          false,
		"order":           false,
		"order_by":        false,
		"page":            false,
		"path":            false,
		"per_page":        false,
		"query":           false,
		"query_params":    false,
		"request_body":    false,
		"request_headers": false,
		"responses":       false,
		"severity":        false,
		"since":           false,
		"sort":            false,
		"sort_order":      false,
		"start":           false,
		"startDate":       false,
		"startid":         false,
		"starting_after":  false,
		"state":           false,
		"to":              false,
		"updateDateFrom":  false,
		"version":         false,
		"wantscandetails": false,
		"x-api-key":       true,
		"x-apikey":        true,
		"x-auth-email":    false,
		"x-auth-key":      true,
		"x-redlock-auth":  true,
	}

	for header, want := range testHeaders {
		t.Run(header, func(t *testing.T) {
			got := isSecret(header)
			if got != want {
				t.Fatalf("got %v; want %v", got, want)
			}
		})
	}
}
