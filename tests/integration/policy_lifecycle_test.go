package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/andrelair-platform/ktayl-policy-service/internal/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"log/slog"
	"os"
)

func testPolicy() *domain.Policy {
	return &domain.Policy{
		PolicyNumber:  "INT-" + uuid.NewString()[:8],
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		EffectiveDate: time.Now().UTC().Add(24 * time.Hour),
		ExpiryDate:    time.Now().UTC().Add(365 * 24 * time.Hour),
	}
}

// ─── CRUD round-trip ─────────────────────────────────────────────────────────

func TestIntegration_Create_GetByID(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "actor"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID == uuid.Nil {
		t.Fatal("ID not set after Create")
	}
	if p.Status != domain.StatusDraft {
		t.Fatalf("expected draft, got %s", p.Status)
	}

	got, err := svc.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.PolicyNumber != p.PolicyNumber {
		t.Errorf("policy_number mismatch: %s != %s", got.PolicyNumber, p.PolicyNumber)
	}
}

func TestIntegration_Update(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "actor"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	upd, err := svc.Update(ctx, p.ID, &domain.Policy{
		HolderName:    "Marie Martin",
		ProductCode:   "IARD-HAB-MRH",
		EffectiveDate: p.EffectiveDate,
		ExpiryDate:    p.ExpiryDate,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if upd.HolderName != "Marie Martin" {
		t.Errorf("holder_name not updated: %s", upd.HolderName)
	}
}

func TestIntegration_Cancel_Draft(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "actor"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.Cancel(ctx, p.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	got, _ := svc.GetByID(ctx, p.ID)
	if got.Status != domain.StatusTerminated {
		t.Errorf("expected terminated, got %s", got.Status)
	}
}

// ─── Full state machine lifecycle ─────────────────────────────────────────────

func TestIntegration_FullLifecycle_CreateSubmitActivateCancelActive(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Draft → Submitted
	p2, err := svc.Submit(ctx, p.ID, "broker@ktayl.fr")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if p2.Status != domain.StatusSubmitted {
		t.Fatalf("expected submitted, got %s", p2.Status)
	}

	// Submitted → Active
	p3, err := svc.Activate(ctx, p.ID, "underwriter@ktayl.fr")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if p3.Status != domain.StatusActive {
		t.Fatalf("expected active, got %s", p3.Status)
	}

	// Active → Cancelled
	p4, err := svc.CancelActive(ctx, p.ID, "broker@ktayl.fr", "CUSTOMER_REQUEST")
	if err != nil {
		t.Fatalf("CancelActive: %v", err)
	}
	if p4.Status != domain.StatusCancelled {
		t.Fatalf("expected cancelled, got %s", p4.Status)
	}

	// Verify final state in DB
	final, err := svc.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if final.Status != domain.StatusCancelled {
		t.Errorf("DB status mismatch: expected cancelled, got %s", final.Status)
	}
}

func TestIntegration_FullLifecycle_SubmitReject(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Submit(ctx, p.ID, "broker"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	p3, err := svc.Reject(ctx, p.ID, "underwriter", "UNDERWRITER_REJECTED")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if p3.Status != domain.StatusRejected {
		t.Fatalf("expected rejected, got %s", p3.Status)
	}
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func TestIntegration_AuditLog_ThreeEntries(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Submit(ctx, p.ID, "broker"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := svc.Activate(ctx, p.ID, "uw"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if _, err := svc.CancelActive(ctx, p.ID, "broker", "NON_PAYMENT"); err != nil {
		t.Fatalf("CancelActive: %v", err)
	}

	logs, err := svc.ListHistory(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 audit entries, got %d", len(logs))
	}

	// Verify ordered transitions
	transitions := [][2]domain.PolicyStatus{
		{domain.StatusDraft, domain.StatusSubmitted},
		{domain.StatusSubmitted, domain.StatusActive},
		{domain.StatusActive, domain.StatusCancelled},
	}
	for i, tr := range transitions {
		if logs[i].FromStatus != tr[0] || logs[i].ToStatus != tr[1] {
			t.Errorf("audit[%d]: want %s→%s, got %s→%s",
				i, tr[0], tr[1], logs[i].FromStatus, logs[i].ToStatus)
		}
	}
}

func TestIntegration_AuditLog_ActorAndReason(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Submit(ctx, p.ID, "broker@ktayl.fr"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := svc.Activate(ctx, p.ID, "uw@ktayl.fr"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if _, err := svc.CancelActive(ctx, p.ID, "broker@ktayl.fr", "CUSTOMER_REQUEST"); err != nil {
		t.Fatalf("CancelActive: %v", err)
	}

	logs, err := svc.ListHistory(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}

	if logs[0].ActorID != "broker@ktayl.fr" {
		t.Errorf("submit actor: want broker@ktayl.fr, got %s", logs[0].ActorID)
	}
	if logs[2].Reason != "CUSTOMER_REQUEST" {
		t.Errorf("cancel reason: want CUSTOMER_REQUEST, got %s", logs[2].Reason)
	}
}

// ─── Invalid transition guard (DB-backed) ────────────────────────────────────

func TestIntegration_InvalidTransition_ActivateFromDraft(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err := svc.Activate(ctx, p.ID, "uw") // draft → active is not valid
	if err == nil {
		t.Fatal("expected ErrInvalidTransition, got nil")
	}

	// DB must still show draft — transaction rolled back
	got, _ := svc.GetByID(ctx, p.ID)
	if got.Status != domain.StatusDraft {
		t.Errorf("expected draft after failed transition, got %s", got.Status)
	}
}

// ─── NATS JetStream events ────────────────────────────────────────────────────

func TestIntegration_NATS_EventsPublished(t *testing.T) {
	if integrationNATSURL == "" {
		t.Skip("INTEGRATION_NATS_URL not set")
	}

	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	pub, err := events.Connect(ctx, integrationNATSURL, "", log)
	if err != nil {
		t.Fatalf("NATS connect: %v", err)
	}
	defer pub.Close()

	svc, _ := newService(t)
	svc.SetPublisher(pub)

	// Subscribe before triggering events
	nc, _ := nats.Connect(integrationNATSURL)
	defer nc.Close()
	js, _ := jetstream.New(nc)

	cons, err := js.CreateOrUpdateConsumer(ctx, events.StreamName, jetstream.ConsumerConfig{
		FilterSubject: "policy.>",
		AckPolicy:     jetstream.AckNonePolicy,
	})
	if err != nil {
		t.Fatalf("create consumer: %v", err)
	}

	// Trigger 3 transitions
	p := testPolicy()
	if err := svc.Create(ctx, p, "system"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.Submit(ctx, p.ID, "broker"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := svc.Activate(ctx, p.ID, "uw"); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	// Wait for async publishes to land (goroutines in PublishAsync)
	time.Sleep(300 * time.Millisecond)

	// Expect 3 messages: created, submitted, activated
	subjects := make(map[string]bool)
	for i := 0; i < 3; i++ {
		msg, err := cons.Next(jetstream.FetchMaxWait(2 * time.Second))
		if err != nil {
			t.Fatalf("message %d not received: %v", i+1, err)
		}
		var ce map[string]any
		if err := json.Unmarshal(msg.Data(), &ce); err != nil {
			t.Fatalf("unmarshal CloudEvent: %v", err)
		}
		// Validate CloudEvents fields
		if ce["specversion"] != "1.0" {
			t.Errorf("msg %d: specversion want 1.0, got %v", i+1, ce["specversion"])
		}
		if ce["source"] != "ktayl-policy-service" {
			t.Errorf("msg %d: source want ktayl-policy-service, got %v", i+1, ce["source"])
		}
		subjects[ce["type"].(string)] = true
	}

	for _, want := range []string{"com.ktayl.policy.created", "com.ktayl.policy.submitted", "com.ktayl.policy.activated"} {
		if !subjects[want] {
			t.Errorf("missing event type: %s", want)
		}
	}
}
