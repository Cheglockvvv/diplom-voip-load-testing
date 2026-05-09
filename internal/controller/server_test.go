package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"diplom_code/api/control"
	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
	"google.golang.org/grpc"
)

func TestHandleStatusProxy(t *testing.T) {
	s := NewServer(&fakeControlClient{
		status: &control.StatusResponse{
			RunID: "run-1",
			State: "running",
		},
	}, metrics.NewRegistry("controller_test"))
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
	s := NewServer(&fakeControlClient{}, metrics.NewRegistry("controller_test"))
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

type fakeControlClient struct {
	start  *control.StartScenarioResponse
	stop   *control.StopScenarioResponse
	status *control.StatusResponse
}

func (f *fakeControlClient) StartScenario(context.Context, *control.StartScenarioRequest, ...grpc.CallOption) (*control.StartScenarioResponse, error) {
	if f.start != nil {
		return f.start, nil
	}
	return &control.StartScenarioResponse{RunID: "run-1", Message: "ok"}, nil
}

func (f *fakeControlClient) StopScenario(context.Context, *control.StopScenarioRequest, ...grpc.CallOption) (*control.StopScenarioResponse, error) {
	if f.stop != nil {
		return f.stop, nil
	}
	return &control.StopScenarioResponse{RunID: "run-1", Message: "stopped"}, nil
}

func (f *fakeControlClient) GetStatus(context.Context, *control.StatusRequest, ...grpc.CallOption) (*control.StatusResponse, error) {
	if f.status != nil {
		return f.status, nil
	}
	return &control.StatusResponse{State: "idle"}, nil
}

func (f *fakeControlClient) StreamStatus(context.Context, *control.StreamStatusRequest, ...grpc.CallOption) (control.ControlService_StreamStatusClient, error) {
	return nil, nil
}
