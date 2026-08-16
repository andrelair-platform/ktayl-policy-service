package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api/handlers"
	"github.com/go-chi/chi/v5"
)

func setupDocumentRouter() *chi.Mux {
	h := handlers.NewDocumentHandler(nil)
	r := chi.NewRouter()
	r.Post("/v1/policies/{id}/documents/attestation", h.GenerateAttestation)
	r.Get("/v1/policies/{id}/documents", h.ListDocuments)
	return r
}

func TestGenerateAttestation_400_InvalidUUID(t *testing.T) {
	rtr := setupDocumentRouter()
	req := httptest.NewRequest(http.MethodPost, "/v1/policies/not-a-uuid/documents/attestation", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestListDocuments_400_InvalidUUID(t *testing.T) {
	rtr := setupDocumentRouter()
	req := httptest.NewRequest(http.MethodGet, "/v1/policies/not-a-uuid/documents", nil)
	w := httptest.NewRecorder()
	rtr.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}
