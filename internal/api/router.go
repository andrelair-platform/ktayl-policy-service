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
// docSvc may be nil when MinIO is not configured (document endpoints return 503).
// svc.NATSConnected() is surfaced in /healthz.
func NewRouter(log *slog.Logger, svc *domain.PolicyService, docSvc *domain.DocumentService, jwtMw func(http.Handler) http.Handler) http.Handler {
	authEnabled := jwtMw != nil

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(log))

	r.Get("/healthz", makeHealthz(svc))

	ph := handlers.NewPolicyHandler(svc)
	th := handlers.NewTransitionHandler(svc)

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

			// State machine transitions (S004)
			r.With(writeScope).Post("/{id}/submit", th.Submit)
			r.With(writeScope).Post("/{id}/activate", th.Activate)
			r.With(writeScope).Post("/{id}/cancel", th.Cancel)
			r.With(readScope).Get("/{id}/history", th.History)

			// Document generation (S005)
			if docSvc != nil {
				dh := handlers.NewDocumentHandler(docSvc)
				r.With(writeScope).Post("/{id}/documents/attestation", dh.GenerateAttestation)
				r.With(readScope).Get("/{id}/documents", dh.ListDocuments)
			}
		})
	})

	return r
}

func makeHealthz(svc *domain.PolicyService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		version := os.Getenv("VERSION")
		if version == "" {
			version = "dev"
		}
		nats := "disconnected"
		if svc != nil && svc.NATSConnected() {
			nats = "connected"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "ktayl-policy-service",
			"version": version,
			"nats":    nats,
		})
	}
}
