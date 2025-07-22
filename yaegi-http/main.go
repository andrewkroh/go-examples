package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-seccomp-bpf"
	"github.com/landlock-lsm/go-landlock/landlock"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var flagSet = flag.NewFlagSet("yaegi-http", flag.ExitOnError)

func init() {
	flagSet.StringVar(&programFile, "prog", "", "program to run (required)")
	flagSet.BoolVar(&restrictStdlib, "restrict", true, "restrict access to stdlib os and io/ioutil packages")
	flagSet.BoolVar(&seccompEnabled, "seccomp", true, "apply Linux sandboxing using seccomp-bpf filtering")
	flagSet.BoolVar(&landlockEnabled, "landlock", true, "apply Linux sandboxing using landlock-lsm")
}

var (
	programFile     string
	restrictStdlib  bool
	seccompEnabled  bool
	landlockEnabled bool
)

var restrictStdlibOnce = sync.OnceFunc(func() {
	syms := []string{
		"path/filepath/filepath",
		"io/ioutil/ioutil",
		"net/net",
		"net/rpc/jsonrpc/jsonrpc",
		"net/rpc/rpc",
		"net/smtp/smtp",
		"net/textproto/textproto",
		"os/os",
		"os/user/user",
		"runtime/debug/debug",
		"runtime/runtime",
	}
	for _, sym := range syms {
		if _, ok := stdlib.Symbols[sym]; !ok {
			panic(fmt.Sprintf("stdlib.Symbols does not contain %q", sym))
		}
		delete(stdlib.Symbols, sym)
	}
})

func main() {
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if programFile == "" {
		flagSet.Usage()
		os.Exit(1)
	}

	p, err := os.ReadFile(programFile)
	if err != nil {
		log.Fatalf("Failed to open program %q: %v", programFile, err)
	}

	if restrictStdlib {
		restrictStdlibOnce()
	}
	if seccompEnabled {
		if err := setupSeccomp(); err != nil {
			log.Fatal(err)
		}
	}
	if landlockEnabled {
		if err := setupLandlock(); err != nil {
			log.Fatal("Failed to sandbox with landlock-lsm", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = withEnvValue(ctx)

	slog.Info("Executing program...")
	start := time.Now()
	events, err := executeProgram(ctx, string(p))
	elapsed := time.Since(start)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("Execution complete", "elapsed", elapsed)

	slog.Info("Emitting events...", "count", len(events))
	for _, event := range events {
		if err := json.NewEncoder(os.Stdout).Encode(event); err != nil {
			log.Fatal(err)
		}
	}
}

// withEnvValue returns a context with the value of all environment variables
// prefixed with YAEGI_HTTP_ as the key and the value as the value.
func withEnvValue(ctx context.Context) context.Context {
	env := map[string]string{}
	for _, v := range os.Environ() {
		after, found := strings.CutPrefix(v, "YAEGI_HTTP_")
		if !found {
			continue
		}

		parts := strings.SplitN(after, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return context.WithValue(ctx, "env", env)
}

func packageName(prog string) (string, error) {
	f, err := parser.ParseFile(&token.FileSet{}, "", prog, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse package name: %w", err)
	}
	return f.Name.Name, nil
}

// ExecuteFunc represents the signature of the Execute function in the plugin.
type ExecuteFunc func(ctx context.Context, c *http.Client, callback func(map[string]any)) error

var executeFuncType = reflect.TypeOf((ExecuteFunc)(nil))

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
	pkgName, err := packageName(prog)
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

// setupLandlock uses the Linux Landlock-LSM to limit filesystem and tcp
// connect permissions of the process.
func setupLandlock() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	err := landlock.V4.Restrict(
		landlock.RODirs(
			"/etc/pki",
			"/etc/ssl").
			IgnoreIfMissing(),
		landlock.ROFiles(
			"/etc/hosts",
			"/etc/resolv.conf",
			"/etc/nsswitch.conf",
			"/etc/localtime").
			IgnoreIfMissing(),
		landlock.ConnectTCP(53),
		landlock.ConnectTCP(443),
	)
	if err != nil {
		return fmt.Errorf("could not setup linux landlock-lsm policy: %w", err)
	}
	slog.Info("Linux Landlock-LSM policy setup complete.")
	return nil
}

func setupSeccomp() error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return nil
	}

	// Create a filter.
	filter := seccomp.Filter{
		NoNewPrivs: true,
		Flag:       seccomp.FilterFlagTSync,
		Policy: seccomp.Policy{
			DefaultAction: seccomp.ActionErrno,
			Syscalls: []seccomp.SyscallGroup{
				{
					Action: seccomp.ActionAllow,
					Names: []string{
						"accept",
						"accept4",
						"bind",
						"brk",
						"clock_gettime",
						"clone",
						"clone3",
						"close",
						"connect",
						"dup",
						"dup3",
						"epoll_create1",
						"epoll_ctl",
						"epoll_pwait",
						"eventfd2",
						"exit_group",
						"exit",
						"faccessat",
						"fcntl",
						"fstat",
						"fstatat",
						"futex",
						"getcwd",
						"getdents64",
						"getpeername",
						"getpid",
						"getrandom",
						"getsockname",
						"getsockopt",
						"gettid",
						"gettimeofday",
						"landlock_add_rule",
						"landlock_create_ruleset",
						"landlock_restrict_self",
						"lseek",
						"mmap",
						"mprotect",
						"munmap",
						"nanosleep",
						"openat",
						"pipe2",
						"ppoll",
						"prctl",
						"read",
						"readlinkat",
						"recvfrom",
						"recvmmsg",
						"recvmsg",
						"rseq",
						"rt_sigaction",
						"rt_sigprocmask",
						"rt_sigreturn",
						"sched_yield",
						"sendmmsg",
						"sendmsg",
						"sendto",
						"set_robust_list",
						"setsockopt",
						"shutdown",
						"sigaltstack",
						"socket",
						"tgkill",
						"tkill",
						"write",
					},
				},
			},
		},
	}

	if err := seccomp.LoadFilter(filter); err != nil {
		return fmt.Errorf("failed to install seccomp-bpf filter: %w", err)
	}
	slog.Info("Linux seccomp-bfp filter installed.")
	return nil
}
