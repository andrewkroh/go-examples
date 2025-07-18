package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

//go:embed testdata/job.go
var program string

// ExecuteFunc represents the signature of the Execute function in the plugin.
type ExecuteFunc func(ctx context.Context, c *http.Client, callback func(map[string]any)) error

var executeFuncType = reflect.TypeOf((ExecuteFunc)(nil))

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.Info("Executing demo program...")
	start := time.Now()
	events, err := executeProgram(ctx, program)
	elapsed := time.Since(start)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Execution complete", "elapsed", elapsed)

	slog.Info("Emitting events...")
	for _, event := range events {
		if err := json.NewEncoder(os.Stdout).Encode(event); err != nil {
			log.Fatal(err)
		}
	}
}

func packageName(prog string) (string, error) {
	f, err := parser.ParseFile(&token.FileSet{}, "", prog, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse package name: %w", err)
	}
	return f.Name.Name, nil
}

func executeProgram(ctx context.Context, prog string) ([]map[string]any, error) {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}

	// Load the program.
	_, err := i.Eval(prog)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate program: %w", err)
	}

	// Get the package name so we can refer to the Execute function.
	pkgName, err := packageName(program)
	if err != nil {
		return nil, err
	}

	// Get reflect.Value representing the Execute function.
	fn, err := i.Eval(pkgName + ".Execute")
	if err != nil {
		return nil, fmt.Errorf("failed to get Execute function from plugin package %q: %w", pkgName, err)
	}

	// Validate the function signature.
	fnType := fn.Type()
	if fnType.Kind() != reflect.Func || !fnType.ConvertibleTo(executeFuncType) {
		return nil, fmt.Errorf("signature of the Execute function does not match expected signature: %s", executeFuncType)
	}
	execute := fn.Convert(executeFuncType).Interface().(ExecuteFunc)

	// Collect events from callback.
	var events []map[string]any
	c := func(e map[string]any) {
		events = append(events, e)
	}

	// Execute the program.
	ctx, cancel := context.WithTimeoutCause(ctx, 10*time.Second, errors.New("execution timeout reached"))
	defer cancel()
	if err = execute(ctx, http.DefaultClient, c); err != nil {
		return nil, fmt.Errorf("program's Execute function returned an error: %w", err)
	}

	return events, nil
}
