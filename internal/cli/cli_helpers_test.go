//nolint:testpackage // Internal helper behavior is validated directly.
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/go-faster/jx"

	"github.com/myyra/procountor/procountorapi"
)

func TestDecodeBodyTrailingData(t *testing.T) {
	t.Parallel()

	err := decodeBody([]byte(`{"x":1}{"y":2}`), func(d *jx.Decoder) error {
		return d.Skip()
	})
	if err == nil {
		t.Fatal("expected trailing data error")
	}
	if !strings.Contains(err.Error(), "unexpected trailing data") {
		t.Fatalf("error = %q, want trailing data error", err.Error())
	}
}

func TestSchemaFlagPresence(t *testing.T) {
	t.Parallel()

	r := &runner{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}}
	parser, _, err := newParser(r, true)
	if err != nil {
		t.Fatalf("newParser: %v", err)
	}

	tests := []struct {
		path string
		flag string
	}{
		{path: "attachments content", flag: "out"},
		{path: "business-partners patch", flag: "business-partner"},
		{path: "business-partners search", flag: "paginate"},
		{path: "cost-receipts search", flag: "paginate"},
		{path: "invoices image", flag: "out"},
		{path: "invoices payment-events search", flag: "paginate"},
		{path: "ledger-receipts search", flag: "paginate"},
		{path: "persons list", flag: "paginate"},
		{path: "persons update", flag: "person"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			node, err := findCommandNode(parser.Model.Node, strings.Fields(tt.path))
			if err != nil {
				t.Fatalf("findCommandNode(%q): %v", tt.path, err)
			}
			flags := schemaFlags(node, false)
			for _, flag := range flags {
				if flag.Name == tt.flag {
					return
				}
			}
			t.Fatalf("command %q missing --%s", tt.path, tt.flag)
		})
	}
}

func TestSchemaFlagAbsence(t *testing.T) {
	t.Parallel()

	r := &runner{out: &bytes.Buffer{}, errOut: &bytes.Buffer{}}
	parser, _, err := newParser(r, true)
	if err != nil {
		t.Fatalf("newParser: %v", err)
	}

	tests := []struct {
		path string
		flag string
	}{
		{path: "attachments rename", flag: "attachment-rename"},
		{path: "attachments save", flag: "attachment-meta"},
		{path: "business-partners patch", flag: "name"},
		{path: "dimensions add-item", flag: "dimension-item"},
		{path: "dimensions update", flag: "dimension-name"},
		{path: "dimensions update-item", flag: "dimension-item"},
		{path: "invoices add-notes", flag: "invoice-notes"},
		{path: "invoices approve", flag: "checking-event"},
		{path: "invoices comments create", flag: "invoice-comment"},
		{path: "invoices payment-events mark-paid", flag: "payment-event"},
		{path: "invoices reject", flag: "checking-event"},
		{path: "invoices verify", flag: "checking-event"},
		{path: "persons update", flag: "first-name"},
		{path: "reports accounting", flag: "accounting-report"},
		{path: "reports general-ledger", flag: "general-ledger-report"},
		{path: "reports ledger-accounts", flag: "ledger-accounts-report"},
	}

	for _, tt := range tests {
		t.Run(tt.path+" --"+tt.flag, func(t *testing.T) {
			t.Parallel()

			node, err := findCommandNode(parser.Model.Node, strings.Fields(tt.path))
			if err != nil {
				t.Fatalf("findCommandNode(%q): %v", tt.path, err)
			}
			flags := schemaFlags(node, false)
			for _, flag := range flags {
				if flag.Name == tt.flag {
					t.Fatalf("command %q unexpectedly exposes --%s", tt.path, tt.flag)
				}
			}
		})
	}
}

func TestWriteErrorFormats(t *testing.T) {
	t.Parallel()

	t.Run("invalid usage", func(t *testing.T) {
		t.Parallel()

		var errOut bytes.Buffer
		r := &runner{errOut: &errOut}
		r.writeError(invalidUsage("bad input"))

		if got := strings.TrimSpace(errOut.String()); got != "invalid usage: bad input" {
			t.Fatalf("stderr = %q", got)
		}
	})

	t.Run("internal", func(t *testing.T) {
		t.Parallel()

		var errOut bytes.Buffer
		r := &runner{errOut: &errOut}
		r.writeError(errors.New("boom"))

		if got := strings.TrimSpace(errOut.String()); got != "boom" {
			t.Fatalf("stderr = %q", got)
		}
	})

	t.Run("api", func(t *testing.T) {
		t.Parallel()

		var errOut bytes.Buffer
		r := &runner{errOut: &errOut}
		r.writeError(&procountorapi.ErrorResponseStatusCode{
			StatusCode: 400,
			Response: procountorapi.ErrorResponse{
				Message: procountorapi.NewStringErrorResponseMessage("bad request"),
			},
		})

		if got := strings.TrimSpace(errOut.String()); !strings.Contains(got, "api error 400") {
			t.Fatalf("stderr = %q", got)
		}
	})
}
