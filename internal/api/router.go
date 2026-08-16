package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api/handlers"
	"github.com/andrelair-platform/ktayl-policy-service/internal/api/middleware"
	"github.com/andrelair-platform/ktayl-policy-service/internal/domain"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds the chi router with all middleware and routes wired up.
func NewRouter(log *slog.Logger, svc *domain.PolicyService) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(log))

	r.Get("/healthz", healthz)

	ph := handlers.NewPolicyHandler(svc)
	r.Route("/v1/policies", func(r chi.Router) {
		r.Post("/", ph.CreatePolicy)
		r.Get("/", ph.ListPolicies)
		r.Get("/{id}", ph.GetPolicy)
		r.Put("/{id}", ph.UpdatePolicy)
		r.Delete("/{id}", ph.CancelPolicy)
	})

	return r
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	version := os.Getenv("VERSION")
	if version == "" {
		version = "dev"
	}
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "ktayl-policy-service",
		"version": version,
	})
}
