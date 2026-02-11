package tracing

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Initialize sets up the OTEL SDK for tracing
// If collectorEndpoint is empty, exports to stdout for debugging
func Initialize(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Choose exporter based on environment
	collectorEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	var exporter sdktrace.SpanExporter

	if collectorEndpoint == "" {
		// Development: Export to stdout
		log.Println("OTEL_COLLECTOR_ENDPOINT not set, traces will be exported to stdout")

		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(), // Human-readable JSON
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
	} else {
		// Production: Export to OTel Collector
		log.Printf("Tracing exporter configured for OTel Collector: %s", collectorEndpoint)

		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(collectorEndpoint),
			otlptracegrpc.WithInsecure(), // Use TLS in production!
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample 100% in dev
	)

	otel.SetTracerProvider(tracerProvider)

	log.Println("OTEL tracing initialized")

	return tracerProvider.Shutdown, nil
}
