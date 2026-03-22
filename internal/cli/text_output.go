package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/myyra/procountor/procountorapi"
)

type textPair struct {
	key   string
	value string
}

const (
	textTablePadding = 4
	textIndentStep   = 2
)

func writeTextOutput(w io.Writer, v any) error {
	var sb strings.Builder
	ok, err := renderTypedText(&sb, v)
	if err != nil {
		return err
	}
	if !ok {
		normalized, err := normalizeOutputValue(v)
		if err != nil {
			return err
		}
		renderStructuredText(&sb, normalized, 0)
	}
	sb.WriteByte('\n')
	_, err = io.WriteString(w, sb.String())
	return wrapErr("write text output", err)
}

func renderTypedText(sb *strings.Builder, v any) (bool, error) {
	switch v.(type) {
	case procountorapi.InfoMessage, *procountorapi.InfoMessage:
		return true, renderInfoMessageText(sb, v)
	case procountorapi.InvoiceIds, *procountorapi.InvoiceIds:
		return true, renderInvoiceIDsText(sb, v)
	case []procountorapi.InvoiceBasicInfo:
		return true, renderInvoiceBasicInfoListText(sb, v)
	case procountorapi.Invoice, *procountorapi.Invoice:
		return true, renderInvoiceText(sb, v)
	case procountorapi.InvoiceTransactions, *procountorapi.InvoiceTransactions:
		return true, renderInvoiceTransactionsText(sb, v)
	case []procountorapi.LedgerReceiptBasicInfo:
		return true, renderLedgerReceiptBasicInfoListText(sb, v)
	case procountorapi.LedgerReceipt, *procountorapi.LedgerReceipt:
		return true, renderLedgerReceiptText(sb, v)
	case []procountorapi.PaymentEvent:
		return true, renderPaymentEventListText(sb, v)
	case procountorapi.PaymentEvent, *procountorapi.PaymentEvent:
		return true, renderPaymentEventText(sb, v)
	case []procountorapi.BusinessPartnerBasicInfo:
		return true, renderBusinessPartnerBasicInfoListText(sb, v)
	case []procountorapi.PersonBasicInfo:
		return true, renderPersonBasicInfoListText(sb, v)
	case []procountorapi.Dimension:
		return true, renderDimensionListText(sb, v)
	case procountorapi.Dimension, *procountorapi.Dimension:
		return true, renderDimensionText(sb, v)
	case procountorapi.DimensionItem, *procountorapi.DimensionItem:
		return true, renderDimensionItemText(sb, v)
	case procountorapi.DimensionName, *procountorapi.DimensionName:
		return true, renderDimensionNameText(sb, v)
	case []procountorapi.CostReceipt:
		return true, renderCostReceiptListText(sb, v)
	case procountorapi.CostReceipt, *procountorapi.CostReceipt:
		return true, renderCostReceiptText(sb, v)
	case procountorapi.AccountingReportResponse, *procountorapi.AccountingReportResponse:
		return true, renderReportResponseText(sb, v, "reportParameters", "reportData")
	case procountorapi.GeneralLedgerReportResponse, *procountorapi.GeneralLedgerReportResponse:
		return true, renderReportResponseText(sb, v, "reportParameters", "ledgerAccount")
	case procountorapi.LedgerAccountsReportResponse, *procountorapi.LedgerAccountsReportResponse:
		return true, renderReportResponseText(sb, v, "reportParameters", "reportData")
	default:
		return false, nil
	}
}

func renderInfoMessageText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	message := scalarString(lookupKey(asMap(normalized), "message"))
	if message == "" {
		renderStructuredText(sb, normalized, 0)
		return nil
	}
	sb.WriteString(message)
	return nil
}

func renderInvoiceIDsText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	ids := asSlice(lookupKey(asMap(normalized), "invoiceIds"))
	if len(ids) == 0 {
		sb.WriteString("[]")
		return nil
	}
	for idx, id := range ids {
		if idx > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(displayScalar(id))
	}
	return nil
}

func renderInvoiceBasicInfoListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "invoiceNumber")),
			scalarString(lookupKey(item, "status")),
			scalarString(lookupKey(item, "type")),
			scalarString(lookupKey(item, "date")),
			scalarString(lookupKey(item, "dueDate")),
			counterPartyName(lookupKey(item, "counterParty")),
			invoiceTotalText(item),
		})
	}
	renderTable(sb, 0, []string{"id", "invoice_number", "status", "type", "date", "due_date", "counterparty", "total"}, rows)
	return nil
}

func renderInvoiceText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	invoice := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(invoice, "id"))},
		{key: "status", value: scalarString(lookupKey(invoice, "status"))},
		{key: "type", value: scalarString(lookupKey(invoice, "type"))},
		{key: "date", value: scalarString(lookupKey(invoice, "date"))},
		{key: "payment_date", value: scalarString(lookupKey(invoice, "paymentDate"))},
		{key: "invoice_number", value: scalarString(lookupKey(invoice, "invoiceNumber"))},
		{key: "original_invoice_number", value: scalarString(lookupKey(invoice, "originalInvoiceNumber"))},
		{key: "invoice_channel", value: scalarString(lookupKey(invoice, "invoiceChannel"))},
		{key: "language", value: scalarString(lookupKey(invoice, "language"))},
		{key: "ledger_receipt_id", value: scalarString(lookupKey(invoice, "ledgerReceiptId"))},
		{key: "counterparty", value: counterPartyName(lookupKey(invoice, "counterParty"))},
		{key: "total", value: invoiceTotalText(invoice)},
		{key: "version", value: scalarString(lookupKey(invoice, "version"))},
	})
	appendSectionValue(sb, "payment_info", lookupKey(invoice, "paymentInfo"))
	appendSectionValue(sb, "counter_party", lookupKey(invoice, "counterParty"))
	appendSectionValue(sb, "billing_address", lookupKey(invoice, "billingAddress"))
	appendSectionValue(sb, "delivery_address", lookupKey(invoice, "deliveryAddress"))
	appendSectionValue(sb, "invoice_sum_info", invoiceSumTableValue(lookupKey(invoice, "invoiceSumInfo")))
	appendSectionValue(sb, "invoice_rows", invoiceRowsTableValue(lookupKey(invoice, "invoiceRows")))
	appendSectionValue(sb, "travel_information_items", lookupKey(invoice, "travelInformationItems"))
	appendSectionValue(sb, "attachments", attachmentTableValue(lookupKey(invoice, "attachments")))
	appendSectionValue(sb, "notes", lookupKey(invoice, "notes"))
	return nil
}

func renderInvoiceTransactionsText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	root := asMap(normalized)
	transactions := asSlice(lookupKey(root, "transactions"))
	if len(transactions) == 0 {
		renderStructuredText(sb, normalized, 0)
		return nil
	}
	rows := make([][]string, 0, len(transactions))
	for _, raw := range transactions {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "date")),
			scalarString(lookupKey(item, "action")),
			scalarString(lookupKey(item, "name")),
			scalarString(lookupKey(item, "details")),
		})
	}
	renderTable(sb, 0, []string{"date", "action", "name", "details"}, rows)
	return nil
}

func renderLedgerReceiptBasicInfoListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "receiptNumber")),
			scalarString(lookupKey(item, "invoiceNumber")),
			scalarString(lookupKey(item, "status")),
			scalarString(lookupKey(item, "type")),
			scalarString(lookupKey(item, "receiptDate")),
			scalarString(lookupKey(item, "name")),
		})
	}
	renderTable(sb, 0, []string{"id", "receipt_number", "invoice_number", "status", "type", "receipt_date", "name"}, rows)
	return nil
}

func renderLedgerReceiptText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	receipt := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(receipt, "id"))},
		{key: "status", value: scalarString(lookupKey(receipt, "status"))},
		{key: "type", value: scalarString(lookupKey(receipt, "type"))},
		{key: "name", value: scalarString(lookupKey(receipt, "name"))},
		{key: "receipt_date", value: scalarString(lookupKey(receipt, "receiptDate"))},
		{key: "receipt_number", value: scalarString(lookupKey(receipt, "receiptNumber"))},
		{key: "invoice_id", value: scalarString(lookupKey(receipt, "invoiceId"))},
		{key: "invoice_number", value: scalarString(lookupKey(receipt, "invoiceNumber"))},
		{key: "vat_type", value: scalarString(lookupKey(receipt, "vatType"))},
		{key: "vat_status", value: scalarString(lookupKey(receipt, "vatStatus"))},
		{key: "transaction_description", value: scalarString(lookupKey(receipt, "transactionDescription"))},
		{key: "version", value: scalarString(lookupKey(receipt, "version"))},
	})
	appendSectionValue(sb, "transactions", ledgerTransactionTableValue(lookupKey(receipt, "transactions")))
	appendLedgerTransactionExtras(sb, asSlice(lookupKey(receipt, "transactions")))
	appendSectionValue(sb, "attachments", attachmentTableValue(lookupKey(receipt, "attachments")))
	appendSectionValue(sb, "accountants_notes", lookupKey(receipt, "accountantsNotes"))
	appendSectionValue(sb, "invoice_notes", lookupKey(receipt, "invoiceNotes"))
	return nil
}

func renderPaymentEventListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "invoiceId")),
			scalarString(lookupKey(item, "status")),
			scalarString(lookupKey(item, "type")),
			scalarString(lookupKey(item, "paymentDate")),
			moneyText(lookupKey(item, "amount"), lookupKey(item, "currency")),
			moneyText(lookupKey(item, "paidAmount"), lookupKey(item, "paidCurrency")),
			scalarString(lookupKey(item, "description")),
		})
	}
	renderTable(sb, 0, []string{"id", "invoice_id", "status", "type", "payment_date", "amount", "paid_amount", "description"}, rows)
	return nil
}

func renderPaymentEventText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	event := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(event, "id"))},
		{key: "invoice_id", value: scalarString(lookupKey(event, "invoiceId"))},
		{key: "status", value: scalarString(lookupKey(event, "status"))},
		{key: "type", value: scalarString(lookupKey(event, "type"))},
		{key: "payment_date", value: scalarString(lookupKey(event, "paymentDate"))},
		{key: "amount", value: moneyText(lookupKey(event, "amount"), lookupKey(event, "currency"))},
		{key: "paid_amount", value: moneyText(lookupKey(event, "paidAmount"), lookupKey(event, "paidCurrency"))},
		{key: "description", value: scalarString(lookupKey(event, "description"))},
	})
	return nil
}

func renderBusinessPartnerBasicInfoListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "name")),
			scalarString(lookupKey(item, "type")),
			scalarString(lookupPath(item, "registryInfo", "customerNumber")),
			scalarString(lookupPath(item, "invoicingInfo", "identifier")),
			scalarString(lookupKey(item, "version")),
		})
	}
	renderTable(sb, 0, []string{"id", "name", "type", "customer_number", "identifier", "version"}, rows)
	return nil
}

func renderPersonBasicInfoListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			firstNonEmpty(scalarString(lookupKey(item, "firstName")), ""),
			scalarString(lookupKey(item, "lastName")),
			scalarString(lookupPath(item, "invoicingInfo", "personNumber")),
			scalarString(lookupPath(item, "invoicingInfo", "identifier")),
		})
	}
	renderTable(sb, 0, []string{"id", "first_name", "last_name", "person_number", "identifier"}, rows)
	return nil
}

func renderDimensionListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "name")),
			strconv.Itoa(len(asSlice(lookupKey(item, "items")))),
		})
	}
	renderTable(sb, 0, []string{"id", "name", "item_count"}, rows)
	return nil
}

func renderDimensionText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	dimension := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(dimension, "id"))},
		{key: "name", value: scalarString(lookupKey(dimension, "name"))},
	})
	appendSectionValue(sb, "items", dimensionItemsTableValue(lookupKey(dimension, "items")))
	return nil
}

func renderDimensionItemText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	item := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(item, "id"))},
		{key: "code_name", value: scalarString(lookupKey(item, "codeName"))},
		{key: "status", value: scalarString(lookupKey(item, "status"))},
		{key: "description", value: scalarString(lookupKey(item, "description"))},
	})
	return nil
}

func renderDimensionNameText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	item := asMap(normalized)
	appendPairs(sb, []textPair{{key: "name", value: scalarString(lookupKey(item, "name"))}})
	return nil
}

func renderCostReceiptListText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	items := asSlice(normalized)
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, []string{
			scalarString(lookupKey(item, "id")),
			scalarString(lookupKey(item, "title")),
			scalarString(lookupKey(item, "receiptDate")),
			scalarString(lookupKey(item, "currency")),
			scalarString(lookupKey(item, "processed")),
			strconv.Itoa(len(asSlice(lookupKey(item, "rows")))),
		})
	}
	renderTable(sb, 0, []string{"id", "title", "receipt_date", "currency", "processed", "row_count"}, rows)
	return nil
}

func renderCostReceiptText(sb *strings.Builder, v any) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	receipt := asMap(normalized)
	appendPairs(sb, []textPair{
		{key: "id", value: scalarString(lookupKey(receipt, "id"))},
		{key: "title", value: scalarString(lookupKey(receipt, "title"))},
		{key: "receipt_date", value: scalarString(lookupKey(receipt, "receiptDate"))},
		{key: "currency", value: scalarString(lookupKey(receipt, "currency"))},
		{key: "currency_rate", value: scalarString(lookupKey(receipt, "currencyRate"))},
		{key: "payment_method", value: scalarString(lookupKey(receipt, "paymentMethod"))},
		{key: "processed", value: scalarString(lookupKey(receipt, "processed"))},
	})
	appendSectionValue(sb, "description", lookupKey(receipt, "description"))
	appendSectionValue(sb, "rows", costReceiptRowsTableValue(lookupKey(receipt, "rows")))
	appendSectionValue(sb, "attachments", attachmentTableValue(lookupKey(receipt, "attachments")))
	return nil
}

func renderReportResponseText(sb *strings.Builder, v any, paramsKey, dataKey string) error {
	normalized, err := normalizeOutputValue(v)
	if err != nil {
		return err
	}
	report := asMap(normalized)
	appendSectionValue(sb, "report_parameters", lookupKey(report, paramsKey))
	appendSectionValue(sb, "report_data", lookupKey(report, dataKey))
	return nil
}

func appendLedgerTransactionExtras(sb *strings.Builder, transactions []any) {
	for idx, raw := range transactions {
		transaction := asMap(raw)
		extras := map[string]any{}
		if dims := lookupKey(transaction, "dimensionItemValues"); !isEmptyValue(dims) {
			extras["dimensionItemValues"] = dims
		}
		if allocations := lookupKey(transaction, "allocations"); !isEmptyValue(allocations) {
			extras["allocations"] = allocations
		}
		if len(extras) == 0 {
			continue
		}
		appendSectionValue(sb, fmt.Sprintf("transaction_%d_extras", idx+1), extras)
	}
}

func appendPairs(sb *strings.Builder, pairs []textPair) {
	first := true
	for _, pair := range pairs {
		if strings.TrimSpace(pair.value) == "" {
			continue
		}
		if !first {
			sb.WriteByte('\n')
		}
		first = false
		sb.WriteString(pair.key)
		sb.WriteString(": ")
		sb.WriteString(pair.value)
	}
	if first {
		sb.WriteString("{}")
	}
}

func appendSectionValue(sb *strings.Builder, title string, value any) {
	if isEmptyValue(value) {
		return
	}
	if sb.Len() > 0 {
		sb.WriteString("\n\n")
	}
	sb.WriteString(title)
	sb.WriteString(":\n")
	renderStructuredText(sb, value, textIndentStep)
}

func renderStructuredText(sb *strings.Builder, v any, indent int) {
	switch value := v.(type) {
	case map[string]any:
		renderStructuredObject(sb, value, indent)
	case []any:
		renderStructuredArray(sb, value, indent)
	default:
		writeIndent(sb, indent)
		sb.WriteString(displayScalar(value))
	}
}

func renderStructuredObject(sb *strings.Builder, value map[string]any, indent int) {
	if len(value) == 0 {
		writeIndent(sb, indent)
		sb.WriteString("{}")
		return
	}
	scalarKeys := make([]string, 0, len(value))
	compositeKeys := make([]string, 0, len(value))
	for _, key := range orderedKeys(value) {
		if isCompositeValue(value[key]) {
			compositeKeys = append(compositeKeys, key)
		} else {
			scalarKeys = append(scalarKeys, key)
		}
	}
	for idx, key := range scalarKeys {
		if idx > 0 {
			sb.WriteByte('\n')
		}
		writeIndent(sb, indent)
		sb.WriteString(displayKey(key))
		sb.WriteString(": ")
		sb.WriteString(displayScalar(value[key]))
	}
	if len(scalarKeys) > 0 && len(compositeKeys) > 0 {
		sb.WriteString("\n\n")
	}
	for idx, key := range compositeKeys {
		if idx > 0 {
			sb.WriteString("\n\n")
		}
		writeIndent(sb, indent)
		sb.WriteString(displayKey(key))
		sb.WriteString(":\n")
		renderStructuredText(sb, value[key], indent+textIndentStep)
	}
}

func renderStructuredArray(sb *strings.Builder, value []any, indent int) {
	if len(value) == 0 {
		writeIndent(sb, indent)
		sb.WriteString("[]")
		return
	}
	if headers, rows, ok := asTable(value); ok {
		renderTable(sb, indent, headers, rows)
		return
	}
	if allScalar(value) {
		for idx, item := range value {
			if idx > 0 {
				sb.WriteByte('\n')
			}
			writeIndent(sb, indent)
			sb.WriteString("- ")
			sb.WriteString(displayScalar(item))
		}
		return
	}
	for idx, item := range value {
		if idx > 0 {
			sb.WriteString("\n\n")
		}
		writeIndent(sb, indent)
		sb.WriteString(strconv.Itoa(idx + 1))
		sb.WriteString(":\n")
		renderStructuredText(sb, item, indent+textIndentStep)
	}
}

func asTable(items []any) ([]string, [][]string, bool) {
	if len(items) == 0 {
		return nil, nil, false
	}
	keysSet := map[string]struct{}{}
	for _, item := range items {
		row := asMap(item)
		if row == nil {
			return nil, nil, false
		}
		for key, value := range row {
			if isCompositeValue(value) {
				return nil, nil, false
			}
			keysSet[key] = struct{}{}
		}
	}
	headers := make([]string, 0, len(keysSet))
	for key := range keysSet {
		headers = append(headers, key)
	}
	sort.Slice(headers, func(i, j int) bool {
		return keyRank(headers[i]) < keyRank(headers[j]) || keyRank(headers[i]) == keyRank(headers[j]) && headers[i] < headers[j]
	})
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rowMap := asMap(item)
		row := make([]string, 0, len(headers))
		for _, header := range headers {
			row = append(row, displayScalar(rowMap[header]))
		}
		rows = append(rows, row)
	}
	for idx, header := range headers {
		headers[idx] = displayKey(header)
	}
	return headers, rows, true
}

func renderTable(sb *strings.Builder, indent int, headers []string, rows [][]string) {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, textTablePadding, textIndentStep, ' ', 0)
	_, _ = fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
	writeIndentedBlock(sb, indent, strings.TrimRight(buf.String(), "\n"))
}

func writeIndentedBlock(sb *strings.Builder, indent int, block string) {
	lines := strings.Split(block, "\n")
	for idx, line := range lines {
		if idx > 0 {
			sb.WriteByte('\n')
		}
		writeIndent(sb, indent)
		sb.WriteString(line)
	}
}

func orderedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keyRank(keys[i]) < keyRank(keys[j]) || keyRank(keys[i]) == keyRank(keys[j]) && keys[i] < keys[j]
	})
	return keys
}

func keyRank(key string) int {
	order := []string{
		"ok", "removed", "logged_in", "auth_file", "company_id", "company",
		"access_token_present", "refresh_token_present", "expires_at", "expired", "updated_at", "path",
		"id", "invoiceId", "invoiceNumber", "receiptNumber", "name", "title", "status", "type",
		"date", "receiptDate", "paymentDate", "dueDate", "created", "version", "description", "message",
	}
	for idx, candidate := range order {
		if key == candidate {
			return idx
		}
	}
	return len(order) + 1
}

func displayKey(key string) string {
	var sb strings.Builder
	for idx, r := range key {
		if idx > 0 && r >= 'A' && r <= 'Z' {
			sb.WriteByte('_')
		}
		sb.WriteRune(r)
	}
	return strings.ToLower(sb.String())
}

func displayScalar(v any) string {
	switch value := v.(type) {
	case nil:
		return "null"
	case string:
		return displayString(value)
	case json.Number:
		return value.String()
	case bool:
		return strconv.FormatBool(value)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 32)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case []any:
		if allScalar(value) {
			parts := make([]string, 0, len(value))
			for _, item := range value {
				parts = append(parts, displayScalar(item))
			}
			return strings.Join(parts, ", ")
		}
		b, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(b)
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(b)
	}
}

func displayString(value string) string {
	switch {
	case value == "":
		return `""`
	case strings.ContainsRune(value, '\n'):
		return strconv.Quote(value)
	case strings.TrimSpace(value) != value:
		return strconv.Quote(value)
	default:
		return value
	}
}

func scalarString(v any) string {
	if isEmptyValue(v) {
		return ""
	}
	return displayScalar(v)
}

func lookupKey(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func lookupPath(v any, path ...string) any {
	current := v
	for _, part := range path {
		current = lookupKey(asMap(current), part)
	}
	return current
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func asSlice(v any) []any {
	if items, ok := v.([]any); ok {
		return items
	}
	return nil
}

func allScalar(items []any) bool {
	for _, item := range items {
		if isCompositeValue(item) {
			return false
		}
	}
	return true
}

func isCompositeValue(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func isEmptyValue(v any) bool {
	switch value := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(value) == ""
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	default:
		return false
	}
}

func counterPartyName(v any) string {
	name := scalarString(lookupPath(v, "counterPartyAddress", "name"))
	if name != "" {
		return name
	}
	return firstNonEmpty(
		scalarString(lookupKey(asMap(v), "name")),
		scalarString(lookupKey(asMap(v), "contactPersonName")),
		scalarString(lookupKey(asMap(v), "identifier")),
	)
}

func invoiceTotalText(invoice map[string]any) string {
	items := asSlice(lookupKey(invoice, "invoiceSumInfo"))
	if len(items) == 0 {
		return ""
	}
	var selected map[string]any
	for _, raw := range items {
		item := asMap(raw)
		if item == nil {
			continue
		}
		if scalarString(lookupKey(item, "inAccountingCurrency")) == "true" {
			selected = item
			break
		}
		if selected == nil {
			selected = item
		}
	}
	if selected == nil {
		return ""
	}
	return moneyText(lookupKey(selected, "invoiceSumTotal"), lookupKey(selected, "currency"))
}

func moneyText(amount, currency any) string {
	amountText := scalarString(amount)
	currencyText := scalarString(currency)
	switch {
	case amountText == "":
		return ""
	case currencyText == "":
		return amountText
	default:
		return amountText + " " + currencyText
	}
}

func invoiceSumTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"currency":             lookupKey(item, "currency"),
			"inAccountingCurrency": lookupKey(item, "inAccountingCurrency"),
			"excludingVatTotal":    lookupKey(item, "excludingVatTotal"),
			"vatSumTotal":          lookupKey(item, "vatSumTotal"),
			"invoiceSumTotal":      lookupKey(item, "invoiceSumTotal"),
		})
	}
	return rows
}

func invoiceRowsTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"product":         lookupKey(item, "product"),
			"quantity":        lookupKey(item, "quantity"),
			"unit":            lookupKey(item, "unit"),
			"unitPrice":       lookupKey(item, "unitPrice"),
			"vatPercent":      lookupKey(item, "vatPercent"),
			"discountPercent": lookupKey(item, "discountPercent"),
			"comment":         lookupKey(item, "comment"),
		})
	}
	return rows
}

func ledgerTransactionTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"id":              lookupKey(item, "id"),
			"transactionType": lookupKey(item, "transactionType"),
			"account":         lookupKey(item, "account"),
			"accountingValue": lookupKey(item, "accountingValue"),
			"vatPercent":      lookupKey(item, "vatPercent"),
			"vatStatus":       lookupKey(item, "vatStatus"),
			"description":     lookupKey(item, "description"),
			"balanceCode":     lookupKey(item, "balanceCode"),
		})
	}
	return rows
}

func attachmentTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"id":              lookupKey(item, "id"),
			"name":            lookupKey(item, "name"),
			"referenceType":   lookupKey(item, "referenceType"),
			"referenceId":     lookupKey(item, "referenceId"),
			"mimeType":        lookupKey(item, "mimeType"),
			"sendWithInvoice": lookupKey(item, "sendWithInvoice"),
		})
	}
	return rows
}

func dimensionItemsTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"id":          lookupKey(item, "id"),
			"codeName":    lookupKey(item, "codeName"),
			"status":      lookupKey(item, "status"),
			"description": lookupKey(item, "description"),
		})
	}
	return rows
}

func costReceiptRowsTableValue(v any) any {
	items := asSlice(v)
	if len(items) == 0 {
		return nil
	}
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		item := asMap(raw)
		rows = append(rows, map[string]any{
			"productId":  lookupKey(item, "productId"),
			"price":      lookupKey(item, "price"),
			"vatPercent": lookupKey(item, "vatPercent"),
		})
	}
	return rows
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
