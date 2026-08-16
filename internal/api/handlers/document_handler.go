package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DocumentHandler struct {
	svc *domain.DocumentService
}

func NewDocumentHandler(svc *domain.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

type attestationResponse struct {
	DocumentID string `json:"document_id"`
	URL        string `json:"url"`
	ExpiresAt  string `json:"expires_at"`
}

// POST /v1/policies/{id}/documents/attestation
func (h *DocumentHandler) GenerateAttestation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "id must be a valid UUID")
		return
	}

	result, err := h.svc.GenerateAttestation(r.Context(), id)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(attestationResponse{
		DocumentID: result.Document.ID.String(),
		URL:        result.URL,
		ExpiresAt:  result.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

type documentListItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

// GET /v1/policies/{id}/documents
func (h *DocumentHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "id must be a valid UUID")
		return
	}

	docs, urls, err := h.svc.ListDocuments(r.Context(), id)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	items := make([]documentListItem, 0, len(docs))
	for i, d := range docs {
		u := ""
		if i < len(urls) {
			u = urls[i]
		}
		items = append(items, documentListItem{
			ID:        d.ID.String(),
			Type:      d.Type,
			URL:       u,
			CreatedAt: d.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"documents": items})
}

func writeDocumentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrPolicyNotFound):
		writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
	case errors.Is(err, domain.ErrDocumentForbidden):
		writeProblem(w, http.StatusConflict, "Conflict", err.Error())
	case errors.Is(err, domain.ErrNoMinIOStore):
		writeProblem(w, http.StatusServiceUnavailable, "Service Unavailable", "document storage not available")
	default:
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
	}
}
