package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTraceIDContext(t *testing.T) {
	ctx := context.Background()
	traceID := NewRandomTraceID()
	require.True(t, traceID.IsValid())

	ctx = WithTraceID(ctx, traceID)

	result := GetTraceID(ctx)
	require.Equal(t, traceID.String(), result.String())
}
