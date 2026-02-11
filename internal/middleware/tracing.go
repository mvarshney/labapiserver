package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware(handlerName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer("labapiserver")
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			ctx, span := tracer.Start(ctx, handlerName,
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPRoute(r.URL.Path),
					semconv.HTTPTarget(r.URL.String()),
					attribute.String("handler", handlerName),
				),
			)
			defer span.End()

			w.Header().Set("traceparent", span.SpanContext().TraceID().String())

			ww := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r.WithContext(ctx))

			span.SetAttributes(semconv.HTTPStatusCode(ww.statusCode))
			if ww.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(ww.statusCode))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
