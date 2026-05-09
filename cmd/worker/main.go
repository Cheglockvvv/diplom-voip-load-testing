package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"diplom_code/api/control"
	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
	"diplom_code/internal/worker"
	"google.golang.org/grpc"
)

func main() {
	addr := env("WORKER_ADDR", ":8081")
	grpcAddr := env("WORKER_GRPC_ADDR", ":19091")
	reg := metrics.NewRegistry("worker")
	runner := worker.NewRunner(reg)

	var mu sync.Mutex
	var cancel context.CancelFunc
	var running int32
	var runSeq uint64
	currentRunID := ""

	launchScenario := func(sc config.Scenario) string {
		mu.Lock()
		if cancel != nil {
			cancel()
		}
		ctx, c := context.WithCancel(context.Background())
		cancel = c
		currentRunID = fmt.Sprintf("run-%d", atomic.AddUint64(&runSeq, 1))
		runID := currentRunID
		atomic.StoreInt32(&running, 1)
		mu.Unlock()

		go func() {
			defer func() {
				atomic.StoreInt32(&running, 0)
				mu.Lock()
				if currentRunID == runID {
					currentRunID = ""
				}
				mu.Unlock()
			}()
			if err := runner.Run(ctx, sc); err != nil {
				log.Printf("scenario stopped with error: %v", err)
			}
		}()
		return runID
	}

	stopScenario := func() string {
		mu.Lock()
		defer mu.Unlock()
		runID := currentRunID
		if cancel != nil {
			cancel()
			cancel = nil
			currentRunID = ""
		}
		return runID
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", reg.Handler())
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		runID := currentRunID
		mu.Unlock()
		writeJSON(w, http.StatusOK, statusResponse{
			RunID: runID,
			State: currentState(atomic.LoadInt32(&running)),
		})
	})
	mux.HandleFunc("POST /run", func(w http.ResponseWriter, r *http.Request) {
		var sc config.Scenario
		if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := sc.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		runID := launchScenario(sc)
		writeJSON(w, http.StatusAccepted, runResponse{
			RunID:   runID,
			State:   "started",
			Message: "scenario started",
		})
	})
	mux.HandleFunc("POST /stop", func(w http.ResponseWriter, r *http.Request) {
		runID := stopScenario()
		writeJSON(w, http.StatusOK, runResponse{
			RunID:   runID,
			State:   "stopped",
			Message: "stopped",
		})
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("worker grpc listen failed: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.ForceServerCodec(control.JSONCodec{}))
	control.RegisterControlServiceServer(grpcServer, &workerControlService{
		getStatus: func() statusResponse {
			mu.Lock()
			runID := currentRunID
			mu.Unlock()
			return statusResponse{
				RunID: runID,
				State: currentState(atomic.LoadInt32(&running)),
			}
		},
		launchScenario: launchScenario,
		stopScenario:   stopScenario,
	})

	log.Printf("worker listening on %s", addr)
	log.Printf("worker grpc listening on %s", grpcAddr)
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("worker grpc serve failed: %v", err)
		}
	}()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Printf("worker shutting down")
		grpcServer.GracefulStop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type runResponse struct {
	RunID   string `json:"run_id,omitempty"`
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

type statusResponse struct {
	RunID string `json:"run_id,omitempty"`
	State string `json:"state"`
}

type workerControlService struct {
	control.UnimplementedControlServiceServer
	getStatus      func() statusResponse
	launchScenario func(sc config.Scenario) string
	stopScenario   func() string
}

func (s *workerControlService) StartScenario(_ context.Context, req *control.StartScenarioRequest) (*control.StartScenarioResponse, error) {
	var sc config.Scenario
	if err := json.Unmarshal([]byte(req.ScenarioJSON), &sc); err != nil {
		return nil, err
	}
	if err := sc.Validate(); err != nil {
		return nil, err
	}
	runID := s.launchScenario(sc)
	return &control.StartScenarioResponse{
		RunID:   runID,
		Message: "scenario started",
	}, nil
}

func (s *workerControlService) StopScenario(_ context.Context, _ *control.StopScenarioRequest) (*control.StopScenarioResponse, error) {
	runID := s.stopScenario()
	return &control.StopScenarioResponse{
		RunID:   runID,
		Message: "stopped",
	}, nil
}

func (s *workerControlService) GetStatus(_ context.Context, _ *control.StatusRequest) (*control.StatusResponse, error) {
	st := s.getStatus()
	return &control.StatusResponse{
		RunID: st.RunID,
		State: st.State,
	}, nil
}

func (s *workerControlService) StreamStatus(req *control.StreamStatusRequest, stream control.ControlService_StreamStatusServer) error {
	_ = req
	sendCurrent := func() error {
		st := s.getStatus()
		return stream.Send(&control.StatusEvent{
			RunID:    st.RunID,
			State:    st.State,
			Details:  "status snapshot",
			UnixTime: time.Now().Unix(),
		})
	}
	if err := sendCurrent(); err != nil {
		return err
	}
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			if err := sendCurrent(); err != nil {
				return err
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(v)
}

func currentState(v int32) string {
	if v == 1 {
		return "running"
	}
	return "idle"
}
