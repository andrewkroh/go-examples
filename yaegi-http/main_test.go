package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var update = flag.Bool("update", false, "update testscript output files")

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"yaegi-http": main,
	})
}

func TestScripts(t *testing.T) {
	t.Parallel()

	programs, _ := filepath.Abs(filepath.Join("testdata", "programs"))
	p := testscript.Params{
		Dir: filepath.Join("testdata"),
		Setup: func(env *testscript.Env) error {
			// Allow a referencing programs which are outside the WORDIR.
			env.Vars = append(env.Vars, "PROGRAMS="+programs)
			return nil
		},
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"serve": serve,
		},
		UpdateScripts: *update,
	}
	testscript.Run(t, p)
}

func serve(ts *testscript.TestScript, neg bool, args []string) {
	server(ts, neg, "serve", httptest.NewServer, args)
}

func server(ts *testscript.TestScript, neg bool, name string, newServer func(handler http.Handler) *httptest.Server, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", name)
	}
	if len(args) != 1 && len(args) != 3 {
		ts.Fatalf("usage: %s body [user password]", name)
	}
	var user, pass string
	body, err := os.ReadFile(ts.MkAbs(args[0]))
	ts.Check(err)
	if len(args) == 3 {
		user = args[1]
		pass = args[2]
	}
	srv := newServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		u, p, _ := req.BasicAuth()
		// Obvious security anti-patterns are obvious; for testing.
		if user != "" && user != u {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("user mismatch"))
			return
		}
		if pass != "" && pass != p {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("password mismatch"))
			return
		}

		// Write an ETag header to allow client caching (this could be more generalized).
		sum := md5.Sum(body)
		h := hex.EncodeToString(sum[:])
		w.Header().Set("ETag", h)

		w.Write(body)
	}))
	ts.Setenv("YAEGI_HTTP_URL", srv.URL)
	ts.Defer(func() { srv.Close() })
}
