package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	metricshandler "labapiserver/internal/handlers/metrics"
	"labapiserver/internal/handlers/salestax"
	"labapiserver/internal/metrics"
	"labapiserver/internal/middleware"
	"labapiserver/internal/tracing"
	"labapiserver/pkg/health"
)

func main() {
	ctx := context.Background()

	// Initialize OTEL tracing
	tracingShutdown, err := tracing.Initialize(ctx, "labapiserver")
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracingShutdown(ctx); err != nil {
			log.Printf("Error shutting down tracing: %v", err)
		}
	}()

	// Initialize OTEL metrics (push to collector)
	collectorEndpoint := os.Getenv("OTEL_COLLECTOR_ENDPOINT")
	if collectorEndpoint == "" {
		log.Println("OTEL_COLLECTOR_ENDPOINT not set, metrics will be exported to stdout")
		collectorEndpoint = "stdout"
	}

	shutdown, err := metrics.Initialize(ctx, collectorEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Printf("Error shutting down metrics: %v", err)
		}
	}()

	// Start memory monitoring
	if err := metricshandler.StartMemoryMonitoring(); err != nil {
		log.Printf("Failed to start memory monitoring: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", health.Handler())
	mux.Handle("/salestax",
		middleware.TracingMiddleware("salestax")(
			middleware.ObservabilityMiddleware("salestax")(salestax.Handler())))

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
