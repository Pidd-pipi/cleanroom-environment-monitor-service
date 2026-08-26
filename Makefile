# Makefile for cleanroom-environment-monitor-service.
#
# Common targets: build / test / vet / fmt-check / race / run / docker-build.

BINARY := cleanroom-environment-monitor-service
IMAGE  := cleanroom-environment-monitor-service:latest

.PHONY: build test vet fmt-check race run clean docker-build

build:
	CGO_ENABLED=0 go build -trimpath -o bin/$(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

fmt-check:
	test -z "$$(gofmt -l .)"

race:
	go test -race ./...

run:
	PORT=$${PORT:-8080} go run .

clean:
	rm -rf bin data

docker-build:
	docker build -t $(IMAGE) .
