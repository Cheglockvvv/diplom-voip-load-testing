package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"diplom_code/api/control"
	"diplom_code/internal/controller"
	"diplom_code/internal/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := env("CONTROLLER_ADDR", ":8080")
	workerGRPCAddr := env("WORKER_GRPC_ADDR", "localhost:19091")

	grpcConn, err := grpc.Dial(
		workerGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(control.JSONCodec{})),
	)
	if err != nil {
		log.Fatalf("failed to connect to worker grpc: %v", err)
	}
	defer grpcConn.Close()

	m := metrics.NewRegistry("controller")
	srv := controller.NewServer(control.NewControlServiceClient(grpcConn), m)
	server := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("controller listening on %s, worker grpc=%s", addr, workerGRPCAddr)
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
