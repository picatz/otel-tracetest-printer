package printer_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/shoenig/test/must"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	printer "github.com/picatz/otel-tracetest-printer"
)

func TestPrintSpanTree(t *testing.T) {
	rootSpan := tracetest.SpanStub{
		Name: "root-span",
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:     [8]byte{10, 11, 12, 13, 14, 15, 16, 17},
			TraceFlags: trace.FlagsSampled,
		}),
		StartTime: time.Now().Add(-2 * time.Second),
		EndTime:   time.Now().Add(-1 * time.Second),
		Attributes: []attribute.KeyValue{
			{Key: "component", Value: attribute.StringValue("root")},
		},
	}

	childSpan1 := tracetest.SpanStub{
		Name: "child-span-1",
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    rootSpan.SpanContext.TraceID(),
			SpanID:     [8]byte{20, 21, 22, 23, 24, 25, 26, 27},
			TraceFlags: trace.FlagsSampled,
		}),
		Parent:    rootSpan.SpanContext,
		StartTime: time.Now().Add(-1500 * time.Millisecond),
		EndTime:   time.Now().Add(-1 * time.Second),
		Attributes: []attribute.KeyValue{
			{Key: "component", Value: attribute.StringValue("child-1")},
		},
	}

	childSpan2 := tracetest.SpanStub{
		Name: "child-span-2",
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    rootSpan.SpanContext.TraceID(),
			SpanID:     [8]byte{30, 31, 32, 33, 34, 35, 36, 37},
			TraceFlags: trace.FlagsSampled,
		}),
		Parent:    rootSpan.SpanContext,
		StartTime: time.Now().Add(-1400 * time.Millisecond),
		EndTime:   time.Now().Add(-1 * time.Second),
		Attributes: []attribute.KeyValue{
			{Key: "component", Value: attribute.StringValue("child-2")},
			{Key: "error_code", Value: attribute.StringValue("something_wrong")},
		},
	}

	childSpan3 := tracetest.SpanStub{
		Name: "child-span-3",
		SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    rootSpan.SpanContext.TraceID(),
			SpanID:     [8]byte{40, 41, 42, 43, 44, 45, 46, 47},
			TraceFlags: trace.FlagsSampled,
		}),
		Parent:    childSpan2.SpanContext,
		StartTime: time.Now().Add(-1300 * time.Millisecond),
		EndTime:   time.Now().Add(-1 * time.Second),
		Attributes: []attribute.KeyValue{
			{Key: "component", Value: attribute.StringValue("child-3")},
		},
	}

	spans := []tracetest.SpanStub{rootSpan, childSpan1, childSpan2, childSpan3}

	var buf bytes.Buffer
	printer.PrintSpanTree(&buf, spans)
	output := buf.String()

	must.StrContains(t, output, "root-span", must.Sprint("Expected 'root-span' in output"))
	must.StrContains(t, output, "child-span-1", must.Sprint("Expected 'child-span-1' in output"))
	must.StrContains(t, output, "child-span-2", must.Sprint("Expected 'child-span-2' in output"))
	must.StrContains(t, output, "child-span-3", must.Sprint("Expected 'child-span-3' in output"))

	// Optionally check for the presence of a known attribute or error highlight.
	must.StrContains(t, output, "error_code", must.Sprint("Should see error-related attribute name in the output"))

	t.Logf("\n%s\n", output)
}
