# sf-tracing

## Setup

```bash
export SF_TRACING=<Collector-URL>
```

### Collector-URL
- stdout://
- cloudtrace://[host:port]?project_id=<project_id>&ratio=<0.25>
- jaeger://[host:port]?scheme=<http|https>
- zipkin://[host:port]?scheme=<http|https>
- otelcol://[host:port]

```go
package main

import (
	"context"
	tracing "github.com/streamingfast/sf-tracing"
	"go.opentelemetry.io/otel"
)

func main() {
	ctx := context.Background()
	
	err := tracing.SetupOpenTelemetry("my-service-name")
	if err != nil {
        panic(err)
    }

	myTracer := otel.Tracer("pipeline")

	ctx, span := myTracer.Start(ctx, "something_start")
	defer span.End()

	span.SetAttributes(attribute.Int64("block_num", 1))
	span.AddEvent("something_append")
	
	span.SetStatus(otelcode.Ok, "")
}
```