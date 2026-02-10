package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

var (
	meter metric.Meter

	// Counters
	RequestsTotal metric.Int64Counter
	ErrorsTotal   metric.Int64Counter

	// Gauges (implemented as UpDownCounters)
	ActiveConnections metric.Int64UpDownCounter

	// Histograms
	RequestDuration metric.Float64Histogram
	ResponseSize    metric.Int64Histogram
)

// Initialize sets up the OTEL SDK and creates all metric instruments
func Initialize(ctx context.Context, endpoint string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("labapiserver"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var exporter sdkmetric.Exporter

	if endpoint == "stdout" {
		// Use stdout exporter
		exporter, err = stdoutmetric.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
		}
		log.Println("OTEL metrics initialized, exporting to stdout")
	} else {
		// Use OTLP exporter
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(), // Use WithTLSCredentials in production
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}
		log.Printf("OTEL metrics initialized, pushing to collector at %s", endpoint)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(10*time.Second),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)
	meter = meterProvider.Meter("labapiserver")

	// Initialize counters
	RequestsTotal, err = meter.Int64Counter(
		"http.requests.total",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	ErrorsTotal, err = meter.Int64Counter(
		"http.errors.total",
		metric.WithDescription("Total number of HTTP errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize up-down counter for active connections
	ActiveConnections, err = meter.Int64UpDownCounter(
		"http.active_connections",
		metric.WithDescription("Number of active HTTP connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize histograms
	RequestDuration, err = meter.Float64Histogram(
		"http.request.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	ResponseSize, err = meter.Int64Histogram(
		"http.response.size",
		metric.WithDescription("HTTP response size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	return meterProvider.Shutdown, nil
}

// GetMeter returns the global meter instance
func GetMeter() metric.Meter {
	return meter
}
