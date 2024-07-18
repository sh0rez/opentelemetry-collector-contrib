// Code generated by mdatagen. DO NOT EDIT.

package metadata

import (
	"errors"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
)

func Meter(settings component.TelemetrySettings) metric.Meter {
	return settings.MeterProvider.Meter("otelcol/deltatocumulative")
}

func Tracer(settings component.TelemetrySettings) trace.Tracer {
	return settings.TracerProvider.Tracer("otelcol/deltatocumulative")
}

// TelemetryBuilder provides an interface for components to report telemetry
// as defined in metadata and user config.
type TelemetryBuilder struct {
	meter                                metric.Meter
	DeltatocumulativeDatapointsDropped   metric.Int64Counter
	DeltatocumulativeDatapointsProcessed metric.Int64Counter
	DeltatocumulativeGapsLength          metric.Int64Counter
	DeltatocumulativeStreamsEvicted      metric.Int64Counter
	DeltatocumulativeStreamsLimit        metric.Int64Gauge
	DeltatocumulativeStreamsMaxStale     metric.Int64Gauge
	DeltatocumulativeStreamsTracked      metric.Int64UpDownCounter
	level                                configtelemetry.Level
}

// telemetryBuilderOption applies changes to default builder.
type telemetryBuilderOption func(*TelemetryBuilder)

// WithLevel sets the current telemetry level for the component.
func WithLevel(lvl configtelemetry.Level) telemetryBuilderOption {
	return func(builder *TelemetryBuilder) {
		builder.level = lvl
	}
}

// NewTelemetryBuilder provides a struct with methods to update all internal telemetry
// for a component
func NewTelemetryBuilder(settings component.TelemetrySettings, options ...telemetryBuilderOption) (*TelemetryBuilder, error) {
	builder := TelemetryBuilder{level: configtelemetry.LevelBasic}
	for _, op := range options {
		op(&builder)
	}
	var err, errs error
	if builder.level >= configtelemetry.LevelBasic {
		builder.meter = Meter(settings)
	} else {
		builder.meter = noop.Meter{}
	}
	builder.DeltatocumulativeDatapointsDropped, err = builder.meter.Int64Counter(
		"otelcol_deltatocumulative.datapoints.dropped",
		metric.WithDescription("number of datapoints dropped due to given 'reason'"),
		metric.WithUnit("{datapoint}"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeDatapointsProcessed, err = builder.meter.Int64Counter(
		"otelcol_deltatocumulative.datapoints.processed",
		metric.WithDescription("number of datapoints processed"),
		metric.WithUnit("{datapoint}"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeGapsLength, err = builder.meter.Int64Counter(
		"otelcol_deltatocumulative.gaps.length",
		metric.WithDescription("total duration where data was expected but not received"),
		metric.WithUnit("s"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeStreamsEvicted, err = builder.meter.Int64Counter(
		"otelcol_deltatocumulative.streams.evicted",
		metric.WithDescription("number of streams evicted"),
		metric.WithUnit("{stream}"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeStreamsLimit, err = builder.meter.Int64Gauge(
		"otelcol_deltatocumulative.streams.limit",
		metric.WithDescription("upper limit of tracked streams"),
		metric.WithUnit("{stream}"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeStreamsMaxStale, err = builder.meter.Int64Gauge(
		"otelcol_deltatocumulative.streams.max_stale",
		metric.WithDescription("duration after which streams inactive streams are dropped"),
		metric.WithUnit("s"),
	)
	errs = errors.Join(errs, err)
	builder.DeltatocumulativeStreamsTracked, err = builder.meter.Int64UpDownCounter(
		"otelcol_deltatocumulative.streams.tracked",
		metric.WithDescription("number of streams tracked"),
		metric.WithUnit("{dps}"),
	)
	errs = errors.Join(errs, err)
	return &builder, errs
}
