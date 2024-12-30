package printer

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// timeFormat is how we display span start/end times.
// Adjust as needed (e.g., omit date, show only HH:MM:SS).
const timeFormat = "2006-01-02 15:04:05.000 MST"

var (
	// boxStyle encloses each span (and its children) within a bordered box,
	// with minimal margin to avoid extra whitespace.
	boxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Margin(0).
			PaddingLeft(1).
			PaddingRight(1)

	// labelStyle is used for the label text (e.g., "Span Name:", "TraceID:", etc.).
	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	// valueStyle is used for standard span attribute values.
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	// errorHighlightStyle is used for attribute values that often indicate errors or warnings.
	errorHighlightStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	// childIndent is the indentation string for nested child boxes.
	childIndent = "  "
)

// PrintSpanTree organizes spans into a hierarchical tree of parent → children
// and writes them to w. Each parent’s box encloses its children’s boxes.
func PrintSpanTree(w io.Writer, spans []tracetest.SpanStub) {
	if len(spans) == 0 {
		return
	}

	// Build a map of SpanID → SpanStub for quick lookups
	spanByID := make(map[string]tracetest.SpanStub, len(spans))
	for _, s := range spans {
		spanByID[s.SpanContext.SpanID().String()] = s
	}

	// Build a parent → slice of children map
	childrenMap := make(map[string][]tracetest.SpanStub)
	for _, s := range spans {
		if parentID := s.Parent.SpanID().String(); s.Parent.SpanID().IsValid() {
			childrenMap[parentID] = append(childrenMap[parentID], s)
		}
	}

	// Sort children by start time for stable ordering
	for pid := range childrenMap {
		sort.Slice(childrenMap[pid], func(i, j int) bool {
			return childrenMap[pid][i].StartTime.Before(childrenMap[pid][j].StartTime)
		})
	}

	// Identify the root spans (i.e., those with no valid parent).
	var roots []tracetest.SpanStub
	for _, s := range spans {
		if !s.Parent.SpanID().IsValid() {
			roots = append(roots, s)
		}
	}

	// Sort roots by start time for stable ordering
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].StartTime.Before(roots[j].StartTime)
	})

	// Recursively build + print each root
	for _, root := range roots {
		treeStr := buildSpanBox(root, childrenMap)
		fmt.Fprintln(w, treeStr)
	}
}

// buildSpanBox returns a single Lip Gloss-rendered string containing:
//   - The current span’s details
//   - All of its children’s boxes (recursively)
func buildSpanBox(span tracetest.SpanStub, childrenMap map[string][]tracetest.SpanStub) string {
	// 1) Build lines for this span
	var lines []string

	lines = append(lines, joinLabelValue("Span Name:", span.Name))
	lines = append(lines, joinLabelValue("TraceID:", span.SpanContext.TraceID().String()))
	lines = append(lines, joinLabelValue("SpanID:", span.SpanContext.SpanID().String()))

	// Include parent ID if valid
	if span.Parent.SpanID().IsValid() {
		lines = append(lines, joinLabelValue("ParentSpan:", span.Parent.SpanID().String()))
	}

	// Format times to avoid the verbose 'm=+...'
	lines = append(lines, joinLabelValue("Start Time:", formatTime(span.StartTime)))
	lines = append(lines, joinLabelValue("End Time:", formatTime(span.EndTime)))

	duration := span.EndTime.Sub(span.StartTime)
	lines = append(lines, joinLabelValue("Duration:", duration))

	// 2) Attributes
	lines = append(lines, labelStyle.Render("Attributes:"))
	for _, attr := range span.Attributes {
		val := attr.Value.AsInterface()

		// If this attribute is an error-related key, highlight it
		attrStyle := valueStyle
		if isErrorAttribute(string(attr.Key), val) {
			attrStyle = errorHighlightStyle
		}

		bullet := fmt.Sprintf("• %s = %v", attr.Key, val)
		lines = append(lines, childIndent+attrStyle.Render(bullet))
	}

	// 3) Recursively build child boxes
	for _, child := range childrenMap[span.SpanContext.SpanID().String()] {
		childBox := buildSpanBox(child, childrenMap)
		// Indent child content so it appears nested
		childBoxIndented := indentAllLines(childBox, childIndent)
		lines = append(lines, childBoxIndented)
	}

	// 4) Combine all lines vertically
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// 5) Wrap in a single box
	return boxStyle.Render(content)
}

// joinLabelValue is a helper that renders "Label: Value" with distinct
// styling for each portion.
func joinLabelValue(label string, val interface{}) string {
	return labelStyle.Render(label) + "  " + valueStyle.Render(fmt.Sprintf("%v", val))
}

// formatTime returns a more concise string for the given time.
func formatTime(t time.Time) string {
	return t.Format(timeFormat)
}

// indentAllLines applies an indent prefix to each line in a multi-line string.
func indentAllLines(s, indent string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, indent+line)
	}
	return strings.Join(out, "\n")
}

// isErrorAttribute is a simple helper to check if an attribute might be error-related.
// Customize this logic to suit your system’s notion of “error” or “warning” attributes.
func isErrorAttribute(key string, val interface{}) bool {
	switch key {
	case "error", "error_code", "rpc.connect_rpc.error_code":
		return true
	}

	// If it's a string that looks like an error sentinel, highlight it:
	if strVal, ok := val.(string); ok && (strVal == "error" || strVal == "not_found") {
		return true
	}

	return false
}
