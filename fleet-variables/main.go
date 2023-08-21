package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/andrewkroh/go-fleetpkg"
)

var (
	integrationsRepoPath string
	ownerFilter          string
)

func init() {
	flag.StringVar(&integrationsRepoPath, "integ-dir", "", "Path to elastic/integrations repo.")
	flag.StringVar(&ownerFilter, "owner", "", "Select only packages owned by this value.")
}

type Variable struct {
	fleetpkg.Var
	Integration string
	Owner       string
}

func main() {
	flag.Parse()

	if integrationsRepoPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	commit, err := elasticIntegrationsCommit(integrationsRepoPath)
	if err != nil {
		log.Fatal(err)
	}

	manifests, err := filepath.Glob(filepath.Join(integrationsRepoPath, "packages/*/manifest.yml"))
	if err != nil {
		log.Fatal(err)
	}

	var variables []Variable

	for _, path := range manifests {
		vars, err := readVariables(filepath.Dir(path))
		if err != nil {
			log.Fatal(err)
		}

		variables = append(variables, vars...)
	}

	sort.Slice(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})

	if err := writeCSV(os.Stdout, commit, variables); err != nil {
		log.Fatal(err)
	}
}

func readVariables(packagePath string) ([]Variable, error) {
	integ, err := fleetpkg.Read(packagePath)
	if err != nil {
		log.Fatal(err)
	}

	if ownerFilter != "" && integ.Manifest.Owner.Github != ownerFilter {
		return nil, nil
	}

	var variables []Variable
	appendVar := func(v fleetpkg.Var) {
		if v.Secret != nil && *v.Secret {
			return
		}

		variables = append(variables, Variable{
			Integration: integ.Manifest.Name,
			Var:         v,
			Owner:       integ.Manifest.Owner.Github,
		})
	}

	// Integration-level variables.
	for _, v := range integ.Manifest.Vars {
		appendVar(v)
	}

	// Policy template-level variables.
	for _, policyTemplate := range integ.Manifest.PolicyTemplates {
		for _, v := range policyTemplate.Vars {
			appendVar(v)
		}

		// Input-level variables.
		for _, input := range policyTemplate.Inputs {
			for _, v := range input.Vars {
				appendVar(v)
			}
		}
	}

	// Data stream-level variables.
	for _, ds := range integ.DataStreams {
		for _, stream := range ds.Manifest.Streams {
			for _, v := range stream.Vars {
				appendVar(v)
			}
		}
	}

	return variables, nil
}

func writeCSV(w io.Writer, commit string, vars []Variable) error {
	csvWriter := csv.NewWriter(w)
	err := csvWriter.Write([]string{
		"name",
		"type",
		"description",
		"integration",
		"path",
		"line",
		"url",
		"owner",
	})
	if err != nil {
		return err
	}

	for _, v := range vars {
		_, repoPath, _ := strings.Cut(v.Path(), "integrations/")

		err = csvWriter.Write([]string{
			v.Name,
			v.Type,
			v.Description,
			v.Integration,
			repoPath,
			strconv.Itoa(v.Line()),
			fmt.Sprintf("https://github.com/elastic/integrations/blob/%v/%v#L%d", commit, repoPath, v.Line()),
			v.Owner,
		})
		if err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return nil
}

func elasticIntegrationsCommit(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(stdout)), nil
}
