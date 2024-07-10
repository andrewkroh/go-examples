// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This simulates the Qualys VMDR user activity log.
// It emulates the time and offset based queries by simulating a world in which
// a new event happens every 5 minutes.
//
// References
//
//   - https://docs.qualys.com/en/vm/api/users/activity/export_activity.htm
//   - https://cdn2.qualys.com/docs/qualys-api-vmpc-user-guide.pdf
package main

import (
	"bytes"
	"cmp"
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// newDataInterval controls how often new events happen in the simulator.
const newDataInterval = 5 * time.Minute

//go:embed assets/banner.txt
var banner string

var (
	addr     string // Listen address
	username string
	password string
)

func init() {
	flag.StringVar(&addr, "http", "localhost:9903", "listen address")
	flag.StringVar(&username, "username", "", "username (required)")
	flag.StringVar(&password, "password", "", "password (required)")
}

func main() {
	flag.Parse()

	if username == "" || password == "" {
		flag.Usage()
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.NewRoute().Path("/api/2.0/fo/activity_log/").Handler(
		newBasicAuthHandler(username, password, newExportActivityHandler()))
	h := handlers.CombinedLoggingHandler(os.Stderr, r)

	done := make(chan struct{})
	go func() {
		defer close(done)
		log.Fatal(http.ListenAndServe(addr, h))
	}()

	log.Println(banner)
	log.Printf("Listening on http://%s/api/2.0/fo/activity_log/", addr)
	log.Println()
	log.Println("Username:", username)
	log.Println("Password:", password)
	<-done
}

type basicAuthHandler struct {
	username, password string
	next               http.Handler
}

func newBasicAuthHandler(username, password string, next http.Handler) basicAuthHandler {
	return basicAuthHandler{
		username: username,
		password: password,
		next:     next,
	}
}

func (h basicAuthHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	u, p, found := req.BasicAuth()
	if !found {
		log.Println("ERROR missing basic auth")
		http.Error(w, "Missing basic auth in request.", http.StatusUnauthorized)
		return
	}
	if u != h.username || p != h.password {
		log.Println("ERROR invalid username/password")
		http.Error(w, "Invalid username/password", http.StatusUnauthorized)
		return
	}

	h.next.ServeHTTP(w, req)
}

type exportActivityHandler struct{}

func newExportActivityHandler() *exportActivityHandler {
	return &exportActivityHandler{}
}

func (*exportActivityHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC()

	// Query params
	var (
		action          string
		userAction      string
		actionDetails   string
		username        string
		since           = now.Add(-24 * time.Hour) // Made up default value.
		until           = now
		userRole        bool
		outputFormat    string
		truncationLimit int

		// idMax is an undocumented parameter observed in responses requiring pagination.
		// This emulator treats id_max as a Unix timestamp in seconds that takes
		// precedence over the 'since' parameter and is inclusive.
		idMax *time.Time
	)

	if h := req.Header.Get("X-Requested-With"); h == "" {
		http.Error(w, "Missing X-Requested-With", http.StatusBadRequest)
		return
	}

	var err error
	for query, values := range req.URL.Query() {
		if len(values) == 0 {
			continue
		}
		val := values[0]

		switch query {
		case "action":
			action = val
		case "user_action":
			userAction = val
		case "action_details":
			actionDetails = val
		case "username":
			username = val
		case "since_datetime":
			since, err = time.Parse(time.RFC3339, val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
		case "until_datetime":
			until, err = time.Parse(time.RFC3339, val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
		case "user_role":
			userRole, err = strconv.ParseBool(val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
		case "output_format":
			outputFormat = val
		case "truncation_limit":
			truncationLimit, err = strconv.Atoi(val)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
		case "id_max":
			sec, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				http.Error(w, "invalid query param "+query, http.StatusBadRequest)
				return
			}
			t := time.Unix(sec, 0)
			idMax = &t
		default:
			http.Error(w, "invalid query param "+query, http.StatusBadRequest)
			return
		}
	}

	// Currently unused parameters.
	_ = userAction
	_ = actionDetails
	_ = username
	_ = userRole
	_ = outputFormat

	w.Header().Set("Content-Type", "text/csv")

	if action != "list" {
		http.Error(w, "'action' query param is required", http.StatusBadRequest)
		return
	}

	start := since
	if idMax != nil {
		start = *idMax
	}
	events, nextPage, err := generateSampleData(start, until, truncationLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	end := until
	if len(events) > 0 {
		end = events[len(events)-1].Date
	}
	log.Printf("Generated %d events covering from %v to %v (truncated=%v).",
		len(events), start, end, !nextPage.IsZero())

	// Example response in docs shows data in descending order.
	slices.SortFunc(events, func(a, b event) int {
		return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
	})

	buf := bytes.NewBuffer(nil)
	buf.WriteString("----BEGIN_RESPONSE_BODY_CSV\n")
	_ = writeCSV(buf, events)
	buf.WriteString("----END_RESPONSE_BODY_CSV\n")

	if !nextPage.IsZero() {
		buf.WriteString("----BEGIN_RESPONSE_FOOTER_CSV\n")
		buf.WriteString("WARNING\n")
		buf.WriteString(`"CODE","TEXT","URL"` + "\n")
		fmt.Fprintf(w, `"1980","%d record limit exceeded. Use URL to get next batch of results.","%s"`+"\n", truncationLimit, paginationURL(req, nextPage))
		buf.WriteString("----END_RESPONSE_FOOTER_CSV\n")
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Println("ERROR writing response:", err)
		return
	}
}

func writeCSV(w io.Writer, events []event) error {
	csv := csv.NewWriter(w)
	defer csv.Flush()
	err := csv.Write([]string{
		"Date", "Action", "Module", "Details", "User Name", "User Role", "User IP",
	})
	if err != nil {
		return err
	}

	for _, e := range events {
		err = csv.Write([]string{
			e.Date.UTC().Format("2006-01-02T15:04:05Z"), e.Action, e.Module, e.Details, e.Username, e.UserRole, e.UserIP,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func paginationURL(req *http.Request, idMax time.Time) string {
	u := url.URL{
		Scheme:   "http",
		Host:     req.Host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	q := u.Query()
	q.Set("id_max", strconv.FormatInt(idMax.Unix(), 10))
	u.RawQuery = q.Encode()
	return u.String()
}

var (
	actions = []string{
		"login",
		"request",
		"set",
		"add",
		"create",
	}
	modules = []string{
		"auth",
		"host_attribute",
		"option",
		"network",
	}
	details = []string{
		"user_logged in",
		"API: /api/2.0/fo/activity_log/index.php",
		"comment=[vvv] for 11.11.11.4",
		"11.11.11.4 added to both VM-PC license",
		"New Network: 'abc'",
	}
	usernames = []string{
		"saand_rn",
		"joe",
	}
	userRoles = []string{
		"Manager",
	}
	userIPs = []string{
		"10.113.195.136",
		"10.113.14.208",
	}
)

type event struct {
	Date     time.Time
	Action   string
	Module   string
	Details  string
	Username string
	UserRole string
	UserIP   string
}

func newRandomEvent(ts time.Time) event {
	return event{
		Date:     ts,
		Action:   actions[rand.Intn(len(actions))],
		Module:   modules[rand.Intn(len(modules))],
		Details:  details[rand.Intn(len(details))],
		Username: usernames[rand.Intn(len(usernames))],
		UserRole: userRoles[rand.Intn(len(userRoles))],
		UserIP:   userIPs[rand.Intn(len(userIPs))],
	}
}

func generateSampleData(from, to time.Time, limit int) (events []event, next time.Time, err error) {
	if !from.Before(to) {
		return nil, time.Time{}, fmt.Errorf("'from' must be earlier than 'to'")
	}

	t := from
	events = append(events, newRandomEvent(t))
	for {
		t = t.Add(newDataInterval)
		if !t.Before(to) {
			break
		}
		if limit > 0 && len(events) >= limit {
			next = t
			break
		}
		events = append(events, newRandomEvent(t))
	}

	return events, next, nil
}
