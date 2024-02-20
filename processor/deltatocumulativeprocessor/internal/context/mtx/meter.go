package mtx

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

type key struct{}

func From(ctx context.Context) metric.Meter {
	meter, ok := ctx.Value(key{}).(metric.Meter)
	if !ok {
		meter = noop.NewMeterProvider().Meter("otelcol/deltatocumulative")
	}
	return meter
}

func With(ctx context.Context, meter metric.Meter) context.Context {
	return context.WithValue(ctx, key{}, meter)
}
