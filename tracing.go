// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
}

// SetupOpenTelemetry sets up tracers based on the `DTRACING` environment variable.
//
// Options are:
//   - stdout://
//   - cloudtrace://[host:port]?project_id=<project_id>&ratio=<0.25>
//   - jaeger://[host:port]?scheme=<http|https>
//   - zipkin://[host:port]?scheme=<http|https>
//   - otelcol://[host:port]
func SetupOpenTelemetry(ctx context.Context, serviceName string) error {
	conf := os.Getenv("SF_TRACING")
	if conf == "" {
		return nil
	}
	u, err := url.Parse(conf)
	if err != nil {
		return fmt.Errorf("parsing env var DTRACING with value %q: %w", conf, err)
	}

	switch u.Scheme {
	case "stdout":
		return registerStdout(ctx, serviceName, u)
	case "cloudtrace":
		return registerCloudTrace(ctx, serviceName, u)
	case "otelcol":
		return registerOtelcol(ctx, serviceName, u)
	case "zipkin":
		return registerZipkin(ctx, serviceName, u)
	case "jaeger":
		return registerJaeger(ctx, serviceName, u)
	default:
		return fmt.Errorf("unsupported tracing scheme %q", u.Scheme)
	}
}

func registerStdout(ctx context.Context, serviceName string, u *url.URL) error {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stderr),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)

	if err != nil {
		return fmt.Errorf("creating stdout exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", os.Getenv("NAMESPACE") /* that won't work, whatever */),
		),
	)

	if err != nil {
		return fmt.Errorf("creating stdout resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return nil
}

func registerCloudTrace(ctx context.Context, serviceName string, u *url.URL) error {
	projectID := u.Query().Get("project_id")
	exp, err := texporter.New(texporter.WithProjectID(projectID))
	if err != nil {
		return fmt.Errorf("creating cloudtrace exporter: %w", err)
	}

	// Identify your application using resource detection
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		// Keep the default detectors
		resource.WithTelemetrySDK(),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	if err != nil {
		return fmt.Errorf("creating resource: %w", err)
	}

	ratio := 0.25
	if u.Query().Get("ratio") != "" {
		ratio, err = strconv.ParseFloat(u.Query().Get("ratio"), 64)
		if err != nil {
			return fmt.Errorf("parsing ratio: %w", err)
		}
	}

	sampler := trace.TraceIDRatioBased(ratio)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)
	otel.SetTracerProvider(tp)

	return nil
}

func registerOtelcol(ctx context.Context, serviceName string, u *url.URL) error {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, u.Host, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return nil
}

func registerZipkin(ctx context.Context, serviceName string, u *url.URL) error {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// Set up a trace exporter
	traceExporter, err := zipkin.New(
		fmt.Sprintf("%s://%s/api/v2/spans", u.Query().Get("scheme"), u.Host),
	)

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return nil
}

func registerJaeger(ctx context.Context, serviceName string, u *url.URL) error {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	traceExporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(fmt.Sprintf("%s://%s/api/traces", u.Query().Get("scheme"), u.Host)),
		),
	)
	if err != nil {
		return err
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return nil
}
