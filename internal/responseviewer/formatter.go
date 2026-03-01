package responseviewer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	grpcpkg "github.com/futuramacoder/protopilot/internal/grpc"
	"github.com/futuramacoder/protopilot/internal/ui"
)

// JSON syntax highlighting colors.
var (
	jsonKeyStyle    = lipgloss.NewStyle().Foreground(ui.ColorSecondary)
	jsonStringStyle = lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	jsonNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))
	jsonBoolStyle   = lipgloss.NewStyle().Foreground(ui.ColorWarning)
	jsonNullStyle   = lipgloss.NewStyle().Foreground(ui.ColorDimmed)
	jsonBracketStyle = lipgloss.NewStyle().Foreground(ui.ColorText)
)

// FormatJSON pretty-prints JSON with syntax highlighting.
func FormatJSON(data []byte) string {
	if len(data) == 0 {
		return ui.DimmedStyle.Render("(empty response)")
	}

	// Pretty-print first.
	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err != nil {
		// Not valid JSON, return as-is.
		return string(data)
	}
	indented, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		return string(data)
	}

	return highlightJSON(string(indented))
}

// highlightJSON applies syntax colors to a pretty-printed JSON string.
func highlightJSON(s string) string {
	var result strings.Builder
	lines := strings.Split(s, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(highlightJSONLine(line))
	}

	return result.String()
}

// highlightJSONLine colors a single line of JSON.
func highlightJSONLine(line string) string {
	trimmed := strings.TrimSpace(line)
	indent := line[:len(line)-len(trimmed)]

	if trimmed == "" {
		return line
	}

	// Bracket-only lines.
	switch trimmed {
	case "{", "}", "{}", "[", "]", "[]", "},", "],":
		return indent + jsonBracketStyle.Render(trimmed)
	}

	// Key-value line: "key": value
	if strings.HasPrefix(trimmed, `"`) {
		colonIdx := strings.Index(trimmed, `": `)
		if colonIdx > 0 {
			key := trimmed[:colonIdx+1]
			rest := trimmed[colonIdx+2:] // ": " then value

			coloredKey := jsonKeyStyle.Render(key)
			coloredVal := colorizeJSONValue(rest)
			return indent + coloredKey + ": " + coloredVal
		}
		// Standalone string value (e.g., in array).
		return indent + colorizeJSONValue(trimmed)
	}

	// Standalone value (number, bool, null in array).
	return indent + colorizeJSONValue(trimmed)
}

// colorizeJSONValue colors a JSON value token.
func colorizeJSONValue(s string) string {
	// Strip trailing comma for styling, then re-add.
	trailing := ""
	if strings.HasSuffix(s, ",") {
		trailing = ","
		s = s[:len(s)-1]
	}

	switch {
	case s == "null":
		return jsonNullStyle.Render(s) + trailing
	case s == "true" || s == "false":
		return jsonBoolStyle.Render(s) + trailing
	case strings.HasPrefix(s, `"`):
		return jsonStringStyle.Render(s) + trailing
	case strings.HasPrefix(s, "{") || strings.HasPrefix(s, "["):
		return jsonBracketStyle.Render(s) + trailing
	default:
		// Assume number.
		return jsonNumberStyle.Render(s) + trailing
	}
}

// FormatMetadata formats gRPC metadata (headers or trailers) as "key: value" lines.
func FormatMetadata(label string, md metadata.MD) string {
	if len(md) == 0 {
		return ""
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorSecondary)
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
	valStyle := ui.DimmedStyle

	var lines []string
	lines = append(lines, headerStyle.Render(label))
	for k, vals := range md {
		for _, v := range vals {
			lines = append(lines, fmt.Sprintf("  %s: %s",
				keyStyle.Render(k),
				valStyle.Render(v)))
		}
	}

	return strings.Join(lines, "\n")
}

// FormatStatus formats the gRPC status code with color coding.
func FormatStatus(st *status.Status) string {
	if st == nil {
		return ""
	}

	code := st.Code()
	codeName := code.String()
	color := ui.StatusColor(code)
	codeStyle := lipgloss.NewStyle().Bold(true).Foreground(color)

	result := codeStyle.Render(fmt.Sprintf("Status: %s (%d)", codeName, code))

	if st.Message() != "" {
		msgStyle := lipgloss.NewStyle().Foreground(ui.ColorText)
		result += "  " + msgStyle.Render(st.Message())
	}

	return result
}

// FormatLatency formats the duration as a human-readable string.
func FormatLatency(d time.Duration) string {
	var text string
	switch {
	case d < time.Millisecond:
		text = fmt.Sprintf("%d\u00B5s", d.Microseconds())
	case d < time.Second:
		text = fmt.Sprintf("%dms", d.Milliseconds())
	default:
		text = fmt.Sprintf("%.1fs", d.Seconds())
	}

	return ui.DimmedStyle.Render(fmt.Sprintf("Latency: %s", text))
}

// FormatErrorDetails formats decoded error details for display.
func FormatErrorDetails(details []grpcpkg.ErrorDetail) string {
	if len(details) == 0 {
		return ""
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorError)
	typeStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWarning)
	contentStyle := lipgloss.NewStyle().Foreground(ui.ColorText)

	var lines []string
	lines = append(lines, headerStyle.Render("Error Details"))
	for _, d := range details {
		lines = append(lines, "  "+typeStyle.Render(d.Type))
		for _, cl := range strings.Split(d.Content, "\n") {
			lines = append(lines, "    "+contentStyle.Render(cl))
		}
	}

	return strings.Join(lines, "\n")
}
