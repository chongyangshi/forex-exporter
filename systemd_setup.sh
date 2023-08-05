#!/bin/sh

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: sh systemd_setup.sh listen_host:listen_port USD/GBP,USD/EUR twelvedata_apikey"
    exit 1
fi;

set -e

GO_VERSION=1.20.7
FOREX_EXPORTER_LISTEN=$1
FOREX_EXPORTER_CURRENCY_PAIRS=$2
FOREX_EXPORTER_TWELVEDATA_API_KEY=$3

arch=""
case $(uname -m) in
    i386)   arch="386" ;;
    i686)   arch="386" ;;
    x86_64) arch="amd64" ;;
    arm64)  arch="arm64" ;;
    *)      echo "Unsupported arch $(uname -m)"; exit 1 ;;
esac

useradd forexexporter || true
sed -i "s#{{FOREX_EXPORTER_LISTEN_VAL}}#${FOREX_EXPORTER_LISTEN}#g" $(pwd)/forex-exporter.service
sed -i "s#{{FOREX_EXPORTER_CURRENCY_PAIRS_VAL}}#${FOREX_EXPORTER_CURRENCY_PAIRS}#g" $(pwd)/forex-exporter.service
sed -i "s#{{FOREX_EXPORTER_TWELVEDATA_API_KEY_VAL}}#${FOREX_EXPORTER_TWELVEDATA_API_KEY}#g" $(pwd)/forex-exporter.service

[ -d "/usr/local/go" ] || [ -f "/tmp/go$GO_VERSION.linux-$arch.tar.gz" ] || wget https://dl.google.com/go/go$GO_VERSION.linux-$arch.tar.gz -O /tmp/go$GO_VERSION.linux-$arch.tar.gz
[ -d "/usr/local/go" ] || sudo tar -C /usr/local -zxf /tmp/go$GO_VERSION.linux-$arch.tar.gz
export PATH=$PATH:/usr/local/go/bin
rm -rf /tmp/go$GO_VERSION.linux-$arch.tar.gz

go get -d -v
sudo make install
systemctl enable forex-exporter
systemctl restart forex-exporter
systemctl status forex-exporter