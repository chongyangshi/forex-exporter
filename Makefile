.PHONY: build
SVC := forex-exporter
COMMIT := $(shell git log -1 --pretty='%h')
REPOSITORY = icydoge/web

build:
	GOOS=linux CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags="-w -s" -o .

docker:
	docker pull golang:alpine
	docker build -t ${SVC} .
	docker tag ${SVC}:latest ${REPOSITORY}:${SVC}-${COMMIT}
	docker push ${REPOSITORY}:${SVC}-${COMMIT}

install: build
	cp ./forex-exporter /usr/local/bin/forex-exporter
	cp ./forex-exporter.service /etc/systemd/system/forex-exporter.service
	chmod 644 /etc/systemd/system/forex-exporter.service
	systemctl daemon-reload

uninstall:
	systemctl stop forex-exporter
	systemctl disable forex-exporter
	rm /etc/systemd/system/forex-exporter.service
	rm /usr/local/bin/forex-exporter

reinstall: uninstall install