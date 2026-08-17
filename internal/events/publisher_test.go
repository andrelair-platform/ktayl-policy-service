package events_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/andrelair-platform/ktayl-policy-service/internal/events"
	"github.com/google/uuid"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"os"
)

// startEmbeddedServer starts a local NATS server with JetStream enabled.
// Returns the server and its client URL.
func startEmbeddedServer(t *testing.T) (*natsserver.Server, string) {
	t.Helper()
	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      natsserver.RANDOM_PORT,
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("nats server start: %v", err)
	}
	srv.Start()
	if !srv.ReadyForConnections(3 * time.Second) {
		t.Fatal("nats server not ready")
	}
	return srv, srv.ClientURL()
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func samplePolicy() *domain.Policy {
	now := time.Now().UTC()
	return &domain.Policy{
		ID:            uuid.New(),
		PolicyNumber:  "POL-TEST-001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		Status:        domain.StatusActive,
		EffectiveDate: now.Add(24 * time.Hour),
		ExpiryDate:    now.Add(365 * 24 * time.Hour),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func TestPublisher_Connect_StreamCreated(t *testing.T) {
	srv, url := startEmbeddedServer(t)
	defer srv.Shutdown()

	ctx := context.Background()
	pub, err := events.Connect(ctx, url, "", testLogger())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pub.Close()

	// Verify stream exists
	nc, _ := nats.Connect(url)
	defer nc.Close()
	js, _ := jetstream.New(nc)
	info, err := js.Stream(ctx, events.StreamName)
	if err != nil {
		t.Fatalf("stream not found: %v", err)
	}
	if info.CachedInfo().Config.Name != events.StreamName {
		t.Errorf("wrong stream name: %s", info.CachedInfo().Config.Name)
	}
}

func TestPublisher_IsConnected(t *testing.T) {
	srv, url := startEmbeddedServer(t)
	defer srv.Shutdown()

	pub, err := events.Connect(context.Background(), url, "", testLogger())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pub.Close()

	if !pub.IsConnected() {
		t.Error("expected IsConnected() == true")
	}
}

func TestPublisher_Publish_CloudEventsFormat(t *testing.T) {
	srv, url := startEmbeddedServer(t)
	defer srv.Shutdown()

	ctx := context.Background()
	pub, err := events.Connect(ctx, url, "", testLogger())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pub.Close()

	// Subscribe before publishing
	nc, _ := nats.Connect(url)
	defer nc.Close()
	js, _ := jetstream.New(nc)
	cons, _ := js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		FilterSubject: "policy.activated",
		AckPolicy:     jetstream.AckNonePolicy,
	})

	p := samplePolicy()
	event := events.BuildTransitionEvent("activated", p, "test-user", "UNDERWRITER_APPROVED")
	pub.Publish(ctx, event)

	msg, err := cons.Next(jetstream.FetchMaxWait(2 * time.Second))
	if err != nil {
		t.Fatalf("no message received: %v", err)
	}

	var ce map[string]any
	if err := json.Unmarshal(msg.Data(), &ce); err != nil {
		t.Fatalf("unmarshal CloudEvent: %v", err)
	}

	if ce["specversion"] != "1.0" {
		t.Errorf("specversion: want 1.0, got %v", ce["specversion"])
	}
	if ce["source"] != "ktayl-policy-service" {
		t.Errorf("source: want ktayl-policy-service, got %v", ce["source"])
	}
	if ce["type"] != "com.ktayl.policy.activated" {
		t.Errorf("type: want com.ktayl.policy.activated, got %v", ce["type"])
	}
	if _, ok := ce["id"]; !ok {
		t.Error("missing id field")
	}
	if _, ok := ce["time"]; !ok {
		t.Error("missing time field")
	}
	data, ok := ce["data"].(map[string]any)
	if !ok {
		t.Fatal("missing or invalid data field")
	}
	if data["policy_id"] != p.ID.String() {
		t.Errorf("data.policy_id: want %s, got %v", p.ID, data["policy_id"])
	}
	if data["actor"] != "test-user" {
		t.Errorf("data.actor: want test-user, got %v", data["actor"])
	}
}

func TestPublisher_PublishAsync_AllEventTypes(t *testing.T) {
	srv, url := startEmbeddedServer(t)
	defer srv.Shutdown()

	ctx := context.Background()
	pub, err := events.Connect(ctx, url, "", testLogger())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pub.Close()

	p := samplePolicy()
	eventTypes := []string{"created", "submitted", "activated", "rejected", "amended", "cancelled", "expired"}
	for _, et := range eventTypes {
		pub.PublishAsync(ctx, et, p, "actor", "")
	}

	// Give goroutines time to publish
	time.Sleep(200 * time.Millisecond)

	nc, _ := nats.Connect(url)
	defer nc.Close()
	js, _ := jetstream.New(nc)
	info, err := js.Stream(ctx, events.StreamName)
	if err != nil {
		t.Fatalf("stream info: %v", err)
	}
	if info.CachedInfo().State.Msgs < uint64(len(eventTypes)) {
		t.Errorf("expected at least %d messages in stream, got %d",
			len(eventTypes), info.CachedInfo().State.Msgs)
	}
}

func TestEventTypeForTransition(t *testing.T) {
	cases := []struct {
		status domain.PolicyStatus
		want   string
	}{
		{domain.StatusSubmitted, "submitted"},
		{domain.StatusActive, "activated"},
		{domain.StatusRejected, "rejected"},
		{domain.StatusAmended, "amended"},
		{domain.StatusCancelled, "cancelled"},
		{domain.StatusExpired, "expired"},
	}
	for _, tc := range cases {
		got := events.EventTypeForTransition(tc.status)
		if got != tc.want {
			t.Errorf("EventTypeForTransition(%s): want %s, got %s", tc.status, tc.want, got)
		}
	}
}
