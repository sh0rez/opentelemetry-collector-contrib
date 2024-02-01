// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"errors"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/data"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatocumulativeprocessor/internal/metrics"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Iterator as per https://go.dev/wiki/RangefuncExperiment
type Iter[V any] func(yield func(Ident, V) bool)

func Test[D data.Point[D]](m metrics.Data[D]) Iter[D] {
	mid := m.Ident()

	return func(yield func(Ident, D) bool) {
		for i := 0; i < m.Len(); i++ {
			dp := m.At(i)
			id := Identify(mid, dp.Attributes())
			if !yield(id, dp) {
				break
			}
		}
	}
}

func Samples[D data.Point[D]](m metrics.Data[D], fn func(id Ident, dp D)) {
	mid := m.Ident()

	for i := 0; i < m.Len(); i++ {
		dp := m.At(i)
		id := Identify(mid, dp.Attributes())
		fn(id, dp)
	}
}

func Update[D data.Point[D]](m metrics.Data[D], aggr Aggregator[D]) error {
	var errs error

	Samples(m, func(id Ident, dp D) {
		next, err := aggr.Aggregate(id, dp)
		if err != nil {
			errs = errors.Join(errs, Error(id, err))
			return
		}
		next.CopyTo(dp)
	})

	return errs
}

func Identify(metric metrics.Ident, attrs pcommon.Map) Ident {
	return Ident{metric: metric, attrs: pdatautil.MapHash(attrs)}
}
