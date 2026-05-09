package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"diplom_code/internal/controller"
	"diplom_code/internal/metrics"
)

func main() {
	addr := env("CONTROLLER_ADDR", ":8080")
	workerURL := env("WORKER_URL", "http://localhost:8081")

	m := metrics.NewRegistry("controller")
	srv := controller.NewServer(workerURL, m)
	server := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("controller listening on %s, worker=%s", addr, workerURL)
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Printf("controller shutting down")
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
