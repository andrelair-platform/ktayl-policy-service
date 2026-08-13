package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthz_StatusOK(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ts := httptest.NewServer(NewRouter(log))
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
	ts := httptest.NewServer(NewRouter(log))
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
	ts := httptest.NewServer(NewRouter(log))
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
