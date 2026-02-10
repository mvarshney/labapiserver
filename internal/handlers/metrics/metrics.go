package metrics

import (
	"context"
	"runtime"

	appmetrics "labapiserver/internal/metrics"

	"go.opentelemetry.io/otel/metric"
)

// StartMemoryMonitoring registers a callback to collect memory metrics
func StartMemoryMonitoring() error {
	meter := appmetrics.GetMeter()
	if meter == nil {
		return nil // Metrics not initialized yet
	}

	memoryGauge, err := meter.Int64ObservableGauge(
		"runtime.memory.heap.alloc",
		metric.WithDescription("Bytes of allocated heap objects"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		o.ObserveInt64(memoryGauge, int64(m.Alloc))
		return nil
	}, memoryGauge)

	return err
}
