package main

import (
	"flag"
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
		UpdateScripts: *update,
	}
	testscript.Run(t, p)
}
