package events

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName    = "POLICY_EVENTS"
	streamSubject = "policy.>"
	// 7-day retention per CdCF §8 (inter-service event replay window).
	streamMaxAge = 7 * 24 * time.Hour

	maxRetries    = 3
	retryBase     = 5 * time.Second
	retryMultiple = 5
)

// Publisher manages the NATS JetStream connection and event publishing.
type Publisher struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	log *slog.Logger
}

// Connect creates a NATS connection, ensures the POLICY_EVENTS stream exists,
// and returns a ready Publisher. caCert may be empty (plain TCP).
func Connect(ctx context.Context, natsURL, caCert string, log *slog.Logger) (*Publisher, error) {
	opts := []nats.Option{
		nats.Name("ktayl-policy-service"),
		nats.MaxReconnects(-1), // infinite reconnect
		nats.ReconnectWait(2 * time.Second),
	}

	if caCert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caCert)) {
			return nil, fmt.Errorf("nats: invalid CA cert PEM")
		}
		opts = append(opts, nats.Secure(&tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}))
	}

	nc, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect %s: %w", natsURL, err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	if err := ensureStream(ctx, js); err != nil {
		nc.Close()
		return nil, err
	}

	log.Info("NATS connected", "url", natsURL, "stream", StreamName)
	return &Publisher{nc: nc, js: js, log: log}, nil
}

func ensureStream(ctx context.Context, js jetstream.JetStream) error {
	cfg := jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{streamSubject},
		MaxAge:    streamMaxAge,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
		Retention: jetstream.LimitsPolicy,
		Discard:   jetstream.DiscardOld,
	}
	_, err := js.CreateOrUpdateStream(ctx, cfg)
	if err != nil {
		return fmt.Errorf("ensure stream %s: %w", StreamName, err)
	}
	return nil
}

// Publish serialises a PolicyEvent as a CloudEvents-encoded NATS message.
// It retries up to maxRetries times with exponential backoff (AC-5).
// Call this AFTER the DB transaction commits (AC-4).
func (p *Publisher) Publish(ctx context.Context, event *PolicyEvent) {
	subject := "policy." + event.EventType
	data, err := json.Marshal(event.toCloudEvent())
	if err != nil {
		p.log.Error("failed to marshal event", "type", event.EventType, "err", err)
		return
	}

	wait := retryBase
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, err = p.js.Publish(ctx, subject, data)
		if err == nil {
			p.log.Info("event published", "subject", subject, "policy_id", event.PolicyID)
			return
		}
		p.log.Error("event publish failed", "subject", subject, "attempt", attempt, "err", err)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			wait *= retryMultiple
		}
	}
	p.log.Error("event publish abandoned after retries", "subject", subject, "policy_id", event.PolicyID)
}

// IsConnected reports whether the underlying NATS connection is alive.
func (p *Publisher) IsConnected() bool {
	return p.nc != nil && p.nc.IsConnected()
}

// Close drains and closes the NATS connection.
func (p *Publisher) Close() {
	if p.nc != nil {
		_ = p.nc.Drain()
	}
}
