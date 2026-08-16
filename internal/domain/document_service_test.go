package domain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockDocGen struct{ err error }

func (m *mockDocGen) GenerateAttestation(p *domain.Policy) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte("%PDF-mock"), nil
}

type mockDocStore struct {
	uploadErr  error
	presignErr error
	lastKey    string
}

func (m *mockDocStore) Upload(_ context.Context, policyID uuid.UUID, _ []byte) (string, string, error) {
	if m.uploadErr != nil {
		return "", "", m.uploadErr
	}
	m.lastKey = "policies/" + policyID.String() + "/attestation-test.pdf"
	return m.lastKey, "http://minio/presigned?token=x", nil
}

func (m *mockDocStore) PresignKey(_ context.Context, key string) (string, error) {
	if m.presignErr != nil {
		return "", m.presignErr
	}
	return "http://minio/" + key + "?token=x", nil
}

type mockDocRepo struct {
	docs []*domain.Document
}

func (m *mockDocRepo) Create(_ context.Context, d *domain.Document) error {
	d.ID = uuid.New()
	m.docs = append(m.docs, d)
	return nil
}

func (m *mockDocRepo) ListByPolicy(_ context.Context, policyID uuid.UUID) ([]*domain.Document, error) {
	var out []*domain.Document
	for _, d := range m.docs {
		if d.PolicyID == policyID {
			out = append(out, d)
		}
	}
	return out, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newDocService(policyStatus domain.PolicyStatus) (*domain.DocumentService, uuid.UUID, *mockDocRepo) {
	policyID := uuid.New()
	pRepo := newMock()
	pRepo.policies[policyID] = &domain.Policy{
		ID:            policyID,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		Status:        policyStatus,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	dRepo := &mockDocRepo{}
	svc := domain.NewDocumentService(pRepo, dRepo, &mockDocGen{}, &mockDocStore{}, time.Hour)
	return svc, policyID, dRepo
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestDocumentService_GenerateAttestation_Active(t *testing.T) {
	svc, id, repo := newDocService(domain.StatusActive)
	result, err := svc.GenerateAttestation(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Document.ID == uuid.Nil {
		t.Error("document ID should be set")
	}
	if result.URL == "" {
		t.Error("presigned URL should not be empty")
	}
	if len(repo.docs) != 1 {
		t.Errorf("expected 1 document record, got %d", len(repo.docs))
	}
}

func TestDocumentService_GenerateAttestation_Amended(t *testing.T) {
	svc, id, _ := newDocService(domain.StatusAmended)
	_, err := svc.GenerateAttestation(context.Background(), id)
	if err != nil {
		t.Fatalf("AMENDED policy should be eligible, got: %v", err)
	}
}

func TestDocumentService_GenerateAttestation_Draft_Forbidden(t *testing.T) {
	svc, id, _ := newDocService(domain.StatusDraft)
	_, err := svc.GenerateAttestation(context.Background(), id)
	if !errors.Is(err, domain.ErrDocumentForbidden) {
		t.Fatalf("expected ErrDocumentForbidden for DRAFT policy, got %v", err)
	}
}

func TestDocumentService_GenerateAttestation_Submitted_Forbidden(t *testing.T) {
	svc, id, _ := newDocService(domain.StatusSubmitted)
	_, err := svc.GenerateAttestation(context.Background(), id)
	if !errors.Is(err, domain.ErrDocumentForbidden) {
		t.Fatalf("expected ErrDocumentForbidden for SUBMITTED policy, got %v", err)
	}
}

func TestDocumentService_GenerateAttestation_NotFound(t *testing.T) {
	svc, _, _ := newDocService(domain.StatusActive)
	_, err := svc.GenerateAttestation(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrPolicyNotFound) {
		t.Fatalf("expected ErrPolicyNotFound, got %v", err)
	}
}

func TestDocumentService_ListDocuments(t *testing.T) {
	svc, id, _ := newDocService(domain.StatusActive)
	// Generate two attestations
	_, _ = svc.GenerateAttestation(context.Background(), id)
	_, _ = svc.GenerateAttestation(context.Background(), id)

	docs, urls, err := svc.ListDocuments(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
	if len(urls) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(urls))
	}
}

func TestDocumentService_GenerateAttestation_MinIOError(t *testing.T) {
	policyID := uuid.New()
	pRepo := newMock()
	pRepo.policies[policyID] = &domain.Policy{
		ID:            policyID,
		PolicyNumber:  "POL-002",
		HolderName:    "Marie Martin",
		ProductCode:   "IARD-HAB-MRH",
		Status:        domain.StatusActive,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	store := &mockDocStore{uploadErr: errors.New("minio: connection refused")}
	svc := domain.NewDocumentService(pRepo, &mockDocRepo{}, &mockDocGen{}, store, time.Hour)

	_, err := svc.GenerateAttestation(context.Background(), policyID)
	if err == nil {
		t.Fatal("expected error from MinIO upload failure")
	}
}
