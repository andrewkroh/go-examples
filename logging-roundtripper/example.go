package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/andrewkroh/go-examples/logging-roundtripper/httplog"
)

func main() {
	// Create a Zap logger the writes trace data to a file.
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "trace.ndjson",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		zap.DebugLevel,
	)
	traceLogger := zap.New(core)

	// Wrap the default transport in a LoggingRoundTripper.
	client := http.Client{
		Transport: httplog.NewLoggingRoundTripper(http.DefaultTransport, traceLogger),
	}

	// Associate a trace.id with logged requests and responses.
	ctx := context.WithValue(context.Background(), httplog.TraceIDKey, "my-trace-id")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org?format=json", nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response [status=%d]:\n%s\n", resp.StatusCode, body)
}
