package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"diplom_code/api/control"
	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
	"google.golang.org/grpc"
)

type Server struct {
	workerClient control.ControlServiceClient
	metrics      *metrics.Registry
	grpcOpts     []grpc.CallOption
}

func NewServer(workerClient control.ControlServiceClient, m *metrics.Registry) *Server {
	return &Server{
		workerClient: workerClient,
		metrics:      m,
		grpcOpts:     []grpc.CallOption{grpc.ForceCodec(control.JSONCodec{})},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", s.metrics.Handler())
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /run", s.handleRun)
	mux.HandleFunc("POST /stop", s.handleStop)
	mux.HandleFunc("GET /status", s.handleStatus)
	return mux
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var sc config.Scenario
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, fmt.Sprintf("invalid body: %v", err), http.StatusBadRequest)
		return
	}
	if err := sc.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scJSON, err := json.Marshal(sc)
	if err != nil {
		http.Error(w, fmt.Sprintf("marshal scenario error: %v", err), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	resp, err := s.workerClient.StartScenario(ctx, &control.StartScenarioRequest{ScenarioJSON: string(scJSON)}, s.grpcOpts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker start failed: %v", err), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	resp, err := s.workerClient.StopScenario(ctx, &control.StopScenarioRequest{}, s.grpcOpts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker stop failed: %v", err), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	resp, err := s.workerClient.GetStatus(ctx, &control.StatusRequest{}, s.grpcOpts...)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker status failed: %v", err), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
