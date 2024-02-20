// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package delta // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/delta"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/otel/metric"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/context/mtx"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/data"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/streams"
)

type Options struct {
	MaxStale time.Duration
}

type Metrics struct {
	Streams metric.Int64ObservableCounter
	Samples metric.Int64Counter
}

func metrics(meter metric.Meter) (Metrics, error) {
	errs := new(error)
	var (
		tryo = try[metric.Int64ObservableCounter](errs)
		tryi = try[metric.Int64Counter](errs)
	)

	metrics := Metrics{
		Streams: tryo(meter.Int64ObservableCounter("streams")),
		Samples: tryi(meter.Int64Counter("samples")),
	}

	return metrics, *errs
}

func try[T any](errs *error) func(T, error) T {
	return func(v T, err error) T {
		*errs = errors.Join(*errs, err)
		return v
	}
}

func construct[D data.Point[D]](ctx context.Context, opts Options) streams.Aggregator[D] {
	dps := streams.EmptyMap[D]()
	if opts.MaxStale > 0 {
		dps = streams.ExpireAfter(ctx, dps, opts.MaxStale)
	}

	meter := mtx.From(ctx)
	metrics, err := metrics(meter)
	if err != nil {
	}
	// TODO

	var metrics Metrics
	meter := mtx.From(ctx)
	meter.Int64ObservableCounter("tracked_streams", ObserveMap(dps))
	samples, err := meter.Int64Counter("samples")

	var (
		acc  = Accumulator[D]{dps: dps}
		lock = Lock[D]{next: &acc}
	)
	return &lock
}

func Numbers(ctx context.Context, opts Options) streams.Aggregator[data.Number] {
	return construct[data.Number](ctx, opts)
}

func Histograms(ctx context.Context, opts Options) streams.Aggregator[data.Histogram] {
	return construct[data.Histogram](ctx, opts)
}

var _ streams.Aggregator[data.Number] = (*Accumulator[data.Number])(nil)

type Accumulator[D data.Point[D]] struct {
	dps streams.Map[D]
}

// Aggregate implements delta-to-cumulative aggregation as per spec:
// https://opentelemetry.io/docs/specs/otel/metrics/data-model/#sums-delta-to-cumulative
func (a *Accumulator[D]) Aggregate(ctx context.Context, id streams.Ident, dp D) (D, error) {
	// make the accumulator start with the current sample, discarding any
	// earlier data. return after use
	reset := func() (D, error) {
		clone := dp.Clone()
		a.dps.Store(id, clone)
		return clone, nil
	}

	aggr, ok := a.dps.Load(id)

	// new series: reset
	if !ok {
		return reset()
	}
	// belongs to older series: drop
	if dp.StartTimestamp() < aggr.StartTimestamp() {
		return aggr, ErrOlderStart{Start: aggr.StartTimestamp(), Sample: dp.StartTimestamp()}
	}
	// belongs to later series: reset
	if dp.StartTimestamp() > aggr.StartTimestamp() {
		return reset()
	}
	// out of order: drop
	if dp.Timestamp() <= aggr.Timestamp() {
		return aggr, ErrOutOfOrder{Last: aggr.Timestamp(), Sample: dp.Timestamp()}
	}

	res := aggr.Add(dp)
	a.dps.Store(id, res)
	return res, nil
}

type ErrOlderStart struct {
	Start  pcommon.Timestamp
	Sample pcommon.Timestamp
}

func (e ErrOlderStart) Error() string {
	return fmt.Sprintf("dropped sample with start_time=%s, because series only starts at start_time=%s. consider checking for multiple processes sending the exact same series", e.Sample, e.Start)
}

type ErrOutOfOrder struct {
	Last   pcommon.Timestamp
	Sample pcommon.Timestamp
}

func (e ErrOutOfOrder) Error() string {
	return fmt.Sprintf("out of order: dropped sample from time=%s, because series is already at time=%s", e.Sample, e.Last)
}
