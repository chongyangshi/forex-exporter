package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/monzo/slog"
	"golang.org/x/sync/errgroup"
)

type currencyPair struct {
	Source string
	Target string
}

type currencyExchangeRate struct {
	Rate      float64
	Timestamp int
}

type exchangeRateFetcher interface {
	initFetcher(ctx context.Context, currencyPairs []currencyPair) error
	getIntervalSeconds() int
	getName() string
	fetchRate(ctx context.Context, sourceCurrency, targetCurrency string) (*currencyExchangeRate, error)
}

func getCurrencyPairsFromEnv() ([]currencyPair, error) {
	currencyPairsRaw := strings.TrimSpace(os.Getenv("FOREX_EXPORTER_CURRENCY_PAIRS"))
	if currencyPairsRaw == "" {
		return nil, fmt.Errorf("No currency pairs supplied in FOREX_EXPORTER_CURRENCY_PAIRS")
	}

	// TODO: better validation with golang.org/x/text/currency
	var currencyPairs []currencyPair
	for _, pair := range strings.Split(currencyPairsRaw, ",") {
		pairSplit := strings.Split(strings.TrimSpace(pair), "/")
		if len(pairSplit) != 2 {
			return nil, fmt.Errorf("Invalid currency pair: %s", pair)
		}

		for _, currency := range pairSplit {
			if !isAlphabetic(currency) || len(currency) != 3 {
				return nil, fmt.Errorf("Invalid currency code: %s", currency)
			}
		}

		currencyPairs = append(currencyPairs, currencyPair{Source: pairSplit[0], Target: pairSplit[1]})
	}

	return currencyPairs, nil
}

func isAlphabetic(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func startFetcher(ctx context.Context, fetcher exchangeRateFetcher) error {
	currencyPairs, err := getCurrencyPairsFromEnv()
	if err != nil {
		return err
	}

	if err := fetcher.initFetcher(ctx, currencyPairs); err != nil {
		slog.Critical(ctx, "Error initialising fetcher %s: %+v", fetcher.getName(), err)
		return err
	}

	ticker := time.NewTicker(time.Second * time.Duration(fetcher.getIntervalSeconds()))
	go func() {
		slog.Info(ctx, "Initialising fetcher %s...", fetcher.getName())
		// Run once at start, then every selected interval of the ticker.
		for ; true; <-ticker.C {
			g, ctx := errgroup.WithContext(ctx)

			for _, pair := range currencyPairs {
				cPair := pair
				slog.Debug(ctx, "Retrieving %s/%s rate from %s...", cPair.Source, cPair.Target, fetcher.getName())

				g.Go(func() error {
					rate, err := fetcher.fetchRate(ctx, cPair.Source, cPair.Target)
					if err != nil {
						slog.Error(ctx, "Error retrieving %s/%s rate from %s: %+v", cPair.Source, cPair.Target, fetcher.getName(), err)
						return err
					}

					slog.Debug(ctx, "Registering forex rate %f for %s/%s", rate, cPair.Source, cPair.Target)
					registerForexRate(cPair.Source, cPair.Target, fetcher.getName(), rate.Rate)
					registerForexRateTimestamp(cPair.Source, cPair.Target, fetcher.getName(), rate.Timestamp)
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				slog.Error(ctx, "Error retrieving rate: %+v, retrying shortly.", err)
			}
		}
	}()

	// Gracefully stop if terminated
	go func() {
		done := make(chan os.Signal, 1)
		signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

		<-done
		slog.Info(ctx, "Shutting down fetcher for %s", fetcher.getName())
		ticker.Stop()
	}()

	return nil
}
