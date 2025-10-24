package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
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
	"github.com/segmentio/kafka-go"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var flagSet = flag.NewFlagSet("webhook-handler", flag.ExitOnError)

func init() {
	flagSet.StringVar(&handlerFile, "handler", "", "handler program file (required)")
	flagSet.Int64Var(&maxPayloadSize, "max-payload-size", 10*1024*1024, "maximum payload size in bytes (default: 10MB)")
	flagSet.StringVar(&listenAddr, "listen", ":8080", "HTTP server listen address")
	flagSet.StringVar(&kafkaBrokers, "kafka-brokers", "localhost:9092", "comma-separated list of Kafka broker addresses")
	flagSet.StringVar(&kafkaTopic, "kafka-topic", "webhooks", "Kafka topic to publish events to")
	flagSet.BoolVar(&restrictStdlib, "restrict", true, "restrict access to stdlib os and io/ioutil packages")
	flagSet.BoolVar(&seccompEnabled, "seccomp", false, "apply Linux sandboxing using seccomp-bpf filtering")
	flagSet.BoolVar(&landlockFSEnabled, "landlock-fs", false, "apply Linux filesystem sandboxing using landlock-lsm")
	flagSet.BoolVar(&landlockNetEnabled, "landlock-net", false, "apply Linux network sandboxing using landlock-lsm")
}

var (
	handlerFile        string
	maxPayloadSize     int64
	listenAddr         string
	kafkaBrokers       string
	kafkaTopic         string
	restrictStdlib     bool
	seccompEnabled     bool
	landlockFSEnabled  bool
	landlockNetEnabled bool
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

	if handlerFile == "" {
		flagSet.Usage()
		os.Exit(1)
	}

	// Read handler program
	handlerCode, err := os.ReadFile(handlerFile)
	if err != nil {
		log.Fatalf("Failed to read handler file %q: %v", handlerFile, err)
	}

	// Apply security restrictions
	if restrictStdlib {
		restrictStdlibOnce()
	}
	if seccompEnabled {
		if err := setupSeccomp(); err != nil {
			log.Fatal(err)
		}
	}
	if landlockFSEnabled || landlockNetEnabled {
		if err := setupLandlock(); err != nil {
			log.Fatal("Failed to sandbox with landlock-lsm", err)
		}
	}

	// Setup Kafka writer
	brokers := strings.Split(kafkaBrokers, ",")
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        kafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		Compression:  kafka.Snappy,
	}
	defer writer.Close()

	// Create webhook handler
	h := &webhookHandler{
		handlerCode: string(handlerCode),
		writer:      writer,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", h.handleWebhook)
	mux.HandleFunc("GET /health", handleHealth)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		slog.Info("Starting webhook handler server", "addr", listenAddr, "topic", kafkaTopic)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	slog.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}

	slog.Info("Server stopped")
}

type webhookHandler struct {
	handlerCode string
	writer      *kafka.Writer
}

func (h *webhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Enforce payload size limit
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)
	defer r.Body.Close()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "Payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	// Execute handler
	ctx := r.Context()
	ctx = withEnvValue(ctx)

	start := time.Now()
	err = h.executeHandler(ctx, payload)
	elapsed := time.Since(start)

	if err != nil {
		slog.Error("Handler execution failed", "error", err, "elapsed", elapsed)
		http.Error(w, "Handler execution failed", http.StatusInternalServerError)
		return
	}

	slog.Info("Handler executed successfully", "elapsed", elapsed, "payload_size", len(payload))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

// ProcessFunc represents the signature of the Process function in the handler.
type ProcessFunc func(ctx context.Context, payload []byte, publish func([]byte) error) error

var processFuncType = reflect.TypeOf((ProcessFunc)(nil))

func (h *webhookHandler) executeHandler(ctx context.Context, payload []byte) error {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return err
	}

	// Load the handler program
	_, err := i.Eval(h.handlerCode)
	if err != nil {
		return fmt.Errorf("failed to evaluate handler: %w", err)
	}

	// Get the package name
	pkgName, err := packageName(h.handlerCode)
	if err != nil {
		return err
	}

	// Get reflect.Value representing the Process function
	fn, err := i.Eval(pkgName + ".Process")
	if err != nil {
		return fmt.Errorf("failed to get Process function from handler package %q: %w", pkgName, err)
	}

	// Validate the function signature
	fnType := fn.Type()
	if fnType.Kind() != reflect.Func || !fnType.ConvertibleTo(processFuncType) {
		return fmt.Errorf("signature of the Process function does not match expected signature: %s", processFuncType)
	}
	process := fn.Convert(processFuncType).Interface().(ProcessFunc)

	// Create publish function
	publishFunc := func(event []byte) error {
		msg := kafka.Message{
			Value: event,
			Time:  time.Now(),
		}
		return h.writer.WriteMessages(ctx, msg)
	}

	// Execute the handler with timeout
	ctx, cancel := context.WithTimeoutCause(ctx, 30*time.Second, errors.New("handler execution timeout reached"))
	defer cancel()

	return process(ctx, payload, publishFunc)
}

func packageName(prog string) (string, error) {
	f, err := parser.ParseFile(&token.FileSet{}, "", prog, parser.PackageClauseOnly)
	if err != nil {
		return "", fmt.Errorf("failed to parse package name: %w", err)
	}
	return f.Name.Name, nil
}

// withEnvValue returns a context with the value of all environment variables
// prefixed with WEBHOOK_ as the key and the value as the value.
func withEnvValue(ctx context.Context) context.Context {
	env := map[string]string{}
	for _, v := range os.Environ() {
		after, found := strings.CutPrefix(v, "WEBHOOK_")
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

// setupLandlock uses the Linux Landlock-LSM to limit filesystem and tcp
// connect permissions of the process.
func setupLandlock() error {
	if runtime.GOOS != "linux" {
		return nil
	}

	if landlockFSEnabled {
		err := landlock.V4.RestrictPaths(
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
		)
		if err != nil {
			return fmt.Errorf("could not setup linux landlock-lsm filesystem policy: %w", err)
		}
	}
	if landlockNetEnabled {
		err := landlock.V4.RestrictNet(
			landlock.ConnectTCP(53),
			landlock.ConnectTCP(443),
			landlock.ConnectTCP(9092),
		)
		if err != nil {
			return fmt.Errorf("could not setup linux landlock-lsm network policy: %w", err)
		}
	}
	slog.Info("Linux Landlock-LSM policy setup complete.")
	return nil
}

func setupSeccomp() error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		return nil
	}

	// Create a filter
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
						"listen",
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
