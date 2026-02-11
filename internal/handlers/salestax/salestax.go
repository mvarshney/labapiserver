package salestax

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"labapiserver/internal/middleware"

	"go.opentelemetry.io/otel/trace"
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
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		if r.Method != http.MethodPost {
			middleware.RecordError(ctx, "salestax", "method_not_allowed")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			middleware.RecordError(ctx, "salestax", "invalid_request_body")
			logger.ErrorContext(ctx, "Failed to decode request body",
				"error", err.Error(),
				"handler", "salestax",
				"trace_id", traceID)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		logger.InfoContext(ctx, "Received sales tax calculation request",
			"handler", "salestax",
			"amount", req.Amount,
			"trace_id", traceID)

		if req.Amount < 0 || taxRate < 0 {
			middleware.RecordError(ctx, "salestax", "invalid_input")
			logger.WarnContext(ctx, "Invalid input values",
				"handler", "salestax",
				"amount", req.Amount,
				"tax_rate", taxRate,
				"trace_id", traceID)
			http.Error(w, "Amount and tax rate must be non-negative", http.StatusBadRequest)
			return
		}

		taxAmount := req.Amount * (taxRate / 100)
		totalAmount := req.Amount + taxAmount

		logger.InfoContext(ctx, "Sales tax calculated successfully",
			"handler", "salestax",
			"amount", req.Amount,
			"tax_rate", taxRate,
			"tax_amount", taxAmount,
			"total_amount", totalAmount,
			"trace_id", traceID)

		resp := Response{
			Amount:      req.Amount,
			TaxRate:     taxRate,
			TaxAmount:   taxAmount,
			TotalAmount: totalAmount,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			middleware.RecordError(ctx, "salestax", "encoding_error")
			logger.ErrorContext(ctx, "Failed to encode response",
				"error", err.Error(),
				"handler", "salestax",
				"trace_id", traceID)
		}
	}
}
