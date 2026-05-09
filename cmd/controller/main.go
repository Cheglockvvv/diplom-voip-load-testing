package main

import (
	"log"
	"net/http"
	"os"

	"diplom_code/internal/controller"
	"diplom_code/internal/metrics"
)

func main() {
	addr := env("CONTROLLER_ADDR", ":8080")
	workerURL := env("WORKER_URL", "http://localhost:8081")

	m := metrics.NewRegistry("controller")
	srv := controller.NewServer(workerURL, m)
	log.Printf("controller listening on %s, worker=%s", addr, workerURL)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
