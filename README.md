# Forex Exporter

The `forex-exporter` is a simgle [Prometheus](https://github.com/prometheus/prometheus) metrics exporter for currency exchange rate data. It is written with Golang and can run either as a systemd service or a Kubernetes deployment.

If refreshing the dashboards of your favourite forex provider and setting [limit orders](https://wise.com/help/articles/0QO88oPwfcCqgAX1fvN6D/what-are-auto-conversions) are not enough for you, then this might be exactly what you need.

<img width="945" alt="Prometheus metrics example" src="https://github.com/chongyangshi/forex-exporter/assets/8771937/cd417b59-f4c9-47a8-b6df-d9552873de1c">

The exporter current uses [Twelve Data](https://twelvedata.com/docs#exchange-rate) as the data source, which provides a free tier. The codebase should have the flexibility to integrate additional providers. 

## How to use

You will first need to obtain an API key [from Twelve Data](https://twelvedata.com/pricing). At the time of writing Twelve Data offers a free API tier which provides 800 API calls per day, and the exporter is designed such that the call rate should not exceed this limit under default configurations. Twelve Data charges one API call credit for one rate of each currency pair, so the more currency pairs you request, the slower the update rate will be set at.

You are responsible for complying with Twelve Data's [Services Agreement](https://twelvedata.com/terms) when using `forex-exporter` to load and display exchange rate data.

Once you have obtained the API key, deploy `forex-exporter` depending on whether you need to run on Linux systemd or Kubernetes:

### Linux `systemd` service deployment

`systemd_setup.sh` will install Golang (if not already present), build the exporter, and then set up a systemd service for you. If you would prefer to set it up manually, read the script for details.

The required parameters are `listen_host:listen_port`, list of currency pairs in the format of "SRC/DST" separated by comma (from [ISO 4217](https://en.wikipedia.org/wiki/ISO_4217#List_of_ISO_4217_currency_codes) currency codes), and the Twelve Data API key.

```bash
git clone https://github.com/chongyangshi/forex-exporter.git
bash systemd_setup.sh :9299 USD/GBP,USD/EUR PLACE_YOUR_TWELVE_DATA_API_KEY_HERE
```

### Kubernetes deployment

See the sample manifest in [k8s.yaml](k8s.yaml), you should change the `FOREX_EXPORTER_CURRENCY_PAIRS` environment variable value to the currency pair(s) you want, and at the bottom you need change the Kubernetes Secret `apikey` value to be the base64-encoded form of your Twelve Data API key:

```bash
echo -n "PLACE_YOUR_TWELVE_DATA_API_KEY_HERE" | base64
```

You can then apply the manifest with `kubectl apply -f k8s.yaml` into your cluster.

## Development

PRs integrating more data sources or otherwise improving the exporter are welcome. To run locally, use the `run_local.sh` script with your API key:

```bash
TWELVEDATA_API_KEY=PLACE_YOUR_TWELVE_DATA_API_KEY_HERE bash run_local.sh
```

And the exported metrics will be available at [http://127.0.0.1:18080/metrics](http://127.0.0.1:18080/metrics).
