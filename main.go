package main

import (
	"context"
	"go/types"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/monzo/slog"
)

var (
	defaultTimeout       = 5
	defaultInterval      = 10
	configReloadInterval = 60
	cfg                  = types.Config{}
	cfgMutex             = sync.RWMutex{}
	configAPIBase        = ""
	leafID               = ""
)

func main() {
	ctx := context.Background()

	// Initialize metrics server
	metricsErr := make(chan error, 1)
	if err := startMetricsServer(metricsErr); err != nil {
		slog.Critical(ctx, "Failed to start metrics server: %+v, cannot continue", err)
		panic(err)
	}

	// Start exchange rate fetchers
	// TODO: start multiple fetchers depending on whether each is configured
	if err := startFetcher(ctx, &twelvedataExchangeRateFetcher{}); err != nil {
		slog.Critical(ctx, "Error starting twelvedata fetcher: %v, cannot continue", err)
		panic(err)
	}

	// Log termination gracefully
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-done:
			slog.Info(ctx, "Forex Exporter shutting down...")
			return
		case err := <-metricsErr:
			slog.Error(ctx, "Forex Exporter stopping due to metrics server error %+v", err)
			return
		}
	}
}
