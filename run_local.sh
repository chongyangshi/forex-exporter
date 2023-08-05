#!/bin/sh

if [[ -z "$TWELVEDATA_API_KEY" ]]; then
    echo "Must set TWELVEDATA_API_KEY to run locally" 1>&2
    exit 1
fi

export FOREX_EXPORTER_LISTEN="127.0.0.1:18080"
export FOREX_EXPORTER_CURRENCY_PAIRS="USD/GBP,USD/EUR"
export FOREX_EXPORTER_TWELVEDATA_API_KEY=${TWELVEDATA_API_KEY}

echo "Running Forex Exporter locally on ${FOREX_EXPORTER_LISTEN}..."
go run github.com/chongyangshi/forex-exporter