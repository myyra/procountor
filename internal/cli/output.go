package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type outputMode string

const (
	outputModeHuman  outputMode = "human"
	outputModeJSON   outputMode = "json"
	outputModePlain  outputMode = "plain"
	outputModeBinary            = "binary"
)

func (r *runner) mode() outputMode {
	return r.output
}

func (r *runner) writeJSONValue(v any) error {
	enc := json.NewEncoder(r.out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return wrapErr("write JSON output", err)
	}
	return nil
}

func (r *runner) writeOutput(v any) error {
	switch r.mode() {
	case outputModeJSON:
		return r.writeJSONValue(v)
	case outputModeHuman:
		return writeTextOutput(r.out, v)
	case outputModePlain:
		return writePlainOutput(r.out, v)
	default:
		return invalidUsage("invalid --output %q", r.output)
	}
}

func (r *runner) writeReaderOutput(value io.Reader, outPath string) error {
	if strings.TrimSpace(outPath) == "" {
		_, err := io.Copy(r.out, value)
		return wrapErr("write binary output", err)
	}

	file, err := os.Create(strings.TrimSpace(outPath))
	if err != nil {
		return fmt.Errorf("create --out file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := io.Copy(file, value); err != nil {
		return wrapErr("write output file", err)
	}
	return r.writeOutput(map[string]any{"ok": true, "path": outPath})
}

func writeHumanError(w io.Writer, msg string) {
	if strings.TrimSpace(msg) == "" {
		return
	}
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	_, _ = io.WriteString(w, msg)
}

func formatAPIError(statusCode int, body any) string {
	summary := fmt.Sprintf("api error %d", statusCode)
	message := extractErrorSummary(body)
	if message != "" {
		summary += ": " + message
	}

	details := prettyJSON(body)
	if details == "" || details == "{}" || details == scalarInputNullLiteral {
		return summary
	}
	return summary + "\n" + details
}

func extractErrorSummary(body any) string {
	m, ok := toJSONMap(body)
	if !ok {
		return ""
	}
	for _, key := range []string{"message", "error", "description"} {
		if value, ok := m[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func prettyJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

func toJSONMap(v any) (map[string]any, bool) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}
	return m, true
}

func writePlainOutput(w io.Writer, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}

	var sb strings.Builder
	renderPlainValue(&sb, normalized, "")
	if sb.Len() == 0 {
		sb.WriteString("null\n")
	}
	_, err = io.WriteString(w, sb.String())
	return wrapErr("write plain output", err)
}

func renderPlainValue(sb *strings.Builder, v any, prefix string) {
	switch value := v.(type) {
	case map[string]any:
		renderPlainObject(sb, value, prefix)
	case []any:
		renderPlainArray(sb, value, prefix)
	default:
		if prefix == "" {
			sb.WriteString(displayScalar(value))
		} else {
			sb.WriteString(prefix)
			sb.WriteByte('\t')
			sb.WriteString(displayScalar(value))
		}
		sb.WriteByte('\n')
	}
}

func renderPlainObject(sb *strings.Builder, value map[string]any, prefix string) {
	if len(value) == 0 {
		if prefix != "" {
			sb.WriteString(prefix)
			sb.WriteString("\t{}\n")
		}
		return
	}
	keys := orderedKeys(value)
	for _, key := range keys {
		nextPrefix := key
		if prefix != "" {
			nextPrefix = prefix + "." + key
		}
		renderPlainValue(sb, value[key], nextPrefix)
	}
}

func renderPlainArray(sb *strings.Builder, value []any, prefix string) {
	if len(value) == 0 {
		if prefix != "" {
			sb.WriteString(prefix)
			sb.WriteString("\t[]\n")
		} else {
			sb.WriteString("[]\n")
		}
		return
	}
	for idx, item := range value {
		nextPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
		if prefix == "" {
			nextPrefix = fmt.Sprintf("[%d]", idx)
		}
		renderPlainValue(sb, item, nextPrefix)
	}
}
