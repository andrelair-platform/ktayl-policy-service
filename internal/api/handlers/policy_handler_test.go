package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api/handlers"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ─── mock repo ──────────────────────────────────────────────────────────────

type mockPolicyRepo struct {
	policies map[uuid.UUID]*domain.Policy
	createFn func(ctx context.Context, p *domain.Policy) error
}

func newMockRepo() *mockPolicyRepo {
	return &mockPolicyRepo{policies: make(map[uuid.UUID]*domain.Policy)}
}

var _ domain.PolicyRepository = (*mockPolicyRepo)(nil)

func (m *mockPolicyRepo) Create(ctx context.Context, p *domain.Policy) error {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	m.policies[p.ID] = p
	return nil
}

func (m *mockPolicyRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Policy, error) {
	p, ok := m.policies[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (m *mockPolicyRepo) GetByPolicyNumber(_ context.Context, number string) (*domain.Policy, error) {
	for _, p := range m.policies {
		if p.PolicyNumber == number {
			return p, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockPolicyRepo) Update(_ context.Context, p *domain.Policy) error {
	m.policies[p.ID] = p
	return nil
}

func (m *mockPolicyRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Policy, error) {
	out := make([]*domain.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		out = append(out, p)
	}
	return out, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

func buildHandler(repo domain.PolicyRepository) *handlers.PolicyHandler {
	svc := domain.NewPolicyService(repo)
	return handlers.NewPolicyHandler(svc)
}

func chiCtxWithID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func validCreateBody(t *testing.T) []byte {
	t.Helper()
	b, _ := json.Marshal(map[string]string{
		"policy_number":  "POL-001",
		"holder_name":    "Jean Dupont",
		"product_code":   "IARD-AUTO-RC",
		"effective_date": "2026-09-01T00:00:00Z",
		"expiry_date":    "2027-09-01T00:00:00Z",
	})
	return b
}

// ─── CreatePolicy ───────────────────────────────────────────────────────────

func TestCreatePolicy_201(t *testing.T) {
	h := buildHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/policies", bytes.NewReader(validCreateBody(t)))
	r.Header.Set("Content-Type", "application/json")

	h.CreatePolicy(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "draft" {
		t.Errorf("expected status=draft, got %v", resp["status"])
	}
}

func TestCreatePolicy_400_InvalidJSON(t *testing.T) {
	h := buildHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/policies", bytes.NewReader([]byte(`{bad json`)))
	r.Header.Set("Content-Type", "application/json")

	h.CreatePolicy(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreatePolicy_400_MissingField(t *testing.T) {
	h := buildHandler(newMockRepo())
	body, _ := json.Marshal(map[string]string{
		"holder_name": "Jean",
		// policy_number missing
		"product_code":   "AUTO",
		"effective_date": "2026-09-01T00:00:00Z",
		"expiry_date":    "2027-09-01T00:00:00Z",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/policies", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	h.CreatePolicy(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreatePolicy_409_DuplicatePolicyNumber(t *testing.T) {
	repo := newMockRepo()
	repo.createFn = func(_ context.Context, _ *domain.Policy) error {
		return domain.ErrPolicyNumberTaken
	}
	h := buildHandler(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/policies", bytes.NewReader(validCreateBody(t)))
	r.Header.Set("Content-Type", "application/json")

	h.CreatePolicy(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// ─── GetPolicy ──────────────────────────────────────────────────────────────

func TestGetPolicy_200(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		Status:        domain.StatusDraft,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	h := buildHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/"+id.String(), nil), id.String())

	h.GetPolicy(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetPolicy_404(t *testing.T) {
	h := buildHandler(newMockRepo())
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/"+id.String(), nil), id.String())

	h.GetPolicy(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetPolicy_400_InvalidUUID(t *testing.T) {
	h := buildHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/not-a-uuid", nil), "not-a-uuid")

	h.GetPolicy(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ─── CancelPolicy ───────────────────────────────────────────────────────────

func TestCancelPolicy_204(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		Status:        domain.StatusDraft,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	h := buildHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodDelete, "/v1/policies/"+id.String(), nil), id.String())

	h.CancelPolicy(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

// ─── UpdatePolicy ───────────────────────────────────────────────────────────

func TestUpdatePolicy_200(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		Status:        domain.StatusDraft,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	h := buildHandler(repo)
	body, _ := json.Marshal(map[string]string{
		"holder_name":    "Marie Martin",
		"product_code":   "IARD-HAB-MRH",
		"effective_date": "2026-09-01T00:00:00Z",
		"expiry_date":    "2027-09-01T00:00:00Z",
	})
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPut, "/v1/policies/"+id.String(), bytes.NewReader(body)), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.UpdatePolicy(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["holder_name"] != "Marie Martin" {
		t.Errorf("holder_name not updated: %v", resp["holder_name"])
	}
}

func TestUpdatePolicy_404(t *testing.T) {
	h := buildHandler(newMockRepo())
	body, _ := json.Marshal(map[string]string{
		"holder_name":    "Marie",
		"product_code":   "AUTO",
		"effective_date": "2026-09-01T00:00:00Z",
		"expiry_date":    "2027-09-01T00:00:00Z",
	})
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPut, "/v1/policies/"+id.String(), bytes.NewReader(body)), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.UpdatePolicy(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ─── ListPolicies ────────────────────────────────────────────────────────────

func TestListPolicies_200(t *testing.T) {
	repo := newMockRepo()
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.policies[id] = &domain.Policy{
			ID:            id,
			PolicyNumber:  uuid.NewString(),
			HolderName:    "Jean",
			ProductCode:   "AUTO",
			Status:        domain.StatusDraft,
			EffectiveDate: time.Now().Add(24 * time.Hour),
			ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
		}
	}
	h := buildHandler(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/policies", nil)

	h.ListPolicies(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 3 {
		t.Errorf("expected 3 items, got %v", resp["items"])
	}
}

func TestCancelPolicy_409_ActivePolicy(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean",
		ProductCode:   "AUTO",
		Status:        domain.StatusActive,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
	h := buildHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodDelete, "/v1/policies/"+id.String(), nil), id.String())

	h.CancelPolicy(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}
