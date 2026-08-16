BINARY   := ktayl-policy-service
IMAGE    := harbor.10.0.0.200.nip.io/library/ktayl-policy-service
SHA      := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

.PHONY: test test-cov lint build build-image run vuln sec

lint:
	golangci-lint run ./...

test:
	go test -v -race -count=1 ./...

test-cov:
	go test -race -count=1 \
		-coverprofile=coverage.out \
		-coverpkg=./internal/domain/...,./internal/api/... \
		./...
	go tool cover -func=coverage.out

vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

sec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -severity high -confidence medium ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags "-s -w" -o bin/$(BINARY) ./cmd/server

build-image:
	docker build \
		--build-arg CA_CERT="$$(cat ~/minicloud-ca.crt)" \
		-t $(IMAGE):$(SHA) \
		-f Dockerfile .

run:
	PORT=8080 go run ./cmd/server
