package salestax

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"labapiserver/internal/metrics"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Request struct {
	Amount float64 `json:"amount"`
}

type Response struct {
	Amount      float64 `json:"amount"`
	TaxRate     float64 `json:"tax_rate"`
	TaxAmount   float64 `json:"tax_amount"`
	TotalAmount float64 `json:"total_amount"`
}

func Handler() http.HandlerFunc {
	const taxRate = 7.5

	return func(w http.ResponseWriter, r *http.Request) {
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
				attribute.String("handler", "salestax"),
				attribute.String("method", r.Method),
				attribute.String("status", status),
			}

			metrics.RequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
			metrics.RequestDuration.Record(ctx, duration, metric.WithAttributes(
				attribute.String("handler", "salestax"),
				attribute.String("method", r.Method),
			))
			metrics.ResponseSize.Record(ctx, int64(rw.body.Len()), metric.WithAttributes(
				attribute.String("handler", "salestax"),
			))
		}()

		if r.Method != http.MethodPost {
			recordError(ctx, "method_not_allowed")
			rw.statusCode = http.StatusMethodNotAllowed
			http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			recordError(ctx, "invalid_request_body")
			rw.statusCode = http.StatusBadRequest
			http.Error(rw, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Amount < 0 || taxRate < 0 {
			recordError(ctx, "invalid_input")
			rw.statusCode = http.StatusBadRequest
			http.Error(rw, "Amount and tax rate must be non-negative", http.StatusBadRequest)
			return
		}

		taxAmount := req.Amount * (taxRate / 100)
		totalAmount := req.Amount + taxAmount

		resp := Response{
			Amount:      req.Amount,
			TaxRate:     taxRate,
			TaxAmount:   taxAmount,
			TotalAmount: totalAmount,
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(resp); err != nil {
			recordError(ctx, "encoding_error")
		}
	}
}

func recordError(ctx context.Context, errorType string) {
	metrics.ErrorsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("handler", "salestax"),
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
