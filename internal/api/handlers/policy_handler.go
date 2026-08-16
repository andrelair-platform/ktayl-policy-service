package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/andrelair-platform/ktayl-policy-service/internal/repository/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

// PolicyHandler handles all /v1/policies routes.
type PolicyHandler struct {
	svc *domain.PolicyService
}

func NewPolicyHandler(svc *domain.PolicyService) *PolicyHandler {
	return &PolicyHandler{svc: svc}
}

// ─── request/response types ─────────────────────────────────────────────────

type createPolicyRequest struct {
	PolicyNumber  string `json:"policy_number"  validate:"required"`
	HolderName    string `json:"holder_name"    validate:"required"`
	ProductCode   string `json:"product_code"   validate:"required"`
	EffectiveDate string `json:"effective_date" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	ExpiryDate    string `json:"expiry_date"    validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
}

type updatePolicyRequest struct {
	HolderName    string `json:"holder_name"    validate:"required"`
	ProductCode   string `json:"product_code"   validate:"required"`
	EffectiveDate string `json:"effective_date" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	ExpiryDate    string `json:"expiry_date"    validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
}

type policyResponse struct {
	ID            string `json:"id"`
	PolicyNumber  string `json:"policy_number"`
	HolderName    string `json:"holder_name"`
	ProductCode   string `json:"product_code"`
	Status        string `json:"status"`
	EffectiveDate string `json:"effective_date"`
	ExpiryDate    string `json:"expiry_date"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type listPoliciesResponse struct {
	Items      []*policyResponse `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

func toResponse(p *domain.Policy) *policyResponse {
	return &policyResponse{
		ID:            p.ID.String(),
		PolicyNumber:  p.PolicyNumber,
		HolderName:    p.HolderName,
		ProductCode:   p.ProductCode,
		Status:        string(p.Status),
		EffectiveDate: p.EffectiveDate.UTC().Format(time.RFC3339),
		ExpiryDate:    p.ExpiryDate.UTC().Format(time.RFC3339),
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// ─── handlers ───────────────────────────────────────────────────────────────

// CreatePolicy POST /v1/policies
func (h *PolicyHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid JSON body")
		return
	}
	if err := validate.Struct(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Validation Error", err.Error())
		return
	}

	eff, err := time.Parse(time.RFC3339, req.EffectiveDate)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid effective_date format")
		return
	}
	exp, err := time.Parse(time.RFC3339, req.ExpiryDate)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid expiry_date format")
		return
	}

	p := &domain.Policy{
		PolicyNumber:  req.PolicyNumber,
		HolderName:    req.HolderName,
		ProductCode:   req.ProductCode,
		EffectiveDate: eff,
		ExpiryDate:    exp,
	}
	if err := h.svc.Create(r.Context(), p); err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidDateRange),
			errors.Is(err, domain.ErrEmptyPolicyNumber),
			errors.Is(err, domain.ErrEmptyHolderName),
			errors.Is(err, domain.ErrEmptyProductCode):
			writeProblem(w, http.StatusBadRequest, "Validation Error", err.Error())
		case errors.Is(err, domain.ErrPolicyNumberTaken):
			writeProblem(w, http.StatusConflict, "Conflict", err.Error())
		default:
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

// GetPolicy GET /v1/policies/{id}
func (h *PolicyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid policy id")
		return
	}

	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPolicyNotFound) {
			writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

// ListPolicies GET /v1/policies
func (h *PolicyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := domain.ListParams{
		Cursor: q.Get("cursor"),
		Limit:  20,
	}
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			params.Limit = n
		}
	}
	if s := q.Get("status"); s != "" {
		st := domain.PolicyStatus(s)
		params.Status = &st
	}

	policies, err := h.svc.List(r.Context(), params)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		return
	}

	items := make([]*policyResponse, 0, len(policies))
	for _, p := range policies {
		items = append(items, toResponse(p))
	}

	resp := listPoliciesResponse{Items: items}
	if len(policies) == params.Limit {
		last := policies[len(policies)-1]
		resp.NextCursor = postgres.EncodeCursor(last.CreatedAt, last.ID)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdatePolicy PUT /v1/policies/{id}
func (h *PolicyHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid policy id")
		return
	}

	var req updatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid JSON body")
		return
	}
	if err := validate.Struct(req); err != nil {
		writeProblem(w, http.StatusBadRequest, "Validation Error", err.Error())
		return
	}

	eff, err := time.Parse(time.RFC3339, req.EffectiveDate)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid effective_date format")
		return
	}
	exp, err := time.Parse(time.RFC3339, req.ExpiryDate)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid expiry_date format")
		return
	}

	upd := &domain.Policy{
		HolderName:    req.HolderName,
		ProductCode:   req.ProductCode,
		EffectiveDate: eff,
		ExpiryDate:    exp,
	}
	p, err := h.svc.Update(r.Context(), id, upd)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPolicyNotFound):
			writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
		case errors.Is(err, domain.ErrInvalidDateRange):
			writeProblem(w, http.StatusBadRequest, "Validation Error", err.Error())
		default:
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toResponse(p))
}

// CancelPolicy DELETE /v1/policies/{id}
func (h *PolicyHandler) CancelPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Bad Request", "invalid policy id")
		return
	}

	if err := h.svc.Cancel(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrPolicyNotFound):
			writeProblem(w, http.StatusNotFound, "Not Found", "policy not found")
		case errors.Is(err, domain.ErrCannotCancel):
			writeProblem(w, http.StatusConflict, "Conflict", err.Error())
		default:
			writeProblem(w, http.StatusInternalServerError, "Internal Server Error", "unexpected error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
