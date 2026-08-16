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
	"github.com/google/uuid"
)

func buildTransitionHandler(repo domain.PolicyRepository) *handlers.TransitionHandler {
	svc := domain.NewPolicyService(repo, domain.NullAuditLog(), nil)
	return handlers.NewTransitionHandler(svc)
}

func draftPolicy(id uuid.UUID) *domain.Policy {
	return &domain.Policy{
		ID:            id,
		PolicyNumber:  "POL-001",
		HolderName:    "Jean Dupont",
		ProductCode:   "IARD-AUTO-RC",
		Status:        domain.StatusDraft,
		EffectiveDate: time.Now().Add(24 * time.Hour),
		ExpiryDate:    time.Now().Add(365 * 24 * time.Hour),
	}
}

func submittedPolicy(id uuid.UUID) *domain.Policy {
	p := draftPolicy(id)
	p.Status = domain.StatusSubmitted
	return p
}

// ─── Submit ──────────────────────────────────────────────────────────────────

func TestSubmit_400_BadUUID(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/bad-uuid/submit", nil), "bad-uuid")

	h.Submit(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmit_404_NotFound(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/submit", nil), id.String())

	h.Submit(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSubmit_409_AlreadySubmitted(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = submittedPolicy(id)
	h := buildTransitionHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/submit", nil), id.String())

	h.Submit(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// ─── Activate ────────────────────────────────────────────────────────────────

func TestActivate_400_BadUUID(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/bad/activate", nil), "bad")

	h.Activate(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestActivate_404_NotFound(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/activate", nil), id.String())

	h.Activate(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestActivate_409_DraftCannotActivate(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = draftPolicy(id)
	h := buildTransitionHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/activate", nil), id.String())

	h.Activate(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// ─── Cancel ──────────────────────────────────────────────────────────────────

func TestCancel_400_BadUUID(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	body, _ := json.Marshal(map[string]string{"reason": "CUSTOMER_REQUEST"})
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/bad/cancel", bytes.NewReader(body)), "bad")
	r.Header.Set("Content-Type", "application/json")

	h.Cancel(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCancel_400_BadJSON(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/cancel", bytes.NewReader([]byte(`{bad`))), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.Cancel(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCancel_422_MissingReason(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"reason": ""})
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/cancel", bytes.NewReader(body)), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.Cancel(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", w.Code)
	}
}

func TestCancel_404_NotFound(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"reason": "CUSTOMER_REQUEST"})
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/cancel", bytes.NewReader(body)), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.Cancel(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCancel_409_DraftCannotCancelActive(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = draftPolicy(id) // draft → can't do CancelActive (EventCancel invalid from draft)
	h := buildTransitionHandler(repo)
	body, _ := json.Marshal(map[string]string{"reason": "CUSTOMER_REQUEST"})
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodPost, "/v1/policies/"+id.String()+"/cancel", bytes.NewReader(body)), id.String())
	r.Header.Set("Content-Type", "application/json")

	h.Cancel(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

// ─── History ─────────────────────────────────────────────────────────────────

func TestHistory_400_BadUUID(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/bad/history", nil), "bad")

	h.History(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHistory_404_NotFound(t *testing.T) {
	h := buildTransitionHandler(newMockRepo())
	id := uuid.New()
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/"+id.String()+"/history", nil), id.String())

	h.History(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHistory_200_OK(t *testing.T) {
	repo := newMockRepo()
	id := uuid.New()
	repo.policies[id] = draftPolicy(id)
	h := buildTransitionHandler(repo)
	w := httptest.NewRecorder()
	r := chiCtxWithID(httptest.NewRequest(http.MethodGet, "/v1/policies/"+id.String()+"/history", nil), id.String())

	h.History(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["history"]; !ok {
		t.Error("response missing 'history' field")
	}
}
