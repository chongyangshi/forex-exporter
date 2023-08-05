package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/monzo/slog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	listenHostPortEnv = "FOREX_EXPORTER_LISTEN"
	defaultListenHost = "0.0.0.0"
	defaultListenPort = 9299

	recentRestartsWindow    = time.Minute * 10
	maxRecentRestarts       = 5
	terminationGraceSeconds = 10
)

var (
	lastRestartTime time.Time
	recentRestarts  int
)

var (
	exchangeRateMetrics = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "forex_exporter",
		Name:      "exchange_rate",
		Help:      "Record the exchange rate between a currency pair",
	}, []string{"source_currency", "target_currency"})
)

func registerForexRate(source_currency, targetCurrency string, rate float64) {
	exchangeRateMetrics.WithLabelValues(source_currency, targetCurrency).Set(rate)
}

func startMetricsServer(errChan chan error) error {
	ctx := context.Background()

	host := defaultListenHost
	port := defaultListenPort
	envHostPort := os.Getenv(listenHostPortEnv)
	if envHostPort != "" {
		parsedHost, parsedPort, err := net.SplitHostPort(envHostPort)
		if err != nil {
			slog.Critical(ctx, "Invalid listen host port: %s, cannot initialize", envHostPort)
			return err
		}
		host = parsedHost

		portNum, err := strconv.ParseInt(parsedPort, 10, 64)
		if err != nil || portNum < 1 || portNum > 32767 {
			slog.Critical(ctx, "Invalid port: %s, cannot initialize", parsedPort)
			return err
		}
		port = int(portNum)
	}

	srvmx := http.NewServeMux()

	// net.SplitHostPort accepts unspecified host, which means when e.g. ":8080" is
	// requested the host will simply be empty string and is valid.
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port), Handler: srvmx}
	srvmx.Handle("/metrics", promhttp.Handler())

	// A simple automatic recovery routine for the metrics server with limited recent retries
	go func() {
		for {

			if err := server.ListenAndServe(); err != nil {
				slog.Error(ctx, "Local metrics server encountered error: %v", err)

				timeOfError := time.Now()

				if timeOfError.Sub(lastRestartTime) > recentRestartsWindow {
					recentRestarts = 0
				}

				if recentRestarts > maxRecentRestarts {
					slog.Critical(ctx, "Too many recent restarts (%d), exiting.", maxRecentRestarts)
					errChan <- err
					return
				}

				slog.Warn(ctx, "Restaring metrics server following recent error %v", err)
				recentRestarts++
			}
		}
	}()

	// Gracefully stop if terminated
	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		<-done
		slog.Info(ctx, "Shutting down metrics server with grace period %ds...", terminationGraceSeconds)
		ctx, cancel := context.WithTimeout(context.Background(), terminationGraceSeconds*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error(ctx, "Error shutting down server: %+v, bailing out", err)
			return
		}
	}()

	return nil
}
