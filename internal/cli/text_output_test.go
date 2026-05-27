//nolint:testpackage // Text renderer behavior is exercised through internal helpers.
package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/myyra/procountor/procountorapi"
)

func TestWriteTextOutputInvoiceUsesStructuredSections(t *testing.T) {
	t.Parallel()

	invoice := procountorapi.Invoice{
		Type:         procountorapi.InvoiceType("PURCHASE_INVOICE"),
		Date:         time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
		PaymentInfo:  procountorapi.PaymentInfo{},
		CounterParty: procountorapi.CounterParty{CounterPartyAddress: procountorapi.InvoiceCounterpartyAddress{Name: "Acme Oy"}},
		InvoiceRows: []procountorapi.InvoiceRow{{
			Product:    "Consulting",
			Quantity:   1,
			Unit:       "pcs",
			UnitPrice:  100,
			VatPercent: 25.5,
		}},
		InvoiceSumInfo: []procountorapi.InvoiceSumInfo{{}},
	}
	invoice.ID.SetTo(12345)
	invoice.Status.SetTo(procountorapi.InvoiceStatus("APPROVED"))
	invoice.InvoiceNumber.SetTo(987)
	invoice.InvoiceChannel.SetTo(procountorapi.InvoiceInvoiceChannel("EMAIL"))
	invoice.Version.SetTo(time.Date(2026, time.February, 2, 12, 0, 0, 0, time.UTC))
	invoice.InvoiceSumInfo[0].Currency.SetTo("EUR")
	invoice.InvoiceSumInfo[0].InvoiceSumTotal.SetTo(125.50)
	invoice.InvoiceSumInfo[0].ExcludingVatTotal.SetTo(100)
	invoice.InvoiceSumInfo[0].VatSumTotal.SetTo(25.50)
	invoice.InvoiceRows[0].Comment.SetTo("first row")

	var out bytes.Buffer
	if err := writeTextOutput(&out, invoice); err != nil {
		t.Fatalf("writeTextOutput: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"id: 12345",
		"status: APPROVED",
		"counterparty: Acme Oy",
		"total: 125.5 EUR",
		"invoice_rows:",
		"product",
		"Consulting",
		"invoice_sum_info:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n%s", want, got)
		}
	}
}

func TestWriteTextOutputLedgerReceiptUsesTransactionTable(t *testing.T) {
	t.Parallel()

	receipt := procountorapi.LedgerReceipt{
		ReceiptDate: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		Transactions: []procountorapi.Transaction{{
			TransactionType: procountorapi.TransactionTransactionType("ENTRY"),
			Account:         "3000",
			AccountingValue: 100,
			VatPercent:      procountorapi.NewOptFloat64(0),
		}},
	}
	receipt.ID.SetTo(678)
	receipt.Type = procountorapi.LedgerReceiptType("JOURNAL")
	receipt.Status.SetTo(procountorapi.LedgerReceiptStatus("APPROVED"))
	receipt.Name.SetTo("Manual adjustment")
	receipt.ReceiptNumber.SetTo(42)
	receipt.VatType.SetTo(procountorapi.LedgerReceiptVatType("SALES"))
	receipt.VatStatus.SetTo(1)
	receipt.Transactions[0].Description.SetTo("Revenue adj")

	var out bytes.Buffer
	if err := writeTextOutput(&out, receipt); err != nil {
		t.Fatalf("writeTextOutput: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"id: 678",
		"type: JOURNAL",
		"name: Manual adjustment",
		"transactions:",
		"transaction_type",
		"3000",
		"Revenue adj",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n%s", want, got)
		}
	}
}
