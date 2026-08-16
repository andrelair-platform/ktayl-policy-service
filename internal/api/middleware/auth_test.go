package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	keyfuncv3 "github.com/MicahParks/keyfunc/v3"
	"github.com/andrelair-platform/ktayl-policy-service/internal/api/middleware"
	"github.com/golang-jwt/jwt/v5"
)

const testKID = "test-key-2048"

// jwtTestEnv holds shared test infrastructure for the middleware tests.
type jwtTestEnv struct {
	privKey *rsa.PrivateKey
	kf      keyfuncv3.Keyfunc
	cleanup func()
}

func newJWTTestEnv(t *testing.T) *jwtTestEnv {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	serverStore := jwkset.NewMemoryStorage()
	jwk, err := jwkset.NewJWKFromKey(privKey, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{KID: testKID},
	})
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	if err := serverStore.KeyWrite(ctx, jwk); err != nil {
		t.Fatalf("write JWK to store: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		raw, err := serverStore.JSONPrivate(ctx)
		if err != nil {
			http.Error(w, "jwks error", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(raw)
	}))

	clientStore, err := jwkset.NewDefaultHTTPClient([]string{server.URL})
	if err != nil {
		cancel()
		server.Close()
		t.Fatalf("create client store: %v", err)
	}
	kf, err := keyfuncv3.New(keyfuncv3.Options{Ctx: ctx, Storage: clientStore})
	if err != nil {
		cancel()
		server.Close()
		t.Fatalf("create keyfunc: %v", err)
	}

	return &jwtTestEnv{
		privKey: privKey,
		kf:      kf,
		cleanup: func() { cancel(); server.Close() },
	}
}

// sign creates a signed JWT with the given claims and the test RSA key.
func (e *jwtTestEnv) sign(claims jwt.MapClaims) string {
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t.Header["kid"] = testKID
	signed, err := t.SignedString(e.privKey)
	if err != nil {
		panic("sign token: " + err.Error())
	}
	return signed
}

func validClaims(scope string) jwt.MapClaims {
	return jwt.MapClaims{
		"sub":   "svc-ktayl-portal",
		"scope": scope,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
}

// — JWT validation tests (6 cases) ——————————————————————————————————————

func TestValidateJWT(t *testing.T) {
	env := newJWTTestEnv(t)
	defer env.cleanup()

	// Generate a separate key to produce an invalid signature.
	wrongKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	signWithWrongKey := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = testKID
		signed, _ := tok.SignedString(wrongKey)
		return signed
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantSub    string
		wantScope  string
	}{
		{
			name:       "valid token — subject and scope stored in context",
			authHeader: "Bearer " + env.sign(validClaims("policy:read policy:write")),
			wantStatus: http.StatusOK,
			wantSub:    "svc-ktayl-portal",
			wantScope:  "policy:read policy:write",
		},
		{
			name:       "missing Authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Basic auth scheme (not Bearer)",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed JWT (not valid token string)",
			authHeader: "Bearer not.a.token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "expired token",
			authHeader: "Bearer " + env.sign(jwt.MapClaims{
				"sub":   "svc",
				"scope": "policy:read",
				"exp":   time.Now().Add(-time.Hour).Unix(),
			}),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token signed with wrong key",
			authHeader: "Bearer " + signWithWrongKey(validClaims("policy:read")),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := middleware.ValidateJWT(env.kf)
			var gotSub, gotScope string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotSub = middleware.SubjectFromContext(r.Context())
				gotScope = middleware.ScopeFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/v1/policies", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()
			mw(next).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if tc.wantStatus == http.StatusOK {
				if gotSub != tc.wantSub {
					t.Errorf("sub = %q, want %q", gotSub, tc.wantSub)
				}
				if gotScope != tc.wantScope {
					t.Errorf("scope = %q, want %q", gotScope, tc.wantScope)
				}
			}
		})
	}
}

// — Scope authz tests (4 cases) ——————————————————————————————————————

func TestRequireScope(t *testing.T) {
	env := newJWTTestEnv(t)
	defer env.cleanup()

	tests := []struct {
		name          string
		tokenScope    string
		requiredScope string
		wantStatus    int
	}{
		{
			name:          "has exactly the required scope",
			tokenScope:    "policy:read",
			requiredScope: "policy:read",
			wantStatus:    http.StatusOK,
		},
		{
			name:          "has multiple scopes including the required one",
			tokenScope:    "policy:read policy:write openid",
			requiredScope: "policy:write",
			wantStatus:    http.StatusOK,
		},
		{
			name:          "has read but endpoint requires write",
			tokenScope:    "policy:read",
			requiredScope: "policy:write",
			wantStatus:    http.StatusForbidden,
		},
		{
			name:          "empty scope claim",
			tokenScope:    "",
			requiredScope: "policy:read",
			wantStatus:    http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jwtMw := middleware.ValidateJWT(env.kf)
			scopeMw := middleware.RequireScope(tc.requiredScope)
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			token := env.sign(validClaims(tc.tokenScope))
			req := httptest.NewRequest(http.MethodGet, "/v1/policies", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()
			jwtMw(scopeMw(next)).ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}
