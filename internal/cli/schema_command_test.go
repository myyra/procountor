package cli_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSchemaCommandReturnsJSONBodySchemaForWholeDocumentCommand(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runCLIJSON(t, []string{"schema", "invoices", "save"})
	assertCommandSucceeded(t, code, stderr)

	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &doc); err != nil {
		t.Fatalf("decode schema output: %v\noutput: %s", err, stdout)
	}

	if got := doc["$schema"]; got != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("$schema = %v", got)
	}
	if got := doc["$ref"]; got != "#/$defs/Invoice" {
		t.Fatalf("$ref = %v", got)
	}

	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		t.Fatalf("$defs missing or wrong type: %#v", doc["$defs"])
	}
	invoice, ok := defs["Invoice"].(map[string]any)
	if !ok {
		t.Fatalf("Invoice schema missing from $defs")
	}
	if got := invoice["type"]; got != "object" {
		t.Fatalf("Invoice.type = %v, want object", got)
	}
}

func TestSchemaCommandReturnsJSONBodySchemaForFlagBuiltCommand(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runCLIJSON(t, []string{"schema", "reports", "accounting"})
	assertCommandSucceeded(t, code, stderr)

	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &doc); err != nil {
		t.Fatalf("decode schema output: %v\noutput: %s", err, stdout)
	}

	defs := doc["$defs"].(map[string]any)
	req := defs["AccountingReportRequest"].(map[string]any)
	required := req["required"].([]any)
	if !containsString(required, "endDate") {
		t.Fatalf("required = %#v, want endDate", required)
	}

	properties := req["properties"].(map[string]any)
	if _, ok := properties["options"].(map[string]any); !ok {
		t.Fatalf("options property missing from AccountingReportRequest schema")
	}
}

func TestSchemaCommandReturnsJSONBodySchemaForOptionalBodyCommand(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runCLIJSON(t, []string{"schema", "invoices", "approve"})
	assertCommandSucceeded(t, code, stderr)

	var doc map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &doc); err != nil {
		t.Fatalf("decode schema output: %v\noutput: %s", err, stdout)
	}

	if got := doc["$ref"]; got != "#/$defs/CheckingEvent" {
		t.Fatalf("$ref = %v", got)
	}
}

func TestSchemaCommandReturnsNothingForCommandsWithoutJSONBody(t *testing.T) {
	t.Parallel()

	tests := [][]string{
		{"schema", "invoices", "payment-events"},
		{"schema", "attachments", "save"},
		{"schema"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Parallel()

			code, stdout, stderr := runCLIJSON(t, args)
			assertCommandSucceeded(t, code, stderr)
			if strings.TrimSpace(stdout) != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
		})
	}
}

func containsString(values []any, want string) bool {
	for _, value := range values {
		if s, ok := value.(string); ok && s == want {
			return true
		}
	}
	return false
}
