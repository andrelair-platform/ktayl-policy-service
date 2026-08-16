package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	keyfuncv3 "github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	// ContextKeySubject is the context key under which the JWT `sub` claim is stored.
	ContextKeySubject contextKey = "subject"
	// ContextKeyScope is the context key under which the JWT `scope` claim is stored.
	ContextKeyScope contextKey = "scope"
)

// NewJWKSMiddleware creates a JWT-validating middleware backed by a remote JWKS at jwksURL.
// Keys are cached and refreshed every 5 minutes. The provided context controls the lifetime
// of the background refresh goroutine.
func NewJWKSMiddleware(ctx context.Context, jwksURL string) (func(http.Handler) http.Handler, error) {
	noErr := true
	kf, err := keyfuncv3.NewDefaultOverrideCtx(ctx, []string{jwksURL}, keyfuncv3.Override{
		RefreshInterval:           5 * time.Minute,
		NoErrorReturnFirstHTTPReq: &noErr,
	})
	if err != nil {
		return nil, err
	}
	return ValidateJWT(kf), nil
}

// ValidateJWT builds a middleware from a pre-initialised keyfunc.Keyfunc.
// Validates RS256/ES256 signatures, enforces expiry, and stores the `sub` and `scope`
// claims in the request context for downstream handlers and authz middleware.
func ValidateJWT(kf keyfuncv3.Keyfunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeAuthProblem(w, http.StatusUnauthorized, "missing_token", "Authorization: Bearer header required")
				return
			}
			raw := strings.TrimPrefix(authHeader, "Bearer ")

			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(raw, claims, kf.Keyfunc, jwt.WithExpirationRequired())
			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					writeAuthProblem(w, http.StatusUnauthorized, "token_expired", "token has expired")
					return
				}
				writeAuthProblem(w, http.StatusUnauthorized, "invalid_token", "token validation failed")
				return
			}
			if !token.Valid {
				writeAuthProblem(w, http.StatusUnauthorized, "invalid_token", "token is not valid")
				return
			}

			sub, _ := claims.GetSubject()
			scope, _ := claims["scope"].(string)

			ctx := context.WithValue(r.Context(), ContextKeySubject, sub)
			ctx = context.WithValue(ctx, ContextKeyScope, scope)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SubjectFromContext extracts the `sub` claim stored by ValidateJWT.
func SubjectFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ContextKeySubject).(string)
	return s
}

// ScopeFromContext extracts the `scope` claim stored by ValidateJWT.
func ScopeFromContext(ctx context.Context) string {
	s, _ := ctx.Value(ContextKeyScope).(string)
	return s
}

type authProblem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func writeAuthProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(authProblem{title, status, detail})
}
