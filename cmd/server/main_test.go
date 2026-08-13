package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/andrelair-platform/ktayl-policy-service/internal/api"
)

func TestHealthz(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ts := httptest.NewServer(api.NewRouter(log))
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
