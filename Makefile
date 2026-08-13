BINARY   := ktayl-policy-service
IMAGE    := harbor.10.0.0.200.nip.io/library/ktayl-policy-service
SHA      := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: lint test test-cov build build-image run

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

test-cov:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY) ./cmd/server

build-image:
	docker build \
		--build-arg CA_CERT="$$(cat ~/minicloud-ca.crt)" \
		-t $(IMAGE):$(SHA) \
		-f Containerfile .

run:
	PORT=8080 go run ./cmd/server
