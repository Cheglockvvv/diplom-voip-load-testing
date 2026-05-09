package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"

	"diplom_code/internal/config"
	"diplom_code/internal/metrics"
	"diplom_code/internal/worker"
)

func main() {
	addr := env("WORKER_ADDR", ":8081")
	reg := metrics.NewRegistry("worker")
	runner := worker.NewRunner(reg)

	var mu sync.Mutex
	var cancel context.CancelFunc
	var running int32

	mux := http.NewServeMux()
	mux.Handle("/metrics", reg.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, _ *http.Request) {
		state := "idle"
		if atomic.LoadInt32(&running) == 1 {
			state = "running"
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("{\"state\":\"%s\"}", state)))
	})
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var sc config.Scenario
		if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := sc.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		if cancel != nil {
			cancel()
		}
		ctx, c := context.WithCancel(context.Background())
		cancel = c
		atomic.StoreInt32(&running, 1)
		mu.Unlock()

		go func() {
			defer atomic.StoreInt32(&running, 0)
			if err := runner.Run(ctx, sc); err != nil {
				log.Printf("scenario stopped with error: %v", err)
			}
		}()

		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("scenario started"))
	})
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		if cancel != nil {
			cancel()
			cancel = nil
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stopped"))
	})

	log.Printf("worker listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
