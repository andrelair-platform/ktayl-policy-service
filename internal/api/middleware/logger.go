package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"id", middleware.GetReqID(r.Context()),
			)
			next.ServeHTTP(w, r)
		})
	}
}
