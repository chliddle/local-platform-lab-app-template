package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/chliddle/local-platform-lab-app-template/internal/handlers"
	"github.com/chliddle/local-platform-lab-app-template/internal/middleware"
)

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	mux := http.NewServeMux()
	// Fault sits inside Metrics (not outside) so an injected failure's 500
	// still gets recorded by http_requests_total -- see
	// internal/middleware/fault.go. Only "/" carries it: /health, /ready,
	// and /version stay real, so probes/CI keep working and only actual
	// user-facing traffic sees the injected fault.
	mux.HandleFunc("/", middleware.Metrics("/", middleware.Fault(handlers.Root)))
	mux.HandleFunc("/health", middleware.Metrics("/health", handlers.Health))
	mux.HandleFunc("/ready", middleware.Metrics("/ready", handlers.Ready))
	mux.HandleFunc("/version", middleware.Metrics("/version", handlers.Version))
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("hello-world listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("shutting down")
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}
