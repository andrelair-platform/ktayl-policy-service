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

// identityMw is a no-op middleware used when auth is disabled.
var identityMw = func(h http.Handler) http.Handler { return h }

// NewRouter builds the chi router. When jwtMw is non-nil it is applied to all /v1/* routes
// and per-endpoint scope guards are enabled. Pass nil to run without authentication (dev/test).
func NewRouter(log *slog.Logger, svc *domain.PolicyService, jwtMw func(http.Handler) http.Handler) http.Handler {
	authEnabled := jwtMw != nil

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(log))

	r.Get("/healthz", healthz)

	ph := handlers.NewPolicyHandler(svc)

	readScope := identityMw
	writeScope := identityMw
	if authEnabled {
		readScope = middleware.RequireScope("policy:read")
		writeScope = middleware.RequireScope("policy:write")
	}

	r.Route("/v1", func(r chi.Router) {
		if authEnabled {
			r.Use(jwtMw)
		}
		r.Route("/policies", func(r chi.Router) {
			r.With(writeScope).Post("/", ph.CreatePolicy)
			r.With(readScope).Get("/", ph.ListPolicies)
			r.With(readScope).Get("/{id}", ph.GetPolicy)
			r.With(writeScope).Put("/{id}", ph.UpdatePolicy)
			r.With(writeScope).Delete("/{id}", ph.CancelPolicy)
		})
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
