package delta

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/streams"
	"go.opentelemetry.io/otel/metric"
)

func ObserveMap[T any](items streams.Map[T]) metric.Int64Callback {
	return func(ctx context.Context, io metric.Int64Observer) error {
		io.Observe(int64(items.Len()))
		return nil
	}
}
