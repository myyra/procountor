//nolint:testpackage // Internal request-input helpers are validated directly.
package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-faster/jx"
)

func TestRequestInputReaderResolveInputSpec(t *testing.T) {
	t.Parallel()

	t.Run("inline JSON", func(t *testing.T) {
		t.Parallel()

		spec := JSONInput(`{"ok":true}`)
		reader := newRequestInputReader(&runner{in: bytes.NewBufferString("")})
		got, ok, err := reader.resolveInputSpec(&spec, "payload")
		if err != nil {
			t.Fatalf("resolveInputSpec returned error: %v", err)
		}
		if !ok {
			t.Fatal("expected spec to be set")
		}
		if string(got) != `{"ok":true}` {
			t.Fatalf("got %q, want %q", got, `{"ok":true}`)
		}
	})

	t.Run("@file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "payload.json")
		if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
			t.Fatalf("write payload: %v", err)
		}

		spec := JSONInput("@" + path)
		reader := newRequestInputReader(&runner{in: bytes.NewBufferString("")})
		got, ok, err := reader.resolveInputSpec(&spec, "payload")
		if err != nil {
			t.Fatalf("resolveInputSpec returned error: %v", err)
		}
		if !ok {
			t.Fatal("expected spec to be set")
		}
		if string(got) != `{"ok":true}` {
			t.Fatalf("got %q, want %q", got, `{"ok":true}`)
		}
	})

	t.Run("stdin only once", func(t *testing.T) {
		t.Parallel()

		first := JSONInput("-")
		second := JSONInput("-")
		reader := newRequestInputReader(&runner{in: bytes.NewBufferString(`{"from":"stdin"}`)})
		got, ok, err := reader.resolveInputSpec(&first, "payload")
		if err != nil {
			t.Fatalf("resolveInputSpec returned error: %v", err)
		}
		if !ok {
			t.Fatal("expected spec to be set")
		}
		if string(got) != `{"from":"stdin"}` {
			t.Fatalf("got %q, want %q", got, `{"from":"stdin"}`)
		}

		_, _, err = reader.resolveInputSpec(&second, "other")
		if err == nil {
			t.Fatal("expected second stdin read to fail")
		}
		var usageErr *usageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("error = %T, want *usageError", err)
		}
	})
}

func TestRequestBodyBuilderDecodeRequiredMutualExclusion(t *testing.T) {
	t.Parallel()

	name := "example"
	thing := JSONInput(`{"name":"example"}`)
	builder := newRequestBodyBuilder(newRequestInputReader(&runner{in: bytes.NewBufferString("")}))
	if err := builder.SetScalar("name", "name", &name, scalarInputString, false); err != nil {
		t.Fatalf("SetScalar returned error: %v", err)
	}

	err := builder.DecodeRequired("thing", &thing, func(d *jx.Decoder) error { return nil })
	if err == nil {
		t.Fatal("expected mutual exclusion error")
	}
}

func TestReadScalarInputNullable(t *testing.T) {
	t.Parallel()

	raw := "null"
	value, set, err := readScalarInput(&raw, scalarInputFloat, true)
	if err != nil {
		t.Fatalf("readScalarInput returned error: %v", err)
	}
	if !set {
		t.Fatal("expected value to be set")
	}
	if value != nullScalarInput {
		t.Fatalf("value = %#v, want nullScalarInput sentinel", value)
	}
}

func TestFormatJSONInputHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		help string
		want string
	}{
		{
			name: "adds period when missing",
			help: "Full invoice JSON document",
			want: "Full invoice JSON document. Accepts inline JSON, @file, or - for stdin.",
		},
		{
			name: "preserves existing punctuation",
			help: "Invoice JSON.",
			want: "Invoice JSON. Accepts inline JSON, @file, or - for stdin.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatJSONInputHelp(tt.help); got != tt.want {
				t.Fatalf("formatJSONInputHelp(%q) = %q, want %q", tt.help, got, tt.want)
			}
		})
	}
}
