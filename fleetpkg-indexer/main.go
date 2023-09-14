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
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/andrewkroh/go-fleetpkg"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"golang.org/x/exp/maps"
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

func usage() {
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "    fleetpkg-indexer [flags]")
	flag.PrintDefaults()
}

type jsonTime time.Time

func (t jsonTime) MarshalJSON() ([]byte, error) {
	v := time.Time(t).UTC().Format(time.RFC3339Nano)
	return json.Marshal(v)
}

type commonFields struct {
	Timestamp      jsonTime   `json:"@timestamp"`
	Type           []string   `json:"@type"`
	Commit         string     `json:"@commit"`
	URL            string     `json:"@url,omitempty"`
	Integration    string     `json:"@integration"`
	DataStream     []string   `json:"@data_stream,omitempty"`
	Input          []string   `json:"@input,omitempty"`
	PolicyTemplate []string   `json:"@policy_template,omitempty"`
	Owner          string     `json:"@owner"`
	Attributes     attributes `json:"@attributes,omitempty"`
}

type attributes struct {
	Deprecated bool `json:"deprecated"`
	RSA2ELK    bool `json:"rsa2elk"`
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

func main() {
	flag.Parse()

	if packagesDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	integrations, err := loadFleetPackages()
	if err != nil {
		slog.Error("Failed to load integrations.",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	commit, err := elasticIntegrationsCommit(packagesDir)
	if err != nil {
		commit = "unknown"
	}

	ts, _ := elasticIntegrationsCommitTimestamp(packagesDir, commit)
	commitTime := jsonTime(ts)

	bi, err := bulkIndexer()
	if err != nil {
		slog.Error("Failed to setup ES client.",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()
	addBulkDoc := func(doc any) {
		data, err := json.Marshal(doc)
		if err != nil {
			slog.Error("Failed marshal document to JSON.",
				slog.String("error", err.Error()))
			os.Exit(1)
		}

		err = bi.Add(
			ctx,
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   bytes.NewReader(data),
				OnFailure: func(ctx context.Context, _ esutil.BulkIndexerItem, item esutil.BulkIndexerResponseItem, err error) {
					slog.Warn("Failed indexing event.",
						slog.String("event_json", string(data)),
						slog.String("error", item.Error.Reason))
				},
			},
		)
		if err != nil {
			slog.Error("Failed indexing document.",
				slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

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
		allDataStreams := maps.Keys(integ.DataStreams)

		rsa2elk, err := fileContains(filepath.Join(integ.Path(), "data_stream/*/agent/stream/*.hbs"), []byte("nwparser"))
		if err != nil {
			slog.Warn("Failed to determine if package is rsa2elk",
				slog.String("integration", integ.Manifest.Name),
				slog.String("error", err.Error()))
		}
		attrs := attributes{
			Deprecated: strings.Contains(strings.ToLower(integ.Manifest.Description), "deprecated"),
			RSA2ELK:    rsa2elk,
		}

		makeCommonFields := func(types, policyTemplates, dataStreams, inputs []string, url string) commonFields {
			return commonFields{
				Timestamp:      commitTime,
				Commit:         commit,
				Owner:          integ.Manifest.Owner.Github,
				Integration:    integ.Manifest.Name,
				Attributes:     attrs,
				Type:           types,
				PolicyTemplate: policyTemplates,
				DataStream:     dataStreams,
				Input:          inputs,
				URL:            url,
			}
		}

		// Root-level variables
		for _, v := range integ.Manifest.Vars {
			addBulkDoc(variable{
				commonFields: makeCommonFields(
					[]string{"package_variable", "variable"},
					allPolicyTemplateNames,
					allDataStreams,
					allPackageInputs,
					sourceURLWithLine(commit, v.FileMetadata),
				),
				Var: v,
			})
		}

		// Build association of data streams to policy templates.
		dataStreamToPolicyTemplates := map[string][]string{}
		for _, ds := range maps.Keys(integ.DataStreams) {
			var pts []string
			for _, pt := range integ.Manifest.PolicyTemplates {
				// An empty 'data_streams' list implies all data streams (empirically determined).
				if len(pt.DataStreams) == 0 || slices.Contains(pt.DataStreams, ds) {
					pts = append(pts, pt.Name)
				}
			}
			dataStreamToPolicyTemplates[ds] = pts
		}

		// Policy templates
		for _, pt := range integ.Manifest.PolicyTemplates {
			policyTemplateDataStreams := pt.DataStreams
			if len(policyTemplateDataStreams) == 0 {
				policyTemplateDataStreams = allDataStreams
			}

			// Policy template input variables
			for j, input := range pt.Inputs {
				for _, v := range input.Vars {
					addBulkDoc(variable{
						commonFields: makeCommonFields(
							[]string{"input_variable", "variable"},
							[]string{pt.Name},
							policyTemplateDataStreams,
							[]string{input.Type},
							sourceURLWithLine(commit, v.FileMetadata),
						),
						Var: v,
					})
				}

				// Clear variables since those are being indexed as separate documents.
				pt.Inputs[j].Vars = nil
			}

			// Determine all inputs associated with the policy template.
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

			// Policy template variables
			for _, v := range pt.Vars {
				addBulkDoc(variable{
					commonFields: makeCommonFields(
						[]string{"policy_template_variable", "variable"},
						[]string{pt.Name},
						policyTemplateDataStreams,
						policyTemplateInputs,
						sourceURLWithLine(commit, v.FileMetadata),
					),
					Var: v,
				})
			}
			// Clear variables since those are being indexed as separate documents.
			pt.Vars = nil

			// Policy template
			addBulkDoc(policyTemplate{
				commonFields: makeCommonFields(
					[]string{"policy_template"},
					[]string{pt.Name},
					policyTemplateDataStreams,
					policyTemplateInputs,
					sourceURL(commit, integ.Manifest.Path()),
				),
				PolicyTemplate: pt,
			})
		}

		// Manifest
		integ.Manifest.Vars = nil
		integ.Manifest.PolicyTemplates = nil
		addBulkDoc(manifest{
			commonFields: makeCommonFields(
				[]string{"manifest"},
				allPolicyTemplateNames,
				allDataStreams,
				allPackageInputs,
				sourceURL(commit, integ.Manifest.Path()),
			),
			Manifest: integ.Manifest,
		})

		// Build Manifest
		if integ.Build != nil {
			addBulkDoc(buildManifest{
				commonFields: makeCommonFields(
					[]string{"build_manifest"},
					allPolicyTemplateNames,
					allDataStreams,
					allPackageInputs,
					sourceURL(commit, integ.Build.Path()),
				),
				BuildManifest: *integ.Build,
			})
		}

		// Data Streams
		for dsName, ds := range integ.DataStreams {
			for i, stream := range ds.Manifest.Streams {
				for _, streamVar := range stream.Vars {
					// Data Stream Variable
					addBulkDoc(variable{
						commonFields: makeCommonFields(
							[]string{"data_stream_variable", "variable"},
							dataStreamToPolicyTemplates[dsName],
							[]string{dsName},
							[]string{stream.Input},
							sourceURLWithLine(commit, streamVar.FileMetadata),
						),
						Var: streamVar,
					})
				}

				// Clear variables because they are indexed separately.
				ds.Manifest.Streams[i].Vars = nil
			}

			var allDataStreamInputs []string
			for _, stream := range ds.Manifest.Streams {
				allDataStreamInputs = append(allDataStreamInputs, stream.Input)
			}

			// Data Stream Manifest
			addBulkDoc(dataStreamManifest{
				commonFields: makeCommonFields(
					[]string{"data_stream_manifest"},
					dataStreamToPolicyTemplates[dsName],
					[]string{dsName},
					allDataStreamInputs,
					sourceURL(commit, ds.Manifest.Path()),
				),
				DataStreamManifest: ds.Manifest,
			})

			// Data Stream Sample Event
			if ds.SampleEvent != nil {
				addBulkDoc(sampleEvent{
					commonFields: makeCommonFields(
						[]string{"sample_event"},
						dataStreamToPolicyTemplates[dsName],
						[]string{dsName},
						allDataStreamInputs,
						sourceURL(commit, ds.SampleEvent.Path()),
					),
					SampleEvent: ds.SampleEvent.Event,
				})
			}

			// TODO: Data stream pipeline

			// Flatten the fields.
			flatFields, err := fleetpkg.FlattenFields(ds.AllFields())
			if err != nil {
				slog.Warn("Failed to flatten fields for integration.",
					slog.String("integration", integ.Manifest.Name),
					slog.String("error", err.Error()))
			}

			for _, f := range flatFields {
				addBulkDoc(field{
					commonFields: makeCommonFields(
						[]string{"field"},
						dataStreamToPolicyTemplates[dsName],
						[]string{dsName},
						allDataStreamInputs,
						sourceURLWithLine(commit, f.FileMetadata),
					),
					Field: f,
				})
			}
		}
	}

	if err = bi.Close(ctx); err != nil {
		slog.Error("Failed to write data to Elasticsearch.",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	biStats := bi.Stats()
	if biStats.NumFailed > 0 {
		slog.Warn("Indexed documents but there were errors.",
			slog.Uint64("flushed", biStats.NumFlushed),
			slog.Uint64("failed", biStats.NumFlushed))
		os.Exit(1)
	}

	slog.Info("Successfully indexed data to ES.",
		slog.Uint64("flushed", biStats.NumFlushed))
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

func bulkIndexer() (esutil.BulkIndexer, error) {
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
		return nil, fmt.Errorf("failed to create ES client: %w", err)
	}

	// Create index with mapping.
	res, err := es.Indices.Exists([]string{index})
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case http.StatusOK:
		slog.Info("Index exists. Not creating.", slog.String("index", index))
	case http.StatusNotFound:
		slog.Info("Creating new index.", slog.String("index", index))
		res, err = es.Indices.Create(index, es.Indices.Create.WithBody(strings.NewReader(indexMapping)))
		if err != nil {
			return nil, err
		}
		if res.IsError() {
			return nil, fmt.Errorf("error creating index: %v", res)
		}
	default:
		return nil, fmt.Errorf("error checking if index exists: %v", res)
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         index,
		Client:        es,               // The Elasticsearch client
		NumWorkers:    1,                // The number of worker goroutines
		FlushBytes:    int(5e6),         // The flush threshold in bytes
		FlushInterval: 30 * time.Second, // The periodic flush interval
	})
	if err != nil {
		return nil, fmt.Errorf("error creating the ES indexer: %w", err)
	}

	return bi, nil
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
