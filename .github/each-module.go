package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var command string

func init() {
	flag.StringVar(&command, "cmd", "", "command to run via 'sh -c'")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if command == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := forEachModule(command); err != nil {
		log.Fatalln(err)
	}
}

func forEachModule(cmd string) error {
	goModFile, err := filepath.Glob("*/go.mod")
	if err != nil {
		return err
	}

	var errs []error
	for _, goMod := range goModFile {
		moduleDir := filepath.Dir(goMod)
		if err = runCommand(moduleDir, cmd); err != nil {
			errs = append(errs, fmt.Errorf("failed running '%v' in '%v': %w", cmd, moduleDir, err))
		}
	}

	return combineErrors(errs...)
}

func runCommand(path, command string) error {
	log.Printf("=== %v: %v", path, command)
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "PATH_PREFIX=./"+path)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type multiError struct {
	errors []error
}

func combineErrors(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}

	return &multiError{errors: errs}
}

func (m *multiError) Error() string {
	if len(m.errors) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s\n\n", m.errors[0])
	}

	points := make([]string, len(m.errors))
	for i, err := range m.errors {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(m.errors), strings.Join(points, "\n\t"))
}
