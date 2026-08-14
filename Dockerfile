# syntax=docker/dockerfile:1.6
FROM golang:1.25.13-alpine AS build
WORKDIR /src

ARG CA_CERT
RUN if [ -n "$CA_CERT" ]; then \
      echo "$CA_CERT" > /usr/local/share/ca-certificates/minicloud-ca.crt && \
      update-ca-certificates; \
    fi

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" -o /out/ktayl-policy-service ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ktayl-policy-service /ktayl-policy-service
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/ktayl-policy-service"]
