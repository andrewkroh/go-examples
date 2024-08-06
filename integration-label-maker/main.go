package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

// Goals
// 1. Generate a set of labels from the integrations found in the repository.
// 2. Use github API to fetch all labels beginning with "integration:" (case-insensitive).
// 3. Output the list of new labels.
// 4. Output the list of labels that have changes.
// 5. Output the list of labels that do not match any known integration.
// 6. Output list of gh CLI commands to make changes (create, update).

// Requirements
//   - must have gh CLI installed (brew install gh)
//   - must be authenticated (gh auth login)

const labelColor = "FFFFFF"

const inactiveLabelColor = "D3D3D3"

var integrationsRepoPath string

func init() {
	flag.StringVar(&integrationsRepoPath, "integrations-repo", "", "Path to elastic/integrations repo (required).")
}

func main() {
	flag.Parse()

	if integrationsRepoPath == "" {
		flag.Usage()
		return
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	labels, err := IntegrationLabels(integrationsRepoPath)
	if err != nil {
		return err
	}
	fmt.Println("----FROM SOURCE----")
	for _, label := range labels {
		fmt.Println(label.Name, "|", label.Description)
	}
	labelsMap := make(map[string]GithubLabel, len(labels))
	for _, label := range labels {
		labelsMap[label.Name] = label
	}

	existingLabels, err := GithubRepoLabels()
	if err != nil {
		return err
	}
	existingLabels = filterLabels(existingLabels, func(l GithubLabel) bool {
		return strings.HasPrefix(strings.ToLower(l.Name), "integration:")
	})
	fmt.Println("----FROM GITHUB----")
	for _, label := range existingLabels {
		fmt.Println(label.Name, "|", label.Description)
	}
	existingLabelsMap := make(map[string]GithubLabel, len(existingLabels))
	for _, label := range existingLabels {
		existingLabelsMap[label.Name] = label
	}

	var needsCreated []GithubLabel
	needsChanged := map[GithubLabel]GithubLabel{} // existing -> want
	for _, label := range labels {
		existing, found := existingLabelsMap[label.Name]
		if found {
			if label != existing {
				needsChanged[existing] = label
			}
		} else {
			needsCreated = append(needsCreated, label)
		}
	}

	var noAssociation []GithubLabel
	for _, existingLabel := range existingLabelsMap {
		if _, found := labelsMap[existingLabel.Name]; !found {
			noAssociation = append(noAssociation, existingLabel)
		}
	}

	var ghCommands []string
	fmt.Println("----NEEDS CREATED----")
	slices.SortFunc(needsCreated, compareLabel)
	for _, label := range needsCreated {
		fmt.Println(label.Name, "|", label.Description)

		ghCommands = append(ghCommands, fmt.Sprintf(
			"gh label create --repo=elastic/integrations %q --color %q --description %q",
			label.Name, label.Color, label.Description))
	}

	fmt.Println("----NEEDS CHANGED----")
	keys := maps.Keys(needsChanged)
	slices.SortFunc(keys, compareLabel)
	for _, got := range keys {
		want := needsChanged[got]
		fmt.Println(got.Name, "|")
		if got.Color != want.Color {
			fmt.Printf("  color: %s -> %s\n", got.Color, want.Color)
		}
		if got.Description != want.Description {
			fmt.Printf("  description: %s -> %s\n", got.Description, want.Description)
		}
		ghCommands = append(ghCommands, fmt.Sprintf(
			"gh label edit --repo=elastic/integrations %q --color %q --description %q",
			want.Name, want.Color, want.Description))
	}

	fmt.Println("----NO ASSOCIATION----")
	slices.SortFunc(noAssociation, compareLabel)
	for _, label := range noAssociation {
		fmt.Println(label.Name, "|", label.Description)

		if !strings.EqualFold(label.Color, inactiveLabelColor) {
			ghCommands = append(ghCommands, fmt.Sprintf(
				"gh label edit --repo=elastic/integrations %q --color %q",
				label.Name, inactiveLabelColor))
		}
	}

	fmt.Println("----COMMANDS TO CREATE/EDIT----")
	for _, cmd := range ghCommands {
		fmt.Println(cmd)
	}

	return nil
}

type GithubLabel struct {
	Name        string
	Description string
	Color       string
}

func compareLabel(a, b GithubLabel) int {
	return cmp.Compare(a.Name, b.Name)
}

type Manifest struct {
	DirectoryName string `yaml:"-"`     // Directory name (not the package name).
	Title         string `yaml:"title"` // Integration Title
}

func IntegrationLabels(integrationsRepoPath string) ([]GithubLabel, error) {
	manifests, err := filepath.Glob(filepath.Join(integrationsRepoPath, "packages/*/manifest.yml"))
	if err != nil {
		return nil, err
	}

	labels := make([]GithubLabel, 0, len(manifests))
	for _, manifestPath := range manifests {
		content, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, err
		}

		var manifest Manifest
		if err := yaml.Unmarshal(content, &manifest); err != nil {
			return nil, err
		}

		name := "Integration:" + filepath.Base(filepath.Dir(manifestPath))
		if len(name) > 50 {
			name = name[:50] // Max label length is 50 characters.
		}

		labels = append(labels, GithubLabel{
			Name:        name,
			Description: manifest.Title,
			Color:       labelColor,
		})
	}

	return labels, nil
}

func GithubRepoLabels() ([]GithubLabel, error) {
	out := new(bytes.Buffer)

	cmd := exec.Command("gh", "label", "list",
		"--repo=elastic/integrations",
		"--json=name,description,color",
		"--limit=10000")
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var labels []GithubLabel
	if err := json.NewDecoder(out).Decode(&labels); err != nil {
		return nil, err
	}

	return labels, nil
}

func filterLabels(labels []GithubLabel, predicate func(l GithubLabel) bool) []GithubLabel {
	var result []GithubLabel
	for _, label := range labels {
		if predicate(label) {
			result = append(result, label)
		}
	}
	return result
}
