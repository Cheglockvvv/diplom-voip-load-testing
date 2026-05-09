package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"time"

	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
)

type Server struct {
	workerURL string
	metrics   *metrics.Registry
	client    *http.Client
}

func NewServer(workerURL string, m *metrics.Registry) *Server {
	return &Server{
		workerURL: workerURL,
		metrics:   m,
		client:    &http.Client{Timeout: 10 * time.Second},
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
	scJSON, _ := json.Marshal(sc)
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.workerURL+"/run", bytes.NewReader(scJSON))
	if err != nil {
		http.Error(w, fmt.Sprintf("build request error: %v", err), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker unreachable: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	if len(body) == 0 {
		_, _ = w.Write([]byte("scenario forwarded"))
		return
	}
	_, _ = w.Write(body)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.workerURL+"/stop", nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("build request error: %v", err), http.StatusInternalServerError)
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker unreachable: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	if len(body) == 0 {
		_, _ = w.Write([]byte("stop forwarded"))
		return
	}
	_, _ = w.Write(body)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.workerURL+"/status", nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("build request error: %v", err), http.StatusInternalServerError)
		return
	}
	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("worker unreachable: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	if len(body) == 0 {
		_, _ = w.Write([]byte("{}"))
		return
	}
	_, _ = w.Write(body)
}
