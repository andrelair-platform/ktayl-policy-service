package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/google/uuid"
)

type routerTestRepo struct{}

var _ domain.PolicyRepository = (*routerTestRepo)(nil)

func (r *routerTestRepo) Create(_ context.Context, _ *domain.Policy) error { return nil }
func (r *routerTestRepo) GetByID(_ context.Context, _ uuid.UUID) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (r *routerTestRepo) GetByPolicyNumber(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, domain.ErrNotFound
}
func (r *routerTestRepo) Update(_ context.Context, _ *domain.Policy) error { return nil }
func (r *routerTestRepo) List(_ context.Context, _ domain.ListParams) ([]*domain.Policy, error) {
	return nil, nil
}

func newTestService() *domain.PolicyService {
	return domain.NewPolicyService(&routerTestRepo{}, domain.NullAuditLog(), nil)
}

func TestHealthz_StatusOK(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ts := httptest.NewServer(NewRouter(log, newTestService(), nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
}

func TestHealthz_Body(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ts := httptest.NewServer(NewRouter(log, newTestService(), nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf(`expected status "ok", got %q`, body["status"])
	}
	if body["service"] != "ktayl-policy-service" {
		t.Fatalf(`expected service "ktayl-policy-service", got %q`, body["service"])
	}
}

func TestNotFound(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ts := httptest.NewServer(NewRouter(log, newTestService(), nil))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatalf("GET /nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
