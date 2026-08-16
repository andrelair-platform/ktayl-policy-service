package handlers_test

import (
	"bytes"
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

func setupTransitionRouter(svc *domain.PolicyService) *chi.Mux {
	h := handlers.NewTransitionHandler(svc)
	r := chi.NewRouter()
	r.Post("/v1/policies/{id}/submit", h.Submit)
	r.Post("/v1/policies/{id}/activate", h.Activate)
	r.Post("/v1/policies/{id}/cancel", h.Cancel)
	r.Get("/v1/policies/{id}/history", h.History)
	return r
}

func newDraftInRepo() (*mockPolicyRepo, *domain.Policy) {
	repo := newMockRepo()
	p := &domain.Policy{
		ID:            uuid.New(),
		PolicyNumber:  "POL-T-001",
		HolderName:    "Marie Dupont",
		ProductCode:   "IARD-AUTO-RC",
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
		Status:        domain.StatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	repo.policies[p.ID] = p
	return repo, p
}

// ─── Submit ──────────────────────────────────────────────────────────────────

func TestSubmitPolicy_400_InvalidUUID(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/not-a-uuid/submit", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSubmitPolicy_404_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+uuid.New().String()+"/submit", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestSubmitPolicy_409_WrongStatus(t *testing.T) {
	repo, p := newDraftInRepo()
	p.Status = domain.StatusActive // Active → Submit is an invalid transition
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+p.ID.String()+"/submit", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Activate ────────────────────────────────────────────────────────────────

func TestActivatePolicy_400_InvalidUUID(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/not-a-uuid/activate", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestActivatePolicy_404(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+uuid.New().String()+"/activate", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestActivatePolicy_409_WrongStatus(t *testing.T) {
	repo, p := newDraftInRepo()
	p.Status = domain.StatusDraft // Draft → Activate is invalid
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+p.ID.String()+"/activate", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── Cancel (transition — active policy) ────────────────────────────────────

func TestCancelTransition_400_BadJSON(t *testing.T) {
	repo, p := newDraftInRepo()
	p.Status = domain.StatusActive
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+p.ID.String()+"/cancel",
		bytes.NewBufferString("{not json"))
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestCancelTransition_422_MissingReason(t *testing.T) {
	repo, p := newDraftInRepo()
	p.Status = domain.StatusActive
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	body, _ := json.Marshal(map[string]string{"reason": ""})
	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+p.ID.String()+"/cancel",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
}

func TestCancelTransition_404_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	body, _ := json.Marshal(map[string]string{"reason": "CUSTOMER_REQUEST"})
	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+uuid.New().String()+"/cancel",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestCancelTransition_409_WrongStatus(t *testing.T) {
	repo, p := newDraftInRepo()
	p.Status = domain.StatusDraft // Draft → Cancel is invalid in applyTransition
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	body, _ := json.Marshal(map[string]string{"reason": "CUSTOMER_REQUEST"})
	req := httptest.NewRequest(http.MethodPost, "/v1/policies/"+p.ID.String()+"/cancel",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── History ─────────────────────────────────────────────────────────────────

func TestHistory_400_InvalidUUID(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/policies/not-a-uuid/history", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestHistory_404_NotFound(t *testing.T) {
	svc := domain.NewPolicyService(newMockRepo(), domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/policies/"+uuid.New().String()+"/history", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestHistory_200_EmptyList(t *testing.T) {
	repo, p := newDraftInRepo()
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	rtr := setupTransitionRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/policies/"+p.ID.String()+"/history", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	hist, ok := resp["history"]
	if !ok {
		t.Fatal("response missing 'history' key")
	}
	arr, ok := hist.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty history array, got %v", hist)
	}
}
