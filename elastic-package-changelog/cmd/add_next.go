package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/andrewkroh/go-examples/elastic-package-changelog/pkg/changelog"
)

type addNextCmd struct {
	cmd *cobra.Command

	// Options
	changelogFile     string
	manifestFile      string
	description       string
	changeType        string
	pullRequestNumber int
}

func newFlattenCmd() *cobra.Command {
	r := &addNextCmd{
		cmd: &cobra.Command{
			Use:   "add-next [flags]",
			Short: "Add a change under a new (next) version.",
		},
	}

	r.cmd.PreRunE = func(c *cobra.Command, args []string) error {
		if r.description == "" {
			fmt.Fprintln(c.ErrOrStderr(), "--description is required.")
			return c.Usage()
		}
		if r.changeType == "" {
			fmt.Fprintln(c.ErrOrStderr(), "--type is required.")
			return c.Usage()
		}
		return nil
	}

	r.cmd.RunE = func(c *cobra.Command, args []string) error {
		return r.Run()
	}

	r.cmd.Flags().StringVarP(&r.manifestFile, "manifest", "m", "manifest.yml", "Manifest file to modify.")
	r.cmd.Flags().StringVarP(&r.changelogFile, "changelog", "c", "changelog.yml", "Changelog file to modify.")
	r.cmd.Flags().StringVar(&r.changeType, "type", "", "Change type (enhancement, bugfix, breaking-change).")
	r.cmd.Flags().IntVar(&r.pullRequestNumber, "pr", 0, "Pull request number.")
	r.cmd.Flags().StringVarP(&r.description, "description", "d", "", "Description of change (use a proper sentence). Target audience is end users.")

	return r.cmd
}

func (c *addNextCmd) Run() error {
	// Input
	var in io.Reader
	if c.changelogFile == "-" {
		in = c.cmd.InOrStdin()
	} else {
		f, err := os.Open(c.changelogFile)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	// Transform
	version, out, err := c.addNext(in)
	if err != nil {
		return err
	}

	// Output
	if c.changelogFile == "-" {
		c.cmd.OutOrStdout().Write(out)
	} else {
		f, err := os.Create(c.changelogFile)
		if err != nil {
			return nil
		}
		defer f.Close()

		if _, err = f.Write(out); err != nil {
			return err
		}
	}

	// Update manifest.yml
	return c.setManifestVersion(version)
}

func (c *addNextCmd) addNext(r io.Reader) (version string, changelogYAML []byte, err error) {
	var cl changelog.Changelog
	if err := yaml.NewDecoder(r).Decode(&cl); err != nil {
		return "", nil, fmt.Errorf("failed decoded yaml: %w", err)
	}

	if len(cl) == 0 {
		return "", nil, errors.New("changelog is empty")
	}

	latest, err := changelog.NewReleaseFromNode(cl[0])
	if err != nil {
		return "", nil, err
	}

	rel, err := c.newRelease(latest.Version)
	if err != nil {
		return "", nil, err
	}

	relNode, err := rel.ToYAMLNode()
	if err != nil {
		return "", nil, err
	}

	// Move comments to first node.
	relNode.HeadComment = cl[0].HeadComment
	cl[0].HeadComment = ""

	// Insert new release at top.
	cl = append([]yaml.Node{*relNode}, cl...)

	buf := new(bytes.Buffer)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err = enc.Encode(cl); err != nil {
		return "", nil, err
	}

	return string(rel.Version), buf.Bytes(), nil
}

func (c *addNextCmd) newRelease(currentVersion changelog.VersionString) (*changelog.Release, error) {
	ver, err := semver.NewVersion(string(currentVersion))
	if err != nil {
		return nil, err
	}

	ct, err := changelog.NewChangeType(c.changeType)
	if err != nil {
		return nil, err
	}

	switch ct {
	case changelog.Bugfix:
		ver.BumpPatch()
	case changelog.Enhancement:
		ver.BumpMinor()
	case changelog.BreakingChange:
		ver.BumpMajor()
	}

	pr := "https://github.com/elastic/integrations/pull/"
	if c.pullRequestNumber > 0 {
		pr += strconv.Itoa(c.pullRequestNumber)
	} else {
		pr += "{{ PULL_REQUEST_NUMBER }}"
	}

	return &changelog.Release{
		Version: changelog.VersionString(ver.String()),
		Changes: []changelog.Change{
			{
				Description: c.description,
				Type:        ct.String(),
				Link:        pr,
			},
		},
	}, nil
}

func (c *addNextCmd) setManifestVersion(version string) error {
	if c.manifestFile == "" {
		return nil
	}

	yml, err := ioutil.ReadFile(c.manifestFile)
	if err != nil {
		return fmt.Errorf("failed to read manifest from %q: %w", c.manifestFile, err)
	}

	var manifest changelog.Manifest
	if err := yaml.Unmarshal(yml, &manifest); err != nil {
		return err
	}

	if err := manifest.SetVersion(version); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err = enc.Encode(manifest); err != nil {
		return err
	}

	return ioutil.WriteFile(c.manifestFile, buf.Bytes(), 0o644)
}
