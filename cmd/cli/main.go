package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sanLimbu/todo-api/pkg/openapi3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

func main() {

	//Initialize the tracer
	initTracer()

	//Create an HTTP client with OpenTelemetry instrumentation
	client0A3 := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	//Intitialize the OpenAPI3 client
	client, err := openapi3.NewClientWithResponses("http://0.0.0.0:9234", openapi3.WithHTTPClient(&client0A3))
	if err != nil {
		log.Fatalf("Couldn't instantiate client: %s", err)
	}

	newPtrStr := func(s string) *string {
		return &s
	}

	newPtrTime := func(t time.Time) *time.Time {
		return &t
	}

	count := 1
	for count < 101 {

		//cREATE
		priority := openapi3.Low

		_, err := client.CreateTaskWithResponse(context.Background(),
			openapi3.CreateTaskJSONRequestBody{
				Dates: &openapi3.Dates{
					Start: newPtrTime(time.Now()),
					Due:   newPtrTime(time.Now().Add(time.Hour * 24)),
				},
				Description: newPtrStr(fmt.Sprintf("Searchable Task %d", count)),
				Priority:    &priority,
			})

		if err != nil {
			log.Fatalf("Couldn't create task %s", err)
		}
		count++

	}

}

//initTracer initializes OpenTelemetry tracing with Jaeger and stdout exporters
func initTracer() {

	jaegerEndpoint := "http://localhost:14268/api/traces"

	//Create a Jaeger exporter
	jaegerExporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerEndpoint)))
	if err != nil {
		log.Fatalf("Couldn't initialize stdout exporter: ", err)
	}

	//Create a stdout exporter to print traces to the console
	_, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalln("couldn't initiate stdout exporter: ", err)
	}

	//Create a trace provider with the exporters
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(jaegerExporter),
		trace.WithResource(resource.NewSchemaless(attribute.KeyValue{
			Key:   semconv.ServiceNameKey,
			Value: attribute.StringValue("rest-server"),
		})),
	)

	//Set the global trace provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

}
