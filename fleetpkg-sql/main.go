package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/andrewkroh/go-fleetpkg"

	"github.com/andrewkroh/go-examples/fleetpkg-sql/schema"

	// Register SQLite driver.
	_ "modernc.org/sqlite"
)

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate

//go:embed schema.sql
var ddl string

var (
	packagesDir string
)

func init() {
	flag.StringVar(&packagesDir, "packages-dir", "", "Directory containing Fleet packages.")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintln(os.Stdout, "    fleetpkg-sql [flags]")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if packagesDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(); err != nil {
		slog.Error("Failed", "error", err)
	}
}

func run() error {
	integrations, err := loadFleetPackages()
	if err != nil {
		return fmt.Errorf("failed to load integrations: %w", err)
	}

	db, err := sql.Open("sqlite", "fleet.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := createSchema(db); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	for _, integ := range integrations {
		if err := insertIntegration(db, integ); err != nil {
			return fmt.Errorf("failed to insert integration for %s: %w", integ.Manifest.Name, err)
		}
	}

	return nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(ddl)
	return err
}

func insertIntegration(db *sql.DB, integ *fleetpkg.Integration) error {
	m := &integ.Manifest
	integPath := integ.Path()

	q := schema.New(db)
	ctx := context.Background()

	dirName := filepath.Base(integPath)

	err := q.InsertManifest(ctx, schema.InsertManifestParams{
		DirName:             dirName,
		Name:                sql.NullString{String: m.Name, Valid: m.Name != ""},
		Title:               sql.NullString{String: m.Title, Valid: m.Title != ""},
		Version:             sql.NullString{String: m.Version, Valid: m.Version != ""},
		Release:             sql.NullString{String: m.Release, Valid: m.Release != ""},
		Description:         sql.NullString{String: m.Description, Valid: m.Description != ""},
		Type:                sql.NullString{String: m.Type, Valid: m.Type != ""},
		FormatVersion:       sql.NullString{String: m.FormatVersion, Valid: m.FormatVersion != ""},
		License:             sql.NullString{String: m.License, Valid: m.License != ""},
		KibanaVersion:       sql.NullString{String: m.Conditions.Kibana.Version, Valid: m.Conditions.Kibana.Version != ""},
		ElasticSubscription: sql.NullString{String: m.Conditions.Elastic.Subscription, Valid: m.Conditions.Elastic.Subscription != ""},
		SourceLicense:       sql.NullString{String: m.Source.License, Valid: m.Source.License != ""},
		OwnerGithub:         sql.NullString{String: m.Owner.Github, Valid: m.Owner.Github != ""},
		OwnerType:           sql.NullString{String: m.Owner.Type, Valid: m.Owner.Type != ""},
	})
	if err != nil {
		return err
	}

	for _, icon := range m.Icons {
		err = q.InsertManifestIcon(ctx, schema.InsertManifestIconParams{
			ManifestDirName: sql.NullString{String: dirName, Valid: true},
			Src:             sql.NullString{String: icon.Src, Valid: icon.Src != ""},
			Title:           sql.NullString{String: icon.Title, Valid: icon.Title != ""},
			Size:            sql.NullString{String: icon.Size, Valid: icon.Size != ""},
			Type:            sql.NullString{String: icon.Type, Valid: icon.Type != ""},
			DarkMode:        sqlBool(icon.DarkMode),
		})
		if err != nil {
			return err
		}
	}

	for _, category := range m.Categories {
		err = q.InsertManifestCategory(ctx, schema.InsertManifestCategoryParams{
			ManifestDirName: sql.NullString{String: dirName, Valid: true},
			Category:        sql.NullString{String: category, Valid: category != ""},
		})
		if err != nil {
			return err
		}
	}

	for _, screenshot := range m.Screenshots {
		err = q.InsertManifestScreenshot(ctx, schema.InsertManifestScreenshotParams{
			ManifestDirName: sql.NullString{String: dirName, Valid: true},
			Src:             sql.NullString{String: screenshot.Src, Valid: screenshot.Src != ""},
			Title:           sql.NullString{String: screenshot.Title, Valid: screenshot.Title != ""},
			Size:            sql.NullString{String: screenshot.Size, Valid: screenshot.Size != ""},
			Type:            sql.NullString{String: screenshot.Type, Valid: screenshot.Type != ""},
		})
		if err != nil {
			return err
		}
	}

	for _, v := range m.Vars {
		defaultValue, _ := json.Marshal(v.Default)
		err = q.InsertManifestVar(ctx, schema.InsertManifestVarParams{
			ManifestDirName: sql.NullString{String: dirName, Valid: true},
			Name:            sql.NullString{String: v.Name, Valid: v.Name != ""},
			DefaultValue:    sql.NullString{String: string(defaultValue), Valid: len(defaultValue) > 0},
			Description:     sql.NullString{String: v.Description, Valid: v.Description != ""},
			Type:            sql.NullString{String: v.Type, Valid: v.Type != ""},
			Title:           sql.NullString{String: v.Title, Valid: v.Title != ""},
			Multi:           sqlBool(v.Multi),
			Required:        sqlBool(v.Required),
			Secret:          sqlBool(v.Secret),
			ShowUser:        sqlBool(v.ShowUser),
		})
		if err != nil {
			return err
		}

		for _, opt := range v.Options {
			err = q.InsertManifestVarOption(ctx, schema.InsertManifestVarOptionParams{
				ManifestDirName: sql.NullString{String: dirName, Valid: true},
				VarName:         sql.NullString{String: v.Name, Valid: v.Name != ""},
				Value:           sql.NullString{String: opt.Value, Valid: opt.Value != ""},
				Text:            sql.NullString{String: opt.Text, Valid: opt.Text != ""},
			})
			if err != nil {
				return err
			}
		}
	}

	for _, pt := range m.PolicyTemplates {
		err = q.InsertManifestPolicyTemplate(ctx, schema.InsertManifestPolicyTemplateParams{
			ManifestDirName: sql.NullString{String: dirName, Valid: true},
			Name:            sql.NullString{String: pt.Name, Valid: pt.Name != ""},
			Title:           sql.NullString{String: pt.Title, Valid: pt.Title != ""},
			Description:     sql.NullString{String: pt.Description, Valid: pt.Description != ""},
		})
		if err != nil {
			return err
		}

		for _, category := range pt.Categories {
			err = q.InsertManifestPolicyTemplateCategory(ctx, schema.InsertManifestPolicyTemplateCategoryParams{
				ManifestDirName:    sql.NullString{String: dirName, Valid: true},
				PolicyTemplateName: sql.NullString{String: pt.Name, Valid: pt.Name != ""},
				Category:           sql.NullString{String: category, Valid: category != ""},
			})
			if err != nil {
				return err
			}
		}

		for _, ds := range pt.DataStreams {
			err = q.InsertManifestPolicyTemplateDataStream(ctx, schema.InsertManifestPolicyTemplateDataStreamParams{
				ManifestDirName:    sql.NullString{String: dirName, Valid: true},
				PolicyTemplateName: sql.NullString{String: pt.Name, Valid: pt.Name != ""},
				DataStream:         sql.NullString{String: ds, Valid: ds != ""},
			})
			if err != nil {
				return err
			}
		}

		for _, icon := range pt.Icons {
			err = q.InsertManifestPolicyTemplateIcon(ctx, schema.InsertManifestPolicyTemplateIconParams{
				ManifestDirName:    sql.NullString{String: dirName, Valid: true},
				PolicyTemplateName: sql.NullString{String: pt.Name, Valid: pt.Name != ""},
				Src:                sql.NullString{String: icon.Src, Valid: icon.Src != ""},
				Title:              sql.NullString{String: icon.Title, Valid: icon.Title != ""},
				Size:               sql.NullString{String: icon.Size, Valid: icon.Size != ""},
				Type:               sql.NullString{String: icon.Type, Valid: icon.Type != ""},
				DarkMode:           sqlBool(icon.DarkMode),
			})
			if err != nil {
				return err
			}
		}

		for _, input := range pt.Inputs {
			err = q.InsertManifestPolicyTemplateInput(ctx, schema.InsertManifestPolicyTemplateInputParams{
				ManifestDirName:    sql.NullString{String: dirName, Valid: true},
				PolicyTemplateName: sql.NullString{String: pt.Name, Valid: pt.Name != ""},
				Type:               sql.NullString{String: input.Type, Valid: input.Type != ""},
				Title:              sql.NullString{String: input.Title, Valid: input.Title != ""},
				Description:        sql.NullString{String: input.Description, Valid: input.Description != ""},
				InputGroup:         sql.NullString{String: input.InputGroup, Valid: input.InputGroup != ""},
				TemplatePath:       sql.NullString{String: input.TemplatePath, Valid: input.TemplatePath != ""},
				Multi:              sqlBool(input.Multi),
			})
			if err != nil {
				return err
			}

			for _, v := range input.Vars {
				defaultValue, _ := json.Marshal(v.Default)
				err = q.InsertManifestPolicyTemplateInputVar(ctx, schema.InsertManifestPolicyTemplateInputVarParams{
					ManifestDirName:    sql.NullString{String: dirName, Valid: true},
					PolicyTemplateName: sql.NullString{String: pt.Name, Valid: pt.Name != ""},
					InputTitle:         sql.NullString{String: input.Title, Valid: input.Title != ""},
					Name:               sql.NullString{String: v.Name, Valid: v.Name != ""},
					DefaultValue:       sql.NullString{String: string(defaultValue), Valid: len(defaultValue) > 0},
					Description:        sql.NullString{String: v.Description, Valid: v.Description != ""},
					Type:               sql.NullString{String: v.Type, Valid: v.Type != ""},
					Title:              sql.NullString{String: v.Title, Valid: v.Title != ""},
					Multi:              sqlBool(v.Multi),
					Required:           sqlBool(v.Required),
					Secret:             sqlBool(v.Secret),
					ShowUser:           sqlBool(v.ShowUser),
				})
				if err != nil {
					return err
				}
			}
		}
	}

	if integ.Build != nil {
		err = q.InsertBuildManifest(ctx, schema.InsertBuildManifestParams{
			ManifestDirName:   dirName,
			EcsReference:      sql.NullString{String: integ.Build.Dependencies.ECS.Reference, Valid: integ.Build.Dependencies.ECS.Reference != ""},
			EcsImportMappings: sqlBool(integ.Build.Dependencies.ECS.ImportMappings),
		})
		if err != nil {
			return err
		}
	}

	for dsName, ds := range integ.DataStreams {
		err = q.InsertDataStream(ctx, schema.InsertDataStreamParams{
			ManifestDirName: dirName,
			DataStreamName:  dsName,
			Dataset:         sql.NullString{String: ds.Manifest.Dataset, Valid: ds.Manifest.Dataset != ""},
			DatasetIsPrefix: sqlBool(ds.Manifest.DatasetIsPrefix),
			IlmPolicy:       sql.NullString{String: ds.Manifest.ILMPolicy, Valid: ds.Manifest.ILMPolicy != ""},
			Release:         sql.NullString{String: ds.Manifest.Release, Valid: ds.Manifest.Release != ""},
			Title:           sql.NullString{String: ds.Manifest.Title, Valid: ds.Manifest.Title != ""},
			Type:            sql.NullString{String: ds.Manifest.Type, Valid: ds.Manifest.Type != ""},
		})
		if err != nil {
			return err
		}

		for _, s := range ds.Manifest.Streams {
			err = q.InsertStream(ctx, schema.InsertStreamParams{
				ManifestDirName: dirName,
				DataStreamName:  dsName,
				Title:           sql.NullString{String: s.Title, Valid: s.Title != ""},
				Input:           sql.NullString{String: s.Input, Valid: s.Input != ""},
				Description:     sql.NullString{String: s.Description, Valid: s.Description != ""},
				TemplatePath:    sql.NullString{String: s.TemplatePath, Valid: s.TemplatePath != ""},
				Enabled:         sqlBool(s.Enabled),
			})
			if err != nil {
				return err
			}
		}

		for pName, p := range ds.Pipelines {
			var version sql.NullInt32
			if p.Version != nil {
				version.Int32 = int32(*p.Version)
				version.Valid = true
			}

			err = q.InsertIngestPipeline(ctx, schema.InsertIngestPipelineParams{
				ManifestDirName: dirName,
				DataStreamName:  dsName,
				PipelineName:    pName,
				Description:     sql.NullString{String: p.Description, Valid: p.Description != ""},
				Version:         sqlInt64(p.Version),
			})
			if err != nil {
				return err
			}

			for i, proc := range p.Processors {
				attrs, _ := json.Marshal(proc.Attributes)
				err = q.InsertProcessor(ctx, schema.InsertProcessorParams{
					ManifestDirName: dirName,
					DataStreamName:  dsName,
					PipelineName:    sqlString(&pName),
					ProcessorOrder:  sqlInt64(&i),
					Type:            sql.NullString{String: proc.Type, Valid: proc.Type != ""},
					Attributes:      attrs,
					OnFailure:       sql.NullBool{Bool: false, Valid: true},
				})
				if err != nil {
					return err
				}
			}

			for i, proc := range p.OnFailure {
				attrs, _ := json.Marshal(proc.Attributes)
				err = q.InsertProcessor(ctx, schema.InsertProcessorParams{
					ManifestDirName: dirName,
					DataStreamName:  dsName,
					PipelineName:    sqlString(&pName),
					ProcessorOrder:  sqlInt64(&i),
					Type:            sql.NullString{String: proc.Type, Valid: proc.Type != ""},
					Attributes:      attrs,
					OnFailure:       sql.NullBool{Bool: true, Valid: true},
				})
				if err != nil {
					return err
				}
			}
		}
	}

	for _, entry := range integ.Changelog.Releases {
		for _, change := range entry.Changes {
			err = q.InsertChangelog(ctx, schema.InsertChangelogParams{
				ManifestDirName: dirName,
				Version:         sql.NullString{String: entry.Version, Valid: entry.Version != ""},
				Description:     sql.NullString{String: change.Description, Valid: change.Description != ""},
				Type:            sql.NullString{String: change.Type, Valid: change.Type != ""},
				Link:            sql.NullString{String: change.Link, Valid: change.Link != ""},
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
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

func sqlBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{
		Bool:  *b,
		Valid: true,
	}
}

func sqlString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{
		String: *s,
		Valid:  true,
	}
}

func sqlInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{
		Int64: int64(*i),
		Valid: true,
	}
}
