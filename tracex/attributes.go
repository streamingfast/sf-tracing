package tracex

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Attributes returns a SpanStartOption that sets the attributes for a Span
// from a free-form key-value list. The received inputs are expect to come
// in pairs, where the first element is the key and the second element is the
// value.
//
// So the call:
//
//	tracer.Start("read rows", attribute.Int64("rows", rowCount), attribute.String("table", tableName))
//
// Can be re-written with
//
//	tracer.Start("read rows", tracex.Attributes("rows", rowCount, "table", tableName))
func Attributes(keyedAttributes ...any) trace.SpanStartOption {
	keyedAttributeCount := len(keyedAttributes)
	if keyedAttributeCount <= 0 {
		return nil
	}

	if keyedAttributeCount%2 != 0 {
		panic(fmt.Errorf("keyedAttributes parameters should be a multiple of 2 (keyed_attributes=%v)", keyedAttributes))
	}

	attributes := make([]attribute.KeyValue, keyedAttributeCount/2)
	for i := 0; i < keyedAttributeCount; i += 2 {
		key := toString(keyedAttributes[i])
		rawValue := keyedAttributes[i+1]
		attributeIndex := (i + 1) / 2

		switch v := rawValue.(type) {
		case int, int8, int16, int32, int64, uintptr, uint, uint8, uint16, uint32, uint64:
			attributes[attributeIndex] = attribute.Int64(key, toInt64(v))
		case bool:
			attributes[attributeIndex] = attribute.Bool(key, v)
		case fmt.Stringer:
			attributes[attributeIndex] = attribute.String(key, v.String())
		case string:
			attributes[attributeIndex] = attribute.String(key, v)
		default:
			attributes[attributeIndex] = attribute.String(key, fmt.Sprintf("%v", v))
		}
	}

	return trace.WithAttributes(attributes...)
}

func toString(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%T", input)
	}
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return int64(v)
	case uint:
		return int64(v)
	case int32:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case uintptr:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	default:
		panic("Value should be castable to int64")
	}
}
