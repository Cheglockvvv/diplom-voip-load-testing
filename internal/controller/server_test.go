package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
)

func TestHandleStatusProxy(t *testing.T) {
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"run_id":"run-1","state":"running"}`))
	}))
	defer worker.Close()

	s := NewServer(worker.URL, metrics.NewRegistry("controller_test"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected non-empty body")
	}
}

func TestHandleRunRejectsInvalidScenario(t *testing.T) {
	worker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer worker.Close()

	s := NewServer(worker.URL, metrics.NewRegistry("controller_test"))
	invalid := config.Scenario{
		Name:            "invalid",
		Mode:            "registration_storm",
		Users:           10,
		CPS:             10,
		DurationSeconds: 30,
		Target: config.TargetConfig{
			SIPPort: 5060,
		},
	}
	body, _ := json.Marshal(invalid)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/run", bytes.NewReader(body))
	s.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid scenario, got %d", rec.Code)
	}
}
