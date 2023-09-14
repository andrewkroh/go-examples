package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

//go:embed assets/mapping.json
var indexMapping string

var (
	packagesDir      string
	index            string // Index to write to.
	elasticsearchURL string
	username         string
	password         string
	apiKey           string
	insecure         bool
)

func init() {
	flag.StringVar(&packagesDir, "packages-dir", "", "Directory containing Fleet packages.")
	flag.StringVar(&index, "index", "fleet-integrations", "name of index to create")
	flag.StringVar(&elasticsearchURL, "es-url", "http://localhost:9200", "Elasticsearch URL")
	flag.StringVar(&username, "u", "", "Elasticsearch username")
	flag.StringVar(&password, "p", "", "Elasticsearch password")
	flag.StringVar(&apiKey, "api-key", "", "Elasticsearch API key")
	flag.BoolVar(&insecure, "insecure", false, "Proceed and operate even for TLS server connections considered insecure.")
	flag.Usage = usage
}

type jsonTime time.Time

func (t jsonTime) MarshalJSON() ([]byte, error) {
	v := time.Time(t).UTC().Format(time.RFC3339Nano)
	return json.Marshal(v)
}

type commonFields struct {
	Timestamp      jsonTime `json:"@timestamp"`
	Type           []string `json:"@type"`
	Commit         string   `json:"@commit"`
	URL            string   `json:"@url,omitempty"`
	Integration    string   `json:"@integration"`
	DataStream     string   `json:"@data_stream,omitempty"`
	Input          []string `json:"@input,omitempty"`
	PolicyTemplate []string `json:"@policy_template,omitempty"`
	Owner          string   `json:"@owner"`
	Attributes     []string `json:"@attributes,omitempty"` // Attributes holds deprecated and rsa2elk.
}

type manifest struct {
	commonFields
	fleetpkg.Manifest
}

type policyTemplate struct {
	commonFields
	fleetpkg.PolicyTemplate
}

type buildManifest struct {
	commonFields
	fleetpkg.BuildManifest
}

type dataStreamManifest struct {
	commonFields
	fleetpkg.DataStreamManifest
}

type sampleEvent struct {
	commonFields
	SampleEvent map[string]any `json:"sample_event,omitempty"`
}

type variable struct {
	commonFields
	fleetpkg.Var
}

type field struct {
	commonFields
	fleetpkg.Field
}

func usage() {
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "    fleetpkg-indexer [flags]")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if packagesDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	integrations, err := loadFleetPackages()
	if err != nil {
		slog.Error("Failed to load integrations.", slog.String("error", err.Error()))
		os.Exit(1)
	}

	commit, err := elasticIntegrationsCommit(packagesDir)
	if err != nil {
		commit = "unknown"
	}

	ts, _ := elasticIntegrationsCommitTimestamp(packagesDir, commit)
	commitTime := jsonTime(ts)

	// WARNING: This is a mess, apologies. It was a rapid hack to answer
	// some questions and explore the data.

	for _, integ := range integrations {
		var allPolicyTemplateNames []string
		var allPackageInputs []string
		for _, pt := range integ.Manifest.PolicyTemplates {
			allPolicyTemplateNames = append(allPolicyTemplateNames, pt.Name)

			if pt.Input != "" {
				allPackageInputs = append(allPackageInputs, pt.Input)
			}
			for _, input := range pt.Inputs {
				allPackageInputs = append(allPackageInputs, input.Type)
			}
		}

		deprecated := strings.Contains(strings.ToLower(integ.Manifest.Description), "deprecated")
		rsa2elk, err := fileContains(filepath.Join(integ.Path(), "data_stream/*/agent/stream/*.hbs"), []byte("nwparser"))
		if err != nil {
			slog.Warn("Failed to determine if package is rsa2elk", slog.String("integration", integ.Manifest.Name), slog.String("error", err.Error()))
		}
		var attributes []string
		if deprecated {
			attributes = append(attributes, "deprecated")
		}
		if rsa2elk {
			attributes = append(attributes, "rsa2elk")
		}

		var docs []any

		// Root-level variables
		for _, v := range integ.Manifest.Vars {
			docs = append(docs, variable{
				commonFields: commonFields{
					Timestamp:      commitTime,
					Type:           []string{"package_variable", "variable"},
					Commit:         commit,
					URL:            sourceURLWithLine(commit, v.FileMetadata),
					Integration:    integ.Manifest.Name,
					PolicyTemplate: allPolicyTemplateNames,
					Owner:          integ.Manifest.Owner.Github,
					Input:          allPackageInputs,
					Attributes:     attributes,
				},
				Var: v,
			})
		}

		// Policy template variables
		for _, pt := range integ.Manifest.PolicyTemplates {
			for j, input := range pt.Inputs {
				for _, v := range input.Vars {
					docs = append(docs, variable{
						commonFields: commonFields{
							Timestamp:      commitTime,
							Type:           []string{"input_variable", "variable"},
							Commit:         commit,
							URL:            sourceURLWithLine(commit, v.FileMetadata),
							Integration:    integ.Manifest.Name,
							Input:          []string{input.Type},
							PolicyTemplate: []string{pt.Name},
							Owner:          integ.Manifest.Owner.Github,
							Attributes:     attributes,
						},
						Var: v,
					})
				}

				pt.Inputs[j].Vars = nil
			}

			var policyTemplateInputs []string
			if pt.Input != "" {
				// Input packages
				policyTemplateInputs = append(policyTemplateInputs, pt.Input)
			} else {
				// Integration packages
				for _, input := range pt.Inputs {
					policyTemplateInputs = append(policyTemplateInputs, input.Type)
				}
			}

			// Policy template variable
			for _, v := range pt.Vars {
				docs = append(docs, variable{
					commonFields: commonFields{
						Timestamp:      commitTime,
						Type:           []string{"policy_template_variable", "variable"},
						Commit:         commit,
						URL:            sourceURLWithLine(commit, v.FileMetadata),
						Integration:    integ.Manifest.Name,
						Input:          policyTemplateInputs,
						PolicyTemplate: []string{pt.Name},
						Owner:          integ.Manifest.Owner.Github,
						Attributes:     attributes,
					},
					Var: v,
				})
			}
			pt.Vars = nil

			// Policy template
			docs = append(docs, policyTemplate{
				commonFields: commonFields{
					Timestamp:      commitTime,
					Type:           []string{"policy_template"},
					Commit:         commit,
					URL:            sourceURL(commit, integ.Manifest.Path()),
					Integration:    integ.Manifest.Name,
					PolicyTemplate: []string{pt.Name},
					Input:          policyTemplateInputs,
					Owner:          integ.Manifest.Owner.Github,
					Attributes:     attributes,
				},
				PolicyTemplate: pt,
			})
		}

		// Manifest
		integ.Manifest.Vars = nil
		integ.Manifest.PolicyTemplates = nil
		docs = append(docs, manifest{
			commonFields: commonFields{
				Timestamp:      commitTime,
				Type:           []string{"manifest"},
				Commit:         commit,
				URL:            sourceURL(commit, integ.Manifest.Path()),
				Integration:    integ.Manifest.Name,
				PolicyTemplate: allPolicyTemplateNames,
				Input:          allPackageInputs,
				Owner:          integ.Manifest.Owner.Github,
				Attributes:     attributes,
			},
			Manifest: integ.Manifest,
		})

		// Build Manifest
		if integ.Build != nil {
			docs = append(docs, buildManifest{
				commonFields: commonFields{
					Timestamp:      commitTime,
					Type:           []string{"build_manifest"},
					Commit:         commit,
					URL:            sourceURL(commit, integ.Build.Path()),
					Integration:    integ.Manifest.Name,
					Input:          allPackageInputs,
					PolicyTemplate: allPolicyTemplateNames,
					Owner:          integ.Manifest.Owner.Github,
					Attributes:     attributes,
				},
				BuildManifest: *integ.Build,
			})
		}

		// Data Streams
		for dsName, ds := range integ.DataStreams {
			for _, stream := range ds.Manifest.Streams {
				for _, streamVar := range stream.Vars {
					// Data Stream Variable
					docs = append(docs, variable{
						commonFields: commonFields{
							Timestamp:   commitTime,
							Type:        []string{"data_stream_variable", "variable"},
							Commit:      commit,
							URL:         sourceURLWithLine(commit, streamVar.FileMetadata),
							Integration: integ.Manifest.Name,
							DataStream:  dsName,
							Input:       []string{stream.Input},
							Owner:       integ.Manifest.Owner.Github,
							Attributes:  attributes,
							// TODO: Set the associated policy_templates.
						},
						Var: streamVar,
					})
				}
			}

			for i := range ds.Manifest.Streams {
				ds.Manifest.Streams[i].Vars = nil
			}

			var allDataStreamInputs []string
			for _, stream := range ds.Manifest.Streams {
				allDataStreamInputs = append(allDataStreamInputs, stream.Input)
			}

			// Data Stream Manifest
			docs = append(docs, dataStreamManifest{
				commonFields: commonFields{
					Timestamp:   commitTime,
					Type:        []string{"data_stream_manifest"},
					Commit:      commit,
					URL:         sourceURL(commit, ds.Manifest.Path()),
					Integration: integ.Manifest.Name,
					DataStream:  dsName,
					Input:       allDataStreamInputs,
					Owner:       integ.Manifest.Owner.Github,
					Attributes:  attributes,
				},
				DataStreamManifest: ds.Manifest,
			})

			// Data Stream Sample Event
			if ds.SampleEvent != nil {
				docs = append(docs, sampleEvent{
					commonFields: commonFields{
						Timestamp:   commitTime,
						Type:        []string{"sample_event"},
						Commit:      commit,
						URL:         sourceURL(commit, ds.SampleEvent.Path()),
						Integration: integ.Manifest.Name,
						DataStream:  dsName,
						Owner:       integ.Manifest.Owner.Github,
						Attributes:  attributes,
					},
					SampleEvent: ds.SampleEvent.Event,
				})
			}

			// TODO: Data stream pipeline

			// Flatten the fields.
			flatFields, err := fleetpkg.FlattenFields(ds.AllFields())
			if err != nil {
				slog.Warn("Failed to flatten fields for integration.", slog.String("integration", integ.Manifest.Name), slog.String("error", err.Error()))
			}

			for _, f := range flatFields {
				docs = append(docs, field{
					commonFields: commonFields{
						Timestamp:   commitTime,
						Type:        []string{"field"},
						Commit:      commit,
						URL:         sourceURLWithLine(commit, f.FileMetadata),
						Integration: integ.Manifest.Name,
						DataStream:  dsName,
						Input:       allDataStreamInputs,
						Owner:       integ.Manifest.Owner.Github,
						Attributes:  attributes,
					},
					Field: f,
				})
			}
		}

		if err := bulkWrite(context.Background(), docs); err != nil {
			slog.Error("Failed to write data to Elasticsearch.", slog.String("integration", integ.Manifest.Name), slog.String("error", err.Error()))
			continue
		}
	}
}

func loadFleetPackages() ([]*fleetpkg.Integration, error) {
	allPackages, err := filepath.Glob(filepath.Join(packagesDir, "*/manifest.yml"))
	if err != nil {
		return nil, err
	}

	rtn := make([]*fleetpkg.Integration, 0, len(allPackages))
	for _, manifestPath := range allPackages {
		integration, err := fleetpkg.Read(filepath.Dir(manifestPath))
		if err != nil {
			return nil, fmt.Errorf("failed reading fleet package from %s: %w", filepath.Dir(manifestPath), err)
		}

		rtn = append(rtn, integration)
	}

	return rtn, nil
}

func bulkWrite(ctx context.Context, docs []any) error {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{elasticsearchURL},
		APIKey:    apiKey,
		Username:  username,
		Password:  password,
		// Retry on 429 TooManyRequests statuses
		RetryOnStatus: []int{502, 503, 504, 429},
		// Retry up to 5 attempts
		MaxRetries: 5,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
				MinVersion:         tls.VersionTLS13,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create ES client: %w", err)
	}

	// Create index with mapping.
	res, err := es.Indices.Exists([]string{index})
	if err != nil {
		return err
	}
	switch res.StatusCode {
	case http.StatusOK:
		slog.Info("Index exists. Not creating.", slog.String("index", index))
	case http.StatusNotFound:
		slog.Info("Creating new index.", slog.String("index", index))
		res, err = es.Indices.Create(index, es.Indices.Create.WithBody(strings.NewReader(indexMapping)))
		if err != nil {
			return err
		}
		if res.IsError() {
			return fmt.Errorf("error creating index: %v", res)
		}
	default:
		return fmt.Errorf("error checking if index exists: %v", res)
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         index,
		Client:        es,               // The Elasticsearch client
		NumWorkers:    1,                // The number of worker goroutines
		FlushBytes:    int(5e6),         // The flush threshold in bytes
		FlushInterval: 30 * time.Second, // The periodic flush interval
	})
	if err != nil {
		return fmt.Errorf("error creating the ES indexer: %w", err)
	}

	// Index docs.
	for _, doc := range docs {
		data, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshall doc [%#v] to JSON: %w", doc, err)
		}

		err = bi.Add(
			ctx,
			esutil.BulkIndexerItem{
				Action: "index",
				// DocumentID: doc.(IDer).ID(),
				Body: bytes.NewReader(data),
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem, err error) {
					slog.Warn("Failed indexing event.", slog.String("event_json", string(data)), slog.String("error", item2.Error.Reason))
				},
			},
		)
		if err != nil {
			return fmt.Errorf("failed indexing docs: %w", err)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return fmt.Errorf("failed to close bulk indexer: %w", err)
	}

	biStats := bi.Stats()
	if biStats.NumFailed > 0 {
		return fmt.Errorf("indexed %d documents, but there were %d errors", int64(biStats.NumFlushed), int64(biStats.NumFailed))
	}

	log.Printf("Successfully indexed [%d] documents.", int64(biStats.NumFlushed))
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

func elasticIntegrationsCommitTimestamp(repoPath, commit string) (time.Time, error) {
	// git show -s --format=%ct <commit>
	cmd := exec.Command("git", "show", "-s", "--format=%ct", commit)
	cmd.Dir = repoPath

	stdout, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	unixSec, err := strconv.ParseInt(string(bytes.TrimSpace(stdout)), 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(unixSec, 0).UTC(), nil
}

func sourceURLWithLine(commit string, meta fleetpkg.FileMetadata) string {
	_, repoPath, _ := strings.Cut(meta.Path(), "integrations/")
	return fmt.Sprintf("https://github.com/elastic/integrations/blob/%v/%v#L%d", commit, repoPath, meta.Line())
}

func sourceURL(commit, path string) string {
	_, repoPath, _ := strings.Cut(path, "integrations/")
	return fmt.Sprintf("https://github.com/elastic/integrations/blob/%v/%v", commit, repoPath)
}

func fileContains(glob string, exactMatch []byte) (bool, error) {
	files, err := filepath.Glob(glob)
	if err != nil {
		return false, err
	}

	for _, path := range files {
		found, err := grepFile(path, exactMatch)
		if err != nil {
			return false, err
		}

		if found {
			return true, nil
		}
	}

	return false, nil
}

func grepFile(file string, exactMatch []byte) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if bytes.Contains(scanner.Bytes(), exactMatch) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
