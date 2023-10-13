package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/fatih/color"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/txtar"
)

var update = flag.Bool("update", false, "Update golden files.")

func TestGoldenFiles(t *testing.T) {
	// Remove the global color blocker that is based on TTY.
	color.NoColor = false

	t.Run("key-value", func(t *testing.T) {
		testGoldenFile(t, "testdata/_key-value.golden.txtar", &keyValueFormatter{})
	})
	t.Run("key-value-color", func(t *testing.T) {
		testGoldenFile(t, "testdata/_key-value-color.golden.txtar", &keyValueFormatter{withColor: true})
	})
	t.Run("json", func(t *testing.T) {
		testGoldenFile(t, "testdata/_json.golden.txtar", &jsonFormatter{})
	})
}

func testGoldenFile(t *testing.T, goldenFile string, format formatter) {
	ar := &txtar.Archive{
		Comment: []byte(t.Name()),
	}

	matches, _ := filepath.Glob("testdata/*json")
	for _, path := range matches {
		addToArchive(t, path, ar, format)
	}

	if *update {
		err := os.WriteFile(goldenFile, txtar.Format(ar), 0o600)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// Compare results.
		goldenAr, err := txtar.ParseFile(goldenFile)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(goldenAr.Files, ar.Files); diff != "" {
			t.Errorf("golden data mismatch (-want +got):\n%s", diff)
		}
	}
}

func addToArchive(t *testing.T, path string, ar *txtar.Archive, format formatter) {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	outBuf := new(bytes.Buffer)
	err = run(f, format, outBuf)
	if err != nil {
		t.Fatal(err)
	}

	ar.Files = append(ar.Files, txtar.File{
		Name: path,
		Data: outBuf.Bytes(),
	})
}
