package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	mux.Handle("/metrics", s.metrics.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/run", s.handleRun)
	mux.HandleFunc("/stop", s.handleStop)
	return mux
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var sc config.Scenario
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		http.Error(w, fmt.Sprintf("invalid body: %v", err), http.StatusBadRequest)
		return
	}
	scJSON, _ := json.Marshal(sc)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.workerURL+"/run", bytes.NewReader(scJSON))
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.workerURL+"/stop", nil)
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
