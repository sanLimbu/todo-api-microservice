package internal

import (
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"

	"github.com/sanLimbu/todo-api/internal"
	envvar "github.com/sanLimbu/todo-api/internal/envar"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

// NewOTExporter instantiates the OpenTelemetry exporters using configuration defined in environment variables.
func NewOTExporter(conf *envvar.Configuration) (*prometheus.Exporter, error) {

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "runtime.Start")
	}

	config := prometheus.Config{}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
	)

	//Set up prometheus exporter
	promExporter, err := prometheus.New(config, c)
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "prometheus.New")

	}

	global.SetMeterProvider(promExporter.MeterProvider())

	//Set up Jaeger exporter
	jaegerEndpoint, _ := conf.Get("JAEGER_ENDPOINT")
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "jaeger.New")
	}

	//Set up the trace provider with the Jaeger exporter
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(jaegerExporter),
		sdktrace.WithResource(resource.NewSchemaless(attribute.KeyValue{
			Key:   semconv.ServiceNameKey,
			Value: attribute.StringValue("rest-server"),
		})),
	)

	otel.SetTracerProvider(traceProvider)

	//Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return promExporter, nil

}
