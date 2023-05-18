package tracing

import (
	"context"
	"encoding/hex"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	ttrace "go.opentelemetry.io/otel/trace"
	"runtime/debug"
)

// Returns a tracer
func GetTracer() ttrace.Tracer {
	opts := []ttrace.TracerOption{
		ttrace.WithInstrumentationAttributes(
			attribute.String("foo", "bar"),
		),
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				opts = append(opts, ttrace.WithInstrumentationVersion(setting.Value))
			}
		}
	}

	return otel.GetTracerProvider().Tracer(
		"github.com/streamingfast/sf-tracing",
		ttrace.WithInstrumentationVersion(""),
	)
}

// GetTraceID gets the TraceID from the context, you should check if it IsValid()
func GetTraceID(ctx context.Context) ttrace.TraceID {
	if span := ttrace.SpanFromContext(ctx); span != nil {
		return span.SpanContext().TraceID()
	}
	return ttrace.TraceID{} // invalid TraceID
}

// WithTraceID adds an otel span with the given traceID
func WithTraceID(ctx context.Context, traceID ttrace.TraceID) context.Context {
	return ttrace.ContextWithSpanContext(ctx, ttrace.NewSpanContext(ttrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  NewRandomSpanID(),
	}))
}

// NewRandomTraceID returns a random trace ID using OpenCensus default config IDGenerator.
func NewRandomTraceID() ttrace.TraceID {
	return config.Load().(*defaultIDGenerator).NewTraceID()
}

// NewRandomSpanID returns a random span ID using OpenCensus default config IDGenerator.
func NewRandomSpanID() ttrace.SpanID {
	return config.Load().(*defaultIDGenerator).NewSpanID()
}

// NewZeroedTraceID returns a mocked, fixed trace ID containing only 0s.
func NewZeroedTraceID() ttrace.TraceID {
	return NewFixedTraceID("00000000000000000000000000000000")
}

// NewFixedTraceID returns a mocked, fixed trace ID from an hexadecimal string.
// The string in question must be a valid hexadecimal string containing exactly
// 32 characters (16 bytes). Any invalid input results in a panic.
func NewFixedTraceID(hexTraceID string) (out ttrace.TraceID) {
	if len(hexTraceID) != 32 {
		panic(fmt.Errorf("trace id hexadecimal value should have 32 characters, received %d for %q", len(hexTraceID), hexTraceID))
	}

	bytes, err := hex.DecodeString(hexTraceID)
	if err != nil {
		panic(fmt.Errorf("unable to decode hex trace id %q: %s", hexTraceID, err))
	}

	for i := 0; i < 16; i++ {
		out[i] = bytes[i]
	}

	return
}

// NewZeroedTraceIDInContext is similar to NewZeroedTraceID but will actually
// insert the span straight into a context that can later be used
// to ensure the trace id is controlled.
//
// This should be use only in testing to provide a fixed trace ID
// instead of generating a new one each time.
func NewZeroedTraceIDInContext(ctx context.Context) context.Context {
	ctx = ttrace.ContextWithRemoteSpanContext(ctx, ttrace.NewSpanContext(ttrace.SpanContextConfig{
		TraceID: NewZeroedTraceID(),
		SpanID:  config.Load().(*defaultIDGenerator).NewSpanID(),
	}))

	return ctx
}

// NewFixedTraceIDInContext is similar to NewFixedTraceID but will actually
// insert the span straight into a context that can later be used
// to ensure the trace id is controlled.
//
// This should be use only in testing to provide a fixed trace ID
// instead of generating a new one each time.
func NewFixedTraceIDInContext(ctx context.Context, hexTraceID string) context.Context {
	ctx = ttrace.ContextWithRemoteSpanContext(ctx, ttrace.NewSpanContext(ttrace.SpanContextConfig{
		TraceID: NewFixedTraceID(hexTraceID),
		SpanID:  config.Load().(*defaultIDGenerator).NewSpanID(),
	}))

	return ctx
}
