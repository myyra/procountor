package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/jx"
)

type scalarInputKind string

var nullScalarInput = &struct{}{}

const (
	scalarInputNullLiteral                 = "null"
	scalarInputString      scalarInputKind = "string"
	scalarInputEnum        scalarInputKind = "enum"
	scalarInputInt         scalarInputKind = "int"
	scalarInputFloat       scalarInputKind = "float"
	scalarInputBool        scalarInputKind = "bool"
	scalarInputDate        scalarInputKind = "date"
	scalarInputDateTime    scalarInputKind = "datetime"
)

const bodyDateTimeLayout = "2006-01-02T15:04:05"

type requestInputReader struct {
	runner    *runner
	stdinUsed bool
}

func newRequestInputReader(r *runner) *requestInputReader {
	return &requestInputReader{runner: r}
}

func formatJSONInputHelp(help string) string {
	trimmed := strings.TrimSpace(help)
	if trimmed == "" {
		return "Accepts inline JSON, @file, or - for stdin."
	}
	switch trimmed[len(trimmed)-1] {
	case '.', '!', '?':
		return trimmed + " Accepts inline JSON, @file, or - for stdin."
	default:
		return trimmed + ". Accepts inline JSON, @file, or - for stdin."
	}
}

type requestBodyBuilder struct {
	input         *requestInputReader
	body          map[string]any
	explicitFlags []string
}

func newRequestBodyBuilder(input *requestInputReader) *requestBodyBuilder {
	return &requestBodyBuilder{
		input: input,
		body:  make(map[string]any),
	}
}

func (b *requestBodyBuilder) SetScalar(fieldName, flagName string, raw *string, kind scalarInputKind, nullable bool) error {
	value, set, err := readScalarInput(raw, kind, nullable)
	if err != nil {
		return invalidUsage("invalid --%s: %v", flagName, err)
	}
	if !set {
		return nil
	}
	if value == nullScalarInput {
		b.body[fieldName] = nil
	} else {
		b.body[fieldName] = value
	}
	b.addExplicitFlag(flagName)
	return nil
}

func (b *requestBodyBuilder) SetScalarArray(fieldName, flagName string, values []string, kind scalarInputKind) error {
	parsed, set, err := readScalarInputArray(values, kind)
	if err != nil {
		return invalidUsage("invalid --%s: %v", flagName, err)
	}
	if !set {
		return nil
	}
	b.body[fieldName] = parsed
	b.addExplicitFlag(flagName)
	return nil
}

func (b *requestBodyBuilder) SetStructured(fieldName, flagName string, spec *JSONInput) error {
	value, set, err := b.input.readStructuredInput(spec, flagName)
	if err != nil {
		return err
	}
	if !set {
		return nil
	}
	b.body[fieldName] = value
	b.addExplicitFlag(flagName)
	return nil
}

func (b *requestBodyBuilder) SetObject(fieldName string, build func(*requestBodyBuilder) error) error {
	nested := newRequestBodyBuilder(b.input)
	if err := build(nested); err != nil {
		return err
	}
	if len(nested.body) == 0 {
		return nil
	}
	b.body[fieldName] = nested.body
	for _, flagName := range nested.explicitFlags {
		b.addExplicitFlag(flagName)
	}
	return nil
}

func (b *requestBodyBuilder) DecodeRequired(importFlagName string, importSpec *JSONInput, decodeFn func(*jx.Decoder) error) error {
	if data, set, err := b.input.readImportInput(importSpec, importFlagName); err != nil {
		return err
	} else if set {
		if err := b.ensureImportExclusive(importFlagName); err != nil {
			return err
		}
		if err := decodeBody(data, decodeFn); err != nil {
			return invalidUsage("invalid --%s JSON: %v", importFlagName, err)
		}
		return nil
	}

	if len(b.body) == 0 {
		return invalidUsage("request body is required: provide explicit input flags or --%s", importFlagName)
	}
	return decodeJSONNode(b.body, decodeFn)
}

func (b *requestBodyBuilder) DecodeOptional(importFlagName string, importSpec *JSONInput, decodeFn func(*jx.Decoder) error) (bool, error) {
	if data, set, err := b.input.readImportInput(importSpec, importFlagName); err != nil {
		return false, err
	} else if set {
		if err := b.ensureImportExclusive(importFlagName); err != nil {
			return false, err
		}
		if err := decodeBody(data, decodeFn); err != nil {
			return false, invalidUsage("invalid --%s JSON: %v", importFlagName, err)
		}
		return true, nil
	}

	if len(b.body) == 0 {
		return false, nil
	}
	return true, decodeJSONNode(b.body, decodeFn)
}

func (b *requestBodyBuilder) DecodeExplicitRequired(decodeFn func(*jx.Decoder) error) error {
	if len(b.body) == 0 {
		return invalidUsage("request body is required: provide explicit input flags")
	}
	return decodeJSONNode(b.body, decodeFn)
}

func (b *requestBodyBuilder) addExplicitFlag(flagName string) {
	b.explicitFlags = append(b.explicitFlags, flagName)
}

func (b *requestBodyBuilder) ensureImportExclusive(importFlagName string) error {
	if len(b.explicitFlags) == 0 {
		return nil
	}
	return invalidUsage("use either --%s or explicit input flags, not both (%s)", importFlagName, joinFlagNames(b.explicitFlags))
}

func decodeRequiredImportInput(input *requestInputReader, importFlagName string, importSpec *JSONInput, decodeFn func(*jx.Decoder) error) error {
	data, set, err := input.readImportInput(importSpec, importFlagName)
	if err != nil {
		return err
	}
	if !set {
		return invalidUsage("request body is required: provide --%s", importFlagName)
	}
	if err := decodeBody(data, decodeFn); err != nil {
		return invalidUsage("invalid --%s JSON: %v", importFlagName, err)
	}
	return nil
}

func (i *requestInputReader) readImportInput(spec *JSONInput, flagName string) ([]byte, bool, error) {
	return i.resolveInputSpec(spec, flagName)
}

func (i *requestInputReader) readStructuredInput(spec *JSONInput, flagName string) (any, bool, error) {
	data, set, err := i.resolveInputSpec(spec, flagName)
	if err != nil {
		return nil, false, err
	}
	if !set {
		return nil, false, nil
	}
	var value any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		return nil, false, invalidUsage("invalid --%s JSON: %v", flagName, err)
	}
	if dec.More() {
		return nil, false, invalidUsage("invalid --%s JSON: unexpected trailing data", flagName)
	}
	return value, true, nil
}

func (i *requestInputReader) resolveInputSpec(spec *JSONInput, flagName string) ([]byte, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	raw := strings.TrimSpace(string(*spec))
	if raw == "" {
		return nil, false, invalidUsage("--%s cannot be empty", flagName)
	}
	switch {
	case raw == "-":
		if i.stdinUsed {
			return nil, false, invalidUsage("stdin can only be consumed once")
		}
		i.stdinUsed = true
		data, err := io.ReadAll(i.runner.in)
		if err != nil {
			return nil, false, fmt.Errorf("read stdin for --%s: %w", flagName, err)
		}
		return data, true, nil
	case strings.HasPrefix(raw, "@"):
		path := strings.TrimSpace(strings.TrimPrefix(raw, "@"))
		if path == "" {
			return nil, false, invalidUsage("--%s file path cannot be empty", flagName)
		}
		data, err := readFile(path)
		if err != nil {
			return nil, false, fmt.Errorf("read --%s file: %w", flagName, err)
		}
		return data, true, nil
	default:
		return []byte(raw), true, nil
	}
}

func decodeJSONNode(node any, decodeFn func(*jx.Decoder) error) error {
	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	if err := decodeBody(data, decodeFn); err != nil {
		return invalidUsage("invalid request body: %v", err)
	}
	return nil
}

func readScalarInput(raw *string, kind scalarInputKind, nullable bool) (any, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, false, errors.New("value cannot be empty")
	}
	if nullable && strings.EqualFold(trimmed, scalarInputNullLiteral) {
		return nullScalarInput, true, nil
	}

	switch kind {
	case scalarInputString, scalarInputEnum:
		return trimmed, true, nil
	case scalarInputInt:
		value, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, false, fmt.Errorf("must be an integer: %w", err)
		}
		return value, true, nil
	case scalarInputFloat:
		value, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, false, fmt.Errorf("must be a number: %w", err)
		}
		return value, true, nil
	case scalarInputBool:
		value, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, false, fmt.Errorf("must be true or false: %w", err)
		}
		return value, true, nil
	case scalarInputDate:
		if _, err := time.Parse("2006-01-02", trimmed); err != nil {
			return nil, false, errors.New("expected YYYY-MM-DD")
		}
		return trimmed, true, nil
	case scalarInputDateTime:
		value, err := parseBodyDateTime(trimmed)
		if err != nil {
			return nil, false, err
		}
		return value, true, nil
	default:
		return nil, false, fmt.Errorf("unsupported input kind %q", kind)
	}
}

func parseBodyDateTime(raw string) (string, error) {
	if _, err := time.Parse(bodyDateTimeLayout, raw); err == nil {
		return raw, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return value.Format(bodyDateTimeLayout), nil
	}
	return "", errors.New("expected RFC3339 or YYYY-MM-DDTHH:MM:SS")
}

func readScalarInputArray(values []string, kind scalarInputKind) ([]any, bool, error) {
	if len(values) == 0 {
		return nil, false, nil
	}
	out := make([]any, 0, len(values))
	for _, raw := range values {
		value, _, err := readScalarInput(&raw, kind, false)
		if err != nil {
			return nil, false, err
		}
		out = append(out, value)
	}
	return out, true, nil
}

func joinFlagNames(flagNames []string) string {
	if len(flagNames) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(flagNames))
	out := make([]string, 0, len(flagNames))
	for _, flagName := range flagNames {
		if _, ok := seen[flagName]; ok {
			continue
		}
		seen[flagName] = struct{}{}
		out = append(out, "--"+flagName)
	}
	return strings.Join(out, ", ")
}
