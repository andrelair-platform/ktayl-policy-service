package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

// ─── mock repo ──────────────────────────────────────────────────────────────

type mockRepo struct {
	policies map[uuid.UUID]*domain.Policy
	createFn func(ctx context.Context, p *domain.Policy) error
}

func newMock() *mockRepo {
	return &mockRepo{policies: make(map[uuid.UUID]*domain.Policy)}
}

var _ domain.PolicyRepository = (*mockRepo)(nil)

func (m *mockRepo) Create(ctx context.Context, p *domain.Policy) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	m.policies[p.ID] = p
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Policy, error) {
	p, ok := m.policies[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (m *mockRepo) GetByPolicyNumber(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}

func (m *mockRepo) Update(_ context.Context, p *domain.Policy) error {
	m.policies[p.ID] = p
	return nil
}

func (m *mockRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Policy, error) {
	out := make([]*domain.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		out = append(out, p)
	}
	return out, nil
}

// ─── helpers ────────────────────────────────────────────────────────name: ─────

func validPolicy() *domain.Policy {
	return &domain.Policy{
		PolicyNumber:  "POL-TEST-001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
}

// ─── Create ─────────────────────────────────────────────────────────────────

func TestPolicyService_Create_OK(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)
	p := validPolicy()

	if err := svc.Create(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if p.Status != domain.StatusDraft {
		t.Errorf("expected status draft, got %s", p.Status)
	}
}

func TestPolicyService_Create_ValidationError(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)
	p := &domain.Policy{} // missing required fields

	if err := svc.Create(context.Background(), p); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPolicyService_Create_DuplicateNumber(t *testing.T) {
	repo := newMock()
	repo.createFn = func(_ context.Context, _ *domain.Policy) error {
		return domain.ErrPolicyNumberTaken
	}
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)

	if err := svc.Create(context.Background(), validPolicy()); err == nil {
		t.Fatal("expected ErrPolicyNumberTaken")
	}
}

// ─── GetByID ────────────────────────────────────────────────────────────────

func TestPolicyService_GetByID_OK(t *testing.T) {
	repo := newMock()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	p := validPolicy()
	_ = svc.Create(context.Background(), p)

	got, err := svc.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, p.ID)
	}
}

func TestPolicyService_GetByID_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)

	_, err := svc.GetByID(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

// ─── List ───────────────────────────────────────────────────────────────────

func TestPolicyService_List_Empty(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)

	policies, err := svc.List(context.Background(), domain.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(policies))
	}
}

func TestPolicyService_List_WithItems(t *testing.T) {
	repo := newMock()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	for i := 0; i < 3; i++ {
		p := validPolicy()
		p.PolicyNumber = uuid.NewString() // ensure unique
		_ = svc.Create(context.Background(), p)
	}

	policies, err := svc.List(context.Background(), domain.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) != 3 {
		t.Errorf("expected 3 policies, got %d", len(policies))
	}
}

// ─── Update ─────────────────────────────────────────────────────────────────

func TestPolicyService_Update_OK(t *testing.T) {
	repo := newMock()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	p := validPolicy()
	_ = svc.Create(context.Background(), p)

	upd := &domain.Policy{
		HolderName:    "Marie Martin",
		ProductCode:   "IARD-HAB-MRH",
		EffectiveDate: time.Now().Add(48 * time.Hour),
		ExpiryDate:    time.Now().Add(400 * 24 * time.Hour),
	}
	got, err := svc.Update(context.Background(), p.ID, upd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.HolderName != "Marie Martin" {
		t.Errorf("holder name not updated: %s", got.HolderName)
	}
}

func TestPolicyService_Update_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)
	_, err := svc.Update(context.Background(), uuid.New(), validPolicy())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPolicyService_Update_InvalidDates(t *testing.T) {
	repo := newMock()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	p := validPolicy()
	_ = svc.Create(context.Background(), p)

	upd := &domain.Policy{
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		EffectiveDate: time.Now().Add(365 * 24 * time.Hour),
		ExpiryDate:    time.Now().Add(24 * time.Hour), // expiry before effective
	}
	_, err := svc.Update(context.Background(), p.ID, upd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// ─── Cancel ─────────────────────────────────────────────────────────────────

func TestPolicyService_Cancel_Draft(t *testing.T) {
	repo := newMock()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	p := validPolicy()
	_ = svc.Create(context.Background(), p)

	if err := svc.Cancel(context.Background(), p.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := svc.GetByID(context.Background(), p.ID)
	if got.Status != domain.StatusTerminated {
		t.Errorf("expected terminated, got %s", got.Status)
	}
}

func TestPolicyService_Cancel_NotDraft(t *testing.T) {
	repo := newMock()
	id := uuid.New()
	repo.policies[id] = &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-X",
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		Status:        domain.StatusActive,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)

	if err := svc.Cancel(context.Background(), id); err == nil {
		t.Fatal("expected ErrCannotCancel")
	}
}

func TestPolicyService_Cancel_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMock(), domain.NullAuditLog(), nil)
	if err := svc.Cancel(context.Background(), uuid.New()); err == nil {
		t.Fatal("expected error")
	}
}
