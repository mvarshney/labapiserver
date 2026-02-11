package middleware

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"time"

	"labapiserver/internal/metrics"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ObservabilityMiddleware wraps handlers with metrics collection
func ObservabilityMiddleware(handlerName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			start := time.Now()

			metrics.ActiveConnections.Add(ctx, 1)
			defer metrics.ActiveConnections.Add(ctx, -1)

			// Wrap ResponseWriter to capture status code and response size
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK, body: &bytes.Buffer{}}

			defer func() {
				duration := time.Since(start).Seconds()
				status := strconv.Itoa(rw.statusCode)

				attrs := []attribute.KeyValue{
					attribute.String("handler", handlerName),
					attribute.String("method", r.Method),
					attribute.String("status", status),
				}

				metrics.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
				metrics.RequestDuration.Record(ctx, duration, metric.WithAttributes(
					attribute.String("handler", handlerName),
					attribute.String("method", r.Method),
				))
				metrics.ResponseSize.Record(ctx, int64(rw.body.Len()), metric.WithAttributes(
					attribute.String("handler", handlerName),
				))
			}()

			next.ServeHTTP(rw, r)
		})
	}
}

// RecordError records an error metric for a handler
func RecordError(ctx context.Context, handlerName, errorType string) {
	metrics.ErrorsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("handler", handlerName),
		attribute.String("error_type", errorType),
	))
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}
