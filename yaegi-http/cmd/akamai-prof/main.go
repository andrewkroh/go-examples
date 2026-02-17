// Command akamai-prof runs an interpreted Yaegi program and records pprof data.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"runtime"
	"runtime/pprof"
)

// ExecuteFunc represents the interpreted program entrypoint signature.
type ExecuteFunc func(ctx context.Context, c *http.Client, callback func(map[string]any)) error

var executeFuncType = reflect.TypeOf((ExecuteFunc)(nil))

var (
	programPath        string
	runDuration        time.Duration
	cpuProfileDuration time.Duration
	outDir             string
)

func init() {
	flag.StringVar(&programPath, "prog", "testdata/programs/akamai.go", "path to interpreted program")
	flag.DurationVar(&runDuration, "duration", 2*time.Minute, "total run duration")
	flag.DurationVar(&cpuProfileDuration, "cpu-profile-duration", 30*time.Second, "CPU profile duration")
	flag.StringVar(&outDir, "out", "profiles", "directory where pprof files are written")
}

func main() {
	flag.Parse()

	if runDuration <= 0 {
		log.Fatal("-duration must be greater than zero")
	}
	if cpuProfileDuration <= 0 {
		log.Fatal("-cpu-profile-duration must be greater than zero")
	}
	if runDuration <= cpuProfileDuration {
		log.Fatal("-duration must be greater than -cpu-profile-duration")
	}

	progBytes, err := os.ReadFile(programPath)
	if err != nil {
		log.Fatalf("failed to read program %q: %v", programPath, err)
	}
	prog := string(progBytes)

	execFn, err := loadExecuteFunc(prog)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory %q: %v", outDir, err)
	}

	env := loadProgramEnv()
	// Keep interpreter loop bounded and let it stop before context deadline
	// so active response reads can finish cleanly.
	if _, ok := env["POLL_TIMEOUT"]; !ok {
		env["POLL_TIMEOUT"] = defaultPollTimeout(runDuration).String()
	}

	ctx, cancel := context.WithTimeout(context.Background(), runDuration)
	defer cancel()
	ctx = context.WithValue(ctx, "env", env)

	start := time.Now()
	var eventCount int64

	profileDone := make(chan error, 1)
	go func() {
		profileDone <- captureProfiles(ctx, runDuration, cpuProfileDuration, outDir)
	}()

	err = execFn(ctx, http.DefaultClient, func(map[string]any) {
		atomic.AddInt64(&eventCount, 1)
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		log.Fatalf("program execution failed: %v", err)
	}

	if profileErr := <-profileDone; profileErr != nil {
		log.Fatalf("profiling failed: %v", profileErr)
	}

	elapsed := time.Since(start)
	count := atomic.LoadInt64(&eventCount)
	rate := float64(count) / elapsed.Seconds()
	log.Printf("done elapsed=%s events=%d eps=%.2f out=%s", elapsed.Round(time.Millisecond), count, rate, outDir)
}

func loadExecuteFunc(prog string) (ExecuteFunc, error) {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}

	if _, err := i.Eval(prog); err != nil {
		return nil, fmt.Errorf("failed to evaluate program: %w", err)
	}

	pkgName, err := packageName(prog)
	if err != nil {
		return nil, err
	}

	fn, err := i.Eval(pkgName + ".Execute")
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate %q: %w", pkgName+".Execute", err)
	}

	if fn.Type().Kind() != reflect.Func || !fn.Type().ConvertibleTo(executeFuncType) {
		return nil, fmt.Errorf("Execute signature must be %s", executeFuncType)
	}
	return fn.Convert(executeFuncType).Interface().(ExecuteFunc), nil
}

func captureProfiles(ctx context.Context, total, cpuDur time.Duration, out string) error {
	midpoint := total / 2
	timer := time.NewTimer(midpoint)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return writeHeapProfile(filepath.Join(out, "heap_end.pprof"))
	case <-timer.C:
	}

	if err := writeHeapProfile(filepath.Join(out, "heap_mid.pprof")); err != nil {
		return err
	}
	if err := writeCPUProfile(filepath.Join(out, "cpu_mid.pprof"), cpuDur); err != nil {
		return err
	}
	return writeHeapProfile(filepath.Join(out, "heap_end.pprof"))
}

func writeCPUProfile(path string, d time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile %q: %w", path, err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	time.Sleep(d)
	pprof.StopCPUProfile()
	return nil
}

func writeHeapProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create heap profile %q: %w", path, err)
	}
	defer f.Close()

	// Force GC for a deterministic heap snapshot point.
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}
	return nil
}

func packageName(prog string) (string, error) {
	f, err := parser.ParseFile(&token.FileSet{}, "", prog, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse package name: %w", err)
	}
	return f.Name.Name, nil
}

func loadProgramEnv() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		after, found := strings.CutPrefix(kv, "YAEGI_HTTP_")
		if !found {
			continue
		}
		parts := strings.SplitN(after, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func defaultPollTimeout(runDuration time.Duration) time.Duration {
	if runDuration <= 2*time.Second {
		return runDuration
	}
	if runDuration <= 20*time.Second {
		return runDuration - 1*time.Second
	}
	return runDuration - 5*time.Second
}
