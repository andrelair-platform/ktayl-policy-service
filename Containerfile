FROM golang:1.25-alpine AS builder
WORKDIR /src

ARG CA_CERT
RUN if [ -n "$CA_CERT" ]; then \
      echo "$CA_CERT" > /usr/local/share/ca-certificates/minicloud-ca.crt && \
      update-ca-certificates; \
    fi

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ktayl-policy-service ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /ktayl-policy-service /ktayl-policy-service
EXPOSE 8080
ENTRYPOINT ["/ktayl-policy-service"]
