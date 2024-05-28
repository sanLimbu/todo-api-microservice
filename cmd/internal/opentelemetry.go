package internal

import (
	"fmt"

	envvar "github.com/sanLimbu/todo-api/internal/envar"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.16.0"
)

// NewOTExporter instantiates the OpenTelemetry exporters using configuration defined in environment variables.
func NewOTExporter(conf *envvar.Configuration) (*prometheus.Exporter, error) {

	//Set up prometheus exporter
	promExporter, err := prometheus.New(prometheus.WithoutUnits())
	if err != nil {
		return nil, fmt.Errorf("prometheus,New: %w", err)
	}

	metricProvider := metric.NewMeterProvider(
		metric.WithReader(promExporter),
	)

	otel.SetMeterProvider(metricProvider)

	//Set up Jaeger exporter
	jaegerEndpoint, _ := conf.Get("JAEGER_ENDPOINT")
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("jaeger.New: %w", err)
	}

	//Set up the trace provider with the Jaeger exporter
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(jaegerExporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(""),
		)),
	)

	otel.SetTracerProvider(traceProvider)

	//Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return promExporter, nil

}
