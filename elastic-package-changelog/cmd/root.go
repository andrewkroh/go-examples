package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const name = "elastic-package-changelog"

func Execute() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		cancel()
	}()

	return ExecuteContext(ctx)
}

func ExecuteContext(ctx context.Context) error {
	logger, err := newLogger()
	if err != nil {
		return nil
	}
	defer logger.Sync()
	ctx = context.WithValue(ctx, loggerKey, logger)

	rootCmd := newRootCommand()
	return rootCmd.ExecuteContext(ctx)
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{Use: name, SilenceUsage: true}

	// Global options.

	// Sub-commands.
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newFlattenCmd())

	// Automatically set flags based on environment variables.
	rootCmd.PersistentFlags().VisitAll(setFlagFromEnv)
	return rootCmd
}

func newLogger() (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	conf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	log, err := conf.Build()
	if err != nil {
		return nil, err
	}
	return log, nil
}

func setFlagFromEnv(flag *pflag.Flag) {
	envVar := strings.ToUpper(flag.Name)
	envVar = strings.ReplaceAll(envVar, "-", "_")
	envVar = "CHANGELOG_" + envVar

	flag.Usage = fmt.Sprintf("%v [env %v]", flag.Usage, envVar)
	if value := os.Getenv(envVar); value != "" {
		flag.Value.Set(value)
	}
}

type contextKey string

var loggerKey = contextKey("logger")

func logger(ctx context.Context) *zap.Logger {
	z, ok := ctx.Value(loggerKey).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}
	return z
}
