package main

// Fetcher for twelvedata API: https://twelvedata.com/docs#exchange-rate

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/monzo/slog"
	"github.com/monzo/typhon"
)

const (
	twelvedataAPIKeyEnv         = "FOREX_EXPORTER_TWELVEDATA_API_KEY"
	twelvedataAPIURLTemplate    = "https://api.twelvedata.com/currency_conversion?symbol=%s/%s&apikey=%s"
	twelvedataAPITimeoutSeconds = 10

	// Currently twelvedata has a free quota of 800 API credit per day,
	// and each request for a currency pair rate consumes one credit.
	// This translates to 2 minutes interval per currency pair.
	twelvedataAPIIntervalSecondsPerPair = 120
)

type twelvedataExchangeRateFetcher struct {
	apiKey          string
	client          typhon.Service
	intervalSeconds int
}

type twelvedataExchangeRateResponse struct {
	Symbol    string  `json:"symbol"`
	Rate      float64 `json:"rate"`
	Timestamp int     `json:"timestamp"`
}

func (f *twelvedataExchangeRateFetcher) initFetcher(ctx context.Context, currencyPairs []currencyPair) error {
	apiKey := strings.TrimSpace(os.Getenv(twelvedataAPIKeyEnv))
	if apiKey == "" {
		err := fmt.Errorf("No API key supplied for twelvedata exporter")
		return err
	}
	f.apiKey = apiKey

	roundTripper := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Second * time.Duration(twelvedataAPITimeoutSeconds),
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   time.Second * time.Duration(twelvedataAPITimeoutSeconds),
		ResponseHeaderTimeout: time.Second * time.Duration(twelvedataAPITimeoutSeconds),
		ExpectContinueTimeout: 1 * time.Second,
	}
	f.client = typhon.HttpService(roundTripper).Filter(typhon.ExpirationFilter).Filter(typhon.H2cFilter).Filter(typhon.ErrorFilter)

	f.intervalSeconds = twelvedataAPIIntervalSecondsPerPair * len(currencyPairs)
	return nil
}

func (f twelvedataExchangeRateFetcher) getIntervalSeconds() int {
	return f.intervalSeconds
}

func (f twelvedataExchangeRateFetcher) getName() string {
	return "twelvedata"
}

func (f twelvedataExchangeRateFetcher) fetchRate(ctx context.Context, sourceCurrency, targetCurrency string) (*currencyExchangeRate, error) {
	requestURL := fmt.Sprintf(twelvedataAPIURLTemplate, sourceCurrency, targetCurrency, f.apiKey)
	r := typhon.NewRequest(ctx, http.MethodGet, requestURL, nil).SendVia(f.client).Response()
	if r.Error != nil {
		slog.Error(ctx, "Received error from twelvedata API: %+v", r.Error)
		return nil, r.Error
	}

	rawResp, err := r.BodyBytes(true)
	if err != nil {
		slog.Error(ctx, "Error reading response from twelvedata API: %+v", err)
		return nil, err
	}

	var resp = twelvedataExchangeRateResponse{}
	if err := json.Unmarshal(rawResp, &resp); err != nil {
		slog.Error(ctx, "Error parsing response from twelvedata API: %+v", err)
		return nil, err
	}

	return &currencyExchangeRate{
		Rate:      resp.Rate,
		Timestamp: resp.Timestamp,
	}, nil
}
