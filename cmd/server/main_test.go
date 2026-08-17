package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

type noopRepo struct{}

var _ domain.PolicyRepository = noopRepo{}

func (noopRepo) Create(_ context.Context, _ *domain.Policy) error { return nil }
func (noopRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (noopRepo) GetByPolicyNumber(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (noopRepo) Update(_ context.Context, _ *domain.Policy) error { return nil }
func (noopRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Policy, error) {
	return nil, nil
}

func TestHealthz(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := domain.NewPolicyService(noopRepo{}, domain.NullAuditLog(), nil)
	ts := httptest.NewServer(api.NewRouter(log, svc, nil, nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
