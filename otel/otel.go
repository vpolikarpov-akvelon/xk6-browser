// Package otel provides higher level APIs around Open Telemetry instrumentation.
package otel

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName = "k6-browser"
	tracerName  = "browser"
)

// ErrUnsupportedProto indicates that the defined exporter protocol is not supported.
var ErrUnsupportedProto = errors.New("unsupported protocol")

// TraceProvider provides methods for tracers initialization and shutdown of the
// processing pipeline.
type TraceProvider interface {
	Tracer(name string, options ...trace.TracerOption) trace.Tracer
	Shutdown(ctx context.Context) error
	// TODO: Remove support for ForceFlush once we can call Shutdown on 'test end'
	// event after integrating with k6 event system
	ForceFlush(ctx context.Context) error
}

type (
	traceProvShutdownFunc   func(ctx context.Context) error
	traceProvForceFlushFunc func(ctx context.Context) error
)

type traceProvider struct {
	trace.TracerProvider

	noop bool

	shutdown   traceProvShutdownFunc
	forceFlush traceProvForceFlushFunc
}

// NewTraceProvider creates a new trace provider.
func NewTraceProvider(
	ctx context.Context, proto, endpoint string, insecure bool,
) (TraceProvider, error) {
	client, err := newClient(proto, endpoint, insecure)
	if err != nil {
		return nil, fmt.Errorf("creating exporter client: %w", err)
	}

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating exporter: %w", err)
	}

	prov := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)

	otel.SetTracerProvider(prov)

	return &traceProvider{
		TracerProvider: prov,
		shutdown:       prov.Shutdown,
		forceFlush:     prov.ForceFlush,
	}, nil
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)
}

func newClient(proto, endpoint string, insecure bool) (otlptrace.Client, error) {
	// TODO: Support gRPC
	switch strings.ToLower(proto) {
	case "http":
		return newHTTPClient(endpoint, insecure), nil
	default:
		return nil, ErrUnsupportedProto
	}
}

func newHTTPClient(endpoint string, insecure bool) otlptrace.Client {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
	}
	if insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptracehttp.NewClient(opts...)
}

// NewNoopTraceProvider creates a new noop trace provider.
func NewNoopTraceProvider() TraceProvider {
	prov := trace.NewNoopTracerProvider()

	otel.SetTracerProvider(prov)

	return &traceProvider{
		TracerProvider: prov,
		noop:           true,
	}
}

// Shutdown shuts down TracerProvider releasing any held computational resources.
// After Shutdown is called, all methods are no-ops.
func (tp *traceProvider) Shutdown(ctx context.Context) error {
	if tp.noop {
		return nil
	}

	return tp.shutdown(ctx)
}

// TODO: Remove once we can call Shutdown on 'test end' event after integrating with
// k6 event system.
func (tp *traceProvider) ForceFlush(ctx context.Context) error {
	if tp.noop {
		return nil
	}

	return tp.forceFlush(ctx)
}

// Trace generates a trace span and a context containing the generated span.
// If the input context already contains a span, the generated spain will be a child of that span
// otherwise it will be a root span. This behavior can be overridden by providing `WithNewRoot()`
// as a SpanOption, causing the newly-created Span to be a root span even if `ctx` contains a Span.
// When creating a Span it is recommended to provide all known span attributes using the `WithAttributes()`
// SpanOption as samplers will only have access to the attributes provided when a Span is created.
// Any Span that is created MUST also be ended. This is the responsibility of the user. Implementations of
// this API may leak memory or other resources if Spans are not ended.
func Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, spanName, opts...)
}
