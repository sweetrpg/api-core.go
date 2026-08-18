package tracing

import (
	"context"
	"fmt"
	"os"

	"github.com/sweetrpg/api-core.go/constants"
	"github.com/sweetrpg/common.go/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer

var tp *sdktrace.TracerProvider
var ctx context.Context

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// Your preferred exporter: console, jaeger, zipkin, OTLP, etc.
	exporter, err := otlptracehttp.New(ctx)
	// url := os.Getenv(constants.ZIPKIN_ENDPOINT)
	// exporter, err := zipkin.New(url)
	if err != nil {
		logging.Logger.Error("Error while setting up Zipkin exporter", "error", err.Error())
		return nil, err
	}

	return exporter, nil
}

func newTraceProvider(exp sdktrace.SpanExporter, serviceName string) *sdktrace.TracerProvider {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)

	if err != nil {
		panic(err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}
func SetupTracing(serviceName string) {
	ctx = context.Background()

	exp, err := newExporter(ctx)
	if err != nil {
		logging.Logger.Error(fmt.Sprintf("Failed to initialize exporter: %v", err))
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp = newTraceProvider(exp, serviceName)

	otel.SetTracerProvider(tp)

	// Without a global propagator, otel.GetTextMapPropagator() defaults to a no-op - both
	// otelgin.Middleware's inbound header extraction AND otelhttp's outbound header injection
	// read from this, so without it neither incoming nor outgoing trace context ever actually
	// propagates, despite the middleware/transport being wired up correctly everywhere else.
	// W3C TraceContext matches the Swift/Rust frontends' own propagators.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Finally, set the tracer that can be used for this package.
	name := os.Getenv(constants.TRACING_NAME)
	Tracer = tp.Tracer(name)
}

func TeardownTracing() {
	_ = tp.Shutdown(ctx)
}
