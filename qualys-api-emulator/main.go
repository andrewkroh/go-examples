// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This simulates the Qualys VMDR user activity log.
// It emulates the time and offset based queries by simulating a world in which
// a new event happens every 5 minutes.
//
// References
//
//	https://docs.qualys.com/en/vm/api/users/activity/export_activity.htm
package main

import (
	"cmp"
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
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
	)

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

	events, err := generateSampleData(since, until, truncationLimit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Generated %d events covering from %v to %v", len(events), since, until)

	// Example response in docs shows data in descending order.
	slices.SortFunc(events, func(a, b event) int {
		return cmp.Compare(b.Date.UnixNano(), a.Date.UnixNano())
	})

	if err := writeCSV(w, events); err != nil {
		log.Printf("ERROR: %v", err)
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

func generateSampleData(from, to time.Time, limit int) (events []event, err error) {
	if !from.Before(to) {
		return nil, fmt.Errorf("'from' must be earlier than 'to'")
	}

	t := from
	for {
		t = t.Add(newDataInterval)
		if !t.Before(to) {
			break
		}
		if limit > 0 && len(events) >= limit {
			break
		}
		events = append(events, newRandomEvent(t))
	}

	return events, nil
}
