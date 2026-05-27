package cli

import (
	"context"

	"github.com/myyra/procountor/procountorapi"
)

type ReportsCmd struct {
	Accounting     ReportsAccountingCmd     `cmd:"" help:"Generate an accounting report."`
	GeneralLedger  ReportsGeneralLedgerCmd  `cmd:"" help:"Generate a general ledger report for one account." name:"general-ledger"`
	LedgerAccounts ReportsLedgerAccountsCmd `cmd:"" help:"Generate a ledger accounts report."                name:"ledger-accounts"`
}

type receiptTypeFlag struct {
	ReceiptType []string `help:"Receipt types: SALES_INVOICE, PURCHASE_INVOICE, TRAVEL_INVOICE, BILL_OF_CHARGES, JOURNAL, SALARY, VAT_FORM, EMPLOYER_CONTRIBUTION, PERIODIC_TAX_RETURN, VAT_SUMMARY, SALES_ORDER, PURCHASE_ORDER, REFERENCE_PAYMENT, BANK_STATEMENT_AS_RECEIPT, RECEIPT_FOR_OPENING_ACCOUNTS. Repeat the flag for multiple values." name:"receipt-type"`
}

type accountingReportOptionsFlags struct {
	receiptTypeFlag     `embed:""`
	ReceiptCurrency     *string `help:"Receipt currency."                     name:"receipt-currency"`
	ReceiptName         *string `help:"Receipt name."                         name:"receipt-name"`
	EntryPeriodStart    *string `help:"Entry period start date (YYYY-MM-DD)." name:"entry-period-start"`
	EntryPeriodEnd      *string `help:"Entry period end date (YYYY-MM-DD)."   name:"entry-period-end"`
	TransactionValue    *string `help:"Transaction value."                    name:"transaction-value"`
	TransactionCurrency *string `help:"Transaction currency."                 name:"transaction-currency"`
	ReportLanguage      *string `help:"Report language."                      name:"report-language"`
	CustomerCompanyID   *string `help:"Customer company ID."                  name:"customer-company-id"`
}

func (f *accountingReportOptionsFlags) applyFields(builder *requestBodyBuilder, prefix string) error {
	if err := builder.SetScalarArray("receiptType", prefix+"receipt-type", f.ReceiptType, scalarInputEnum); err != nil {
		return err
	}
	if err := builder.SetScalar("receiptCurrency", prefix+"receipt-currency", f.ReceiptCurrency, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("receiptName", prefix+"receipt-name", f.ReceiptName, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("entryPeriodStart", prefix+"entry-period-start", f.EntryPeriodStart, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("entryPeriodEnd", prefix+"entry-period-end", f.EntryPeriodEnd, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("transactionValue", prefix+"transaction-value", f.TransactionValue, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("transactionCurrency", prefix+"transaction-currency", f.TransactionCurrency, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("reportLanguage", prefix+"report-language", f.ReportLanguage, scalarInputString, false); err != nil {
		return err
	}
	return builder.SetScalar("customerCompanyId", prefix+"customer-company-id", f.CustomerCompanyID, scalarInputString, false)
}

func (f *accountingReportOptionsFlags) apply(builder *requestBodyBuilder, fieldName, prefix string) error {
	return builder.SetObject(fieldName, func(nested *requestBodyBuilder) error {
		return f.applyFields(nested, prefix)
	})
}

type generalLedgerReportOptionsFlags struct {
	accountingReportOptionsFlags `embed:""`
	ReceiptNumbers               *string `help:"Receipt numbers."         name:"receipt-numbers"`
	TransactionDescription       *string `help:"Transaction description." name:"transaction-description"`
}

func (f *generalLedgerReportOptionsFlags) apply(builder *requestBodyBuilder, fieldName, prefix string) error {
	return builder.SetObject(fieldName, func(nested *requestBodyBuilder) error {
		if err := f.applyFields(nested, prefix); err != nil {
			return err
		}
		if err := nested.SetScalar("receiptNumbers", prefix+"receipt-numbers", f.ReceiptNumbers, scalarInputString, false); err != nil {
			return err
		}
		return nested.SetScalar("transactionDescription", prefix+"transaction-description", f.TransactionDescription, scalarInputString, false)
	})
}

type accountingReportTypeFlag struct {
	Type *string `help:"Accounting report type: INCOME_STATEMENT, CASH_FLOW, BALANCE_SHEET." name:"type"`
}

type ReportsAccountingCmd struct {
	StartDate                *string  `help:"Report start date (YYYY-MM-DD)."                        name:"start-date"`
	EndDate                  *string  `help:"Report end date (YYYY-MM-DD)."                          name:"end-date"`
	ReceiptStatus            []string `help:"Receipt statuses. Repeat the flag for multiple values." name:"receipt-status"`
	accountingReportTypeFlag `embed:""`
	AccountingReportOptions  accountingReportOptionsFlags `embed:""                                                      prefix:"options."`
}

func (c *ReportsAccountingCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.AccountingReportRequest
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("startDate", "start-date", c.StartDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("endDate", "end-date", c.EndDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalarArray("receiptStatus", "receipt-status", c.ReceiptStatus, scalarInputEnum); err != nil {
		return err
	}
	if err := builder.SetScalar("type", "type", c.Type, scalarInputEnum, false); err != nil {
		return err
	}
	if err := c.AccountingReportOptions.apply(builder, "options", "options."); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.GetAccountingReport(ctx, &req)
	if err != nil {
		return wrapErr("get accounting report", err)
	}
	return r.writeOutput(res)
}

type ReportsGeneralLedgerCmd struct {
	ID                   int                             `arg:""                                                        help:"Ledger account code ID." name:"id"`
	StartDate            *string                         `help:"Report start date (YYYY-MM-DD)."                        name:"start-date"`
	EndDate              *string                         `help:"Report end date (YYYY-MM-DD)."                          name:"end-date"`
	ReceiptStatus        []string                        `help:"Receipt statuses. Repeat the flag for multiple values." name:"receipt-status"`
	GeneralLedgerOptions generalLedgerReportOptionsFlags `embed:""                                                      prefix:"options."`
}

func (c *ReportsGeneralLedgerCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.GeneralLedgerReportRequest
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("startDate", "start-date", c.StartDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("endDate", "end-date", c.EndDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalarArray("receiptStatus", "receipt-status", c.ReceiptStatus, scalarInputEnum); err != nil {
		return err
	}
	if err := c.GeneralLedgerOptions.apply(builder, "options", "options."); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.GetGeneralLedgerReport(ctx, &req, procountorapi.GetGeneralLedgerReportParams{ID: c.ID})
	if err != nil {
		return wrapErr("get general ledger report", err)
	}
	return r.writeOutput(res)
}

type ReportsLedgerAccountsCmd struct {
	StartDate             *string                      `help:"Report start date (YYYY-MM-DD)."                        name:"start-date"`
	EndDate               *string                      `help:"Report end date (YYYY-MM-DD)."                          name:"end-date"`
	ReceiptStatus         []string                     `help:"Receipt statuses. Repeat the flag for multiple values." name:"receipt-status"`
	LedgerAccountsOptions accountingReportOptionsFlags `embed:""                                                      prefix:"options."`
}

func (c *ReportsLedgerAccountsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.LedgerAccountsReportRequest
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("startDate", "start-date", c.StartDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("endDate", "end-date", c.EndDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalarArray("receiptStatus", "receipt-status", c.ReceiptStatus, scalarInputEnum); err != nil {
		return err
	}
	if err := c.LedgerAccountsOptions.apply(builder, "options", "options."); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.GetLedgerAccountsReport(ctx, &req)
	if err != nil {
		return wrapErr("get ledger accounts report", err)
	}
	return r.writeOutput(res)
}
