package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api/middleware"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TransitionHandler struct {
	svc *domain.PolicyService
}

func NewTransitionHandler(svc *domain.PolicyService) *TransitionHandler {
	return &TransitionHandler{svc: svc}
}

// POST /v1/policies/{id}/submit — DRAFT → SUBMITTED
func (h *TransitionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}
	actor := middleware.SubjectFromContext(r.Context())
	p, err := h.svc.Submit(r.Context(), id, actor)
	if err != nil {
		writeTransitionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

// POST /v1/policies/{id}/activate — SUBMITTED → ACTIVE
func (h *TransitionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}
	actor := middleware.SubjectFromContext(r.Context())
	p, err := h.svc.Activate(r.Context(), id, actor)
	if err != nil {
		writeTransitionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

type cancelRequest struct {
	Reason        string    `json:"reason"`
	EffectiveDate time.Time `json:"effective_date"`
}

// POST /v1/policies/{id}/cancel — ACTIVE → CANCELLED
func (h *TransitionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}
	var body cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "request body must be valid JSON")
		return
	}
	if body.Reason == "" {
		writeProblem(w, http.StatusUnprocessableEntity, "Unprocessable Entity", "reason is required for cancellation")
		return
	}
	actor := middleware.SubjectFromContext(r.Context())
	p, err := h.svc.CancelActive(r.Context(), id, actor, body.Reason)
	if err != nil {
		writeTransitionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

type auditLogResponse struct {
	ID         string `json:"id"`
	PolicyID   string `json:"policy_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	ActorID    string `json:"actor_id"`
	Reason     string `json:"reason,omitempty"`
	OccurredAt string `json:"occurred_at"`
}

// GET /v1/policies/{id}/history — ordered DORA audit trail
func (h *TransitionHandler) History(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}
	logs, err := h.svc.ListHistory(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPolicyNotFound) {
			writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		return
	}

	resp := make([]auditLogResponse, 0, len(logs))
	for _, l := range logs {
		resp = append(resp, auditLogResponse{
			ID:         l.ID.String(),
			PolicyID:   l.PolicyID.String(),
			FromStatus: string(l.FromStatus),
			ToStatus:   string(l.ToStatus),
			ActorID:    l.ActorID,
			Reason:     l.Reason,
			OccurredAt: l.OccurredAt.UTC().Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"history": resp})
}

func parseUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := chi.URLParam(r, "id")
	id, err := uuid.Parse(raw)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func writeTransitionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrPolicyNotFound):
		writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
	case errors.Is(err, domain.ErrInvalidTransition):
		writeProblem(w, http.StatusConflict, "Conflict", err.Error())
	case errors.Is(err, domain.ErrReasonRequired):
		writeProblem(w, http.StatusUnprocessableEntity, "Unprocessable Entity", err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
	}
}
