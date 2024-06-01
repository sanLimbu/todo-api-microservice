package main

import (
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {

	//Initialize the tracer
	// initTracer()

	// //Create an HTTP client with OpenTelemetry instrumentation
	// client0A3 := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	// //Intitialize the OpenAPI3 client
	// client, err := openapi3.NewCli
}

//initTracer initializes OpenTelemetry tracing with Jaeger and stdout exporters
func initTracer() {

	jaegerEndpoint := "http;//localhost:14268/api/traces"

	//Create a Jaeger exporter
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		log.Fatalf("Couldn't initialize stdout exporter: ", err)
	}

	//Create a stdout exporter to print traces to the console
	stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalln("couldn't initiate stdout exporter: ", err)
	}

	//Create a trace provider with the exporters
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(stdoutExporter),
		sdktrace.WithBatcher(jaegerExporter),
	)

	//Set the global trace provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

}
