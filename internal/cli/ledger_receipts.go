package cli

import (
	"context"
	"strconv"
	"time"

	"github.com/myyra/procountor/procountorapi"
)

type LedgerReceiptsCmd struct {
	Search                      LedgerReceiptsSearchCmd                      `cmd:"" help:"Search ledger receipts."`
	Get                         LedgerReceiptsGetCmd                         `cmd:"" help:"Get one ledger receipt by ID."`
	Create                      LedgerReceiptsCreateCmd                      `cmd:"" help:"Create a new ledger receipt."`
	Update                      LedgerReceiptsUpdateCmd                      `cmd:"" help:"Update a ledger receipt."`
	Transactions                LedgerReceiptTransactionsCmd                 `cmd:"" help:"Add, update, and remove ledger receipt transactions."`
	Approve                     LedgerReceiptsApproveCmd                     `cmd:"" help:"Approve a ledger receipt."`
	Invalidate                  LedgerReceiptsInvalidateCmd                  `cmd:"" help:"Invalidate a ledger receipt."`
	Unfinished                  LedgerReceiptsUnfinishedCmd                  `cmd:"" help:"Move a ledger receipt back to unfinished."`
	UpdateTransactionDimensions LedgerReceiptsUpdateTransactionDimensionsCmd `cmd:"" help:"Update dimensions for a ledger receipt transaction."  name:"update-transaction-dimensions"`
}

type LedgerReceiptsSearchCmd struct {
	StartDate        *DateValue     `help:"Filter start date."                                  name:"start-date"`
	EndDate          *DateValue     `help:"Filter end date."                                    name:"end-date"`
	CreatedStartDate *DateTimeValue `help:"Filter created start timestamp."                     name:"created-start-date"`
	CreatedEndDate   *DateTimeValue `help:"Filter created end timestamp."                       name:"created-end-date"`
	VersionStartDate *DateTimeValue `help:"Filter version start timestamp."                     name:"version-start-date"`
	VersionEndDate   *DateTimeValue `help:"Filter version end timestamp."                       name:"version-end-date"`
	Types            []string       `help:"Receipt types. Repeat the flag for multiple values." name:"types"`
	OrderByID        *string        `enum:"ASC,DESC"                                            help:"Order by ID."                                               name:"order-by-id"`
	OrderByDate      *string        `enum:"ASC,DESC"                                            help:"Order by date."                                             name:"order-by-date"`
	OrderByCreated   *string        `enum:"ASC,DESC"                                            help:"Order by created date."                                     name:"order-by-created"`
	OrderByVersion   *string        `enum:"ASC,DESC"                                            help:"Order by version date."                                     name:"order-by-version"`
	Status           *string        `help:"Ledger receipt status."                              name:"status"`
	Paginate         PaginateValue  `default:"0:200"                                            help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *LedgerReceiptsSearchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := buildSearchLedgerReceiptParams(c)
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.LedgerReceiptBasicInfo, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.SearchLedgerReceipt(ctx, params)
		if err != nil {
			return nil, wrapErr("search ledger receipts", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

func buildSearchLedgerReceiptParams(c *LedgerReceiptsSearchCmd) procountorapi.SearchLedgerReceiptParams {
	var params procountorapi.SearchLedgerReceiptParams
	if c.StartDate != nil {
		params.StartDate.SetTo(c.StartDate.Time)
	}
	if c.EndDate != nil {
		params.EndDate.SetTo(c.EndDate.Time)
	}
	if c.CreatedStartDate != nil {
		params.CreatedStartDate.SetTo(c.CreatedStartDate.Time)
	}
	if c.CreatedEndDate != nil {
		params.CreatedEndDate.SetTo(c.CreatedEndDate.Time)
	}
	if c.VersionStartDate != nil {
		params.VersionStartDate.SetTo(c.VersionStartDate.Time)
	}
	if c.VersionEndDate != nil {
		params.VersionEndDate.SetTo(c.VersionEndDate.Time)
	}
	if len(c.Types) > 0 {
		params.Types = make([]procountorapi.SearchLedgerReceiptTypesItem, 0, len(c.Types))
		for _, value := range c.Types {
			params.Types = append(params.Types, procountorapi.SearchLedgerReceiptTypesItem(value))
		}
	}
	if c.OrderByID != nil {
		params.OrderById.SetTo(procountorapi.SearchLedgerReceiptOrderById(*c.OrderByID))
	}
	if c.OrderByDate != nil {
		params.OrderByDate.SetTo(procountorapi.SearchLedgerReceiptOrderByDate(*c.OrderByDate))
	}
	if c.OrderByCreated != nil {
		params.OrderByCreated.SetTo(procountorapi.SearchLedgerReceiptOrderByCreated(*c.OrderByCreated))
	}
	if c.OrderByVersion != nil {
		params.OrderByVersion.SetTo(procountorapi.SearchLedgerReceiptOrderByVersion(*c.OrderByVersion))
	}
	if c.Status != nil {
		params.Status.SetTo(*c.Status)
	}
	return params
}

type LedgerReceiptsGetCmd struct {
	ReceiptID string `arg:"" help:"Ledger receipt ID." name:"receipt-id"`
}

func (c *LedgerReceiptsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetLedgerReceipt(ctx, procountorapi.GetLedgerReceiptParams{ReceiptId: c.ReceiptID})
	if err != nil {
		return wrapErr("get ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsCreateCmd struct {
	CreateReconciliation *bool      `help:"Ask Procountor to create a reconciliation transaction when saving the receipt." name:"create-reconciliation"`
	LedgerReceipt        *JSONInput `help:"Full ledger receipt JSON document. Accepts inline JSON, @file, or - for stdin." name:"ledger-receipt"        required:""`
}

func (c *LedgerReceiptsCreateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.LedgerReceipt
	if err := decodeRequiredImportInput(newRequestInputReader(r), "ledger-receipt", c.LedgerReceipt, req.Decode); err != nil {
		return err
	}
	var params procountorapi.SaveLedgerReceiptParams
	if c.CreateReconciliation != nil {
		params.CreateReconciliation.SetTo(*c.CreateReconciliation)
	}
	res, err := r.client.SaveLedgerReceipt(ctx, &req, params)
	if err != nil {
		return wrapErr("create ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsUpdateCmd struct {
	ReceiptID            int        `arg:""                                                                                help:"Ledger receipt ID."    name:"receipt-id"`
	CreateReconciliation *bool      `help:"Ask Procountor to create a reconciliation transaction when saving the receipt." name:"create-reconciliation"`
	LedgerReceipt        *JSONInput `help:"Full ledger receipt JSON document. Accepts inline JSON, @file, or - for stdin." name:"ledger-receipt"        required:""`
}

func (c *LedgerReceiptsUpdateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.LedgerReceipt
	if err := decodeRequiredImportInput(newRequestInputReader(r), "ledger-receipt", c.LedgerReceipt, req.Decode); err != nil {
		return err
	}
	params := procountorapi.UpdateLedgerReceiptParams{ReceiptId: c.ReceiptID}
	if c.CreateReconciliation != nil {
		params.CreateReconciliation.SetTo(*c.CreateReconciliation)
	}
	res, err := r.client.UpdateLedgerReceipt(ctx, &req, params)
	if err != nil {
		return wrapErr("update ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptTransactionsCmd struct {
	Add    LedgerReceiptTransactionsAddCmd    `cmd:"" help:"Add a transaction to a ledger receipt."`
	Update LedgerReceiptTransactionsUpdateCmd `cmd:"" help:"Update one ledger receipt transaction."`
	Remove LedgerReceiptTransactionsRemoveCmd `cmd:"" help:"Remove one transaction from a ledger receipt."`
}

type ledgerReceiptMutationFlags struct {
	ReceiptID            int   `arg:""                                                                                help:"Ledger receipt ID."    name:"receipt-id"`
	CreateReconciliation *bool `help:"Ask Procountor to create a reconciliation transaction when saving the receipt." name:"create-reconciliation"`
}

type ledgerReceiptTransactionTargetFlags struct {
	ledgerReceiptMutationFlags `embed:""`
	TransactionID              int `arg:""   help:"Transaction ID within the ledger receipt." name:"transaction-id"`
}

type ledgerReceiptTransactionInputFlags struct {
	TransactionType     *string `help:"Transaction type such as ENTRY or RECONCILIATION_ENTRY." name:"transaction-type"`
	Account             *string `help:"Ledger account number for the transaction."              name:"account"`
	AccountingValue     *string `help:"Accounting value for the transaction."                   name:"accounting-value"`
	VatPercent          *string `help:"VAT percentage for the transaction."                     name:"vat-percent"`
	VatType             *string `help:"VAT type for the transaction."                           name:"vat-type"`
	VatStatus           *string `help:"VAT status code for the transaction."                    name:"vat-status"`
	Description         *string `help:"Transaction description."                                name:"description"`
	BalanceCode         *string `help:"Balance sheet code for the transaction."                 name:"balance-code"`
	VatDeductionPercent *string `help:"VAT deduction percentage for the transaction."           name:"vat-deduction-percent"`
	StartDate           *string `help:"Start date for the transaction period (YYYY-MM-DD)."     name:"start-date"`
	EndDate             *string `help:"End date for the transaction period (YYYY-MM-DD)."       name:"end-date"`
}

func (f *ledgerReceiptTransactionInputFlags) buildRequired() (procountorapi.Transaction, error) {
	if f.TransactionType == nil {
		return procountorapi.Transaction{}, invalidUsage("missing required flag --transaction-type")
	}
	if f.Account == nil {
		return procountorapi.Transaction{}, invalidUsage("missing required flag --account")
	}
	if f.AccountingValue == nil {
		return procountorapi.Transaction{}, invalidUsage("missing required flag --accounting-value")
	}
	if f.VatPercent == nil {
		return procountorapi.Transaction{}, invalidUsage("missing required flag --vat-percent")
	}
	transaction := procountorapi.Transaction{
		TransactionType: procountorapi.TransactionTransactionType(*f.TransactionType),
		Account:         *f.Account,
	}
	value, _, err := readScalarInput(f.AccountingValue, scalarInputFloat, false)
	if err != nil {
		return procountorapi.Transaction{}, invalidUsage("invalid --accounting-value: %v", err)
	}
	transaction.AccountingValue = value.(float64)
	value, _, err = readScalarInput(f.VatPercent, scalarInputFloat, false)
	if err != nil {
		return procountorapi.Transaction{}, invalidUsage("invalid --vat-percent: %v", err)
	}
	transaction.VatPercent = value.(float64)
	if _, err := f.apply(&transaction); err != nil {
		return procountorapi.Transaction{}, err
	}
	return transaction, nil
}

func (f *ledgerReceiptTransactionInputFlags) apply(transaction *procountorapi.Transaction) (bool, error) {
	changed := false
	if f.TransactionType != nil {
		transaction.TransactionType = procountorapi.TransactionTransactionType(*f.TransactionType)
		changed = true
	}
	if f.Account != nil {
		transaction.Account = *f.Account
		changed = true
	}
	if f.AccountingValue != nil {
		value, _, err := readScalarInput(f.AccountingValue, scalarInputFloat, false)
		if err != nil {
			return false, invalidUsage("invalid --accounting-value: %v", err)
		}
		transaction.AccountingValue = value.(float64)
		changed = true
	}
	if f.VatPercent != nil {
		value, _, err := readScalarInput(f.VatPercent, scalarInputFloat, false)
		if err != nil {
			return false, invalidUsage("invalid --vat-percent: %v", err)
		}
		transaction.VatPercent = value.(float64)
		changed = true
	}
	if f.VatType != nil {
		transaction.VatType.SetTo(procountorapi.TransactionVatType(*f.VatType))
		changed = true
	}
	if f.VatStatus != nil {
		value, _, err := readScalarInput(f.VatStatus, scalarInputInt, false)
		if err != nil {
			return false, invalidUsage("invalid --vat-status: %v", err)
		}
		transaction.VatStatus.SetTo(value.(int))
		changed = true
	}
	if f.Description != nil {
		transaction.Description.SetTo(*f.Description)
		changed = true
	}
	if f.BalanceCode != nil {
		transaction.BalanceCode.SetTo(*f.BalanceCode)
		changed = true
	}
	if f.VatDeductionPercent != nil {
		value, _, err := readScalarInput(f.VatDeductionPercent, scalarInputFloat, false)
		if err != nil {
			return false, invalidUsage("invalid --vat-deduction-percent: %v", err)
		}
		transaction.VatDeductionPercent.SetTo(value.(float64))
		changed = true
	}
	if f.StartDate != nil {
		value, _, err := readScalarInput(f.StartDate, scalarInputDate, false)
		if err != nil {
			return false, invalidUsage("invalid --start-date: %v", err)
		}
		transaction.StartDate.SetTo(value.(time.Time))
		changed = true
	}
	if f.EndDate != nil {
		value, _, err := readScalarInput(f.EndDate, scalarInputDate, false)
		if err != nil {
			return false, invalidUsage("invalid --end-date: %v", err)
		}
		transaction.EndDate.SetTo(value.(time.Time))
		changed = true
	}
	return changed, nil
}

type LedgerReceiptTransactionsAddCmd struct {
	ledgerReceiptMutationFlags         `embed:""`
	ledgerReceiptTransactionInputFlags `embed:""`
}

func (c *LedgerReceiptTransactionsAddCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	receipt, err := r.getLedgerReceiptForMutation(ctx, c.ReceiptID)
	if err != nil {
		return err
	}
	transaction, err := c.buildRequired()
	if err != nil {
		return err
	}
	receipt.Transactions = append(receipt.Transactions, transaction)
	return r.updateLedgerReceiptFromMutation(ctx, c.ReceiptID, c.CreateReconciliation, receipt)
}

type LedgerReceiptTransactionsUpdateCmd struct {
	ledgerReceiptTransactionTargetFlags `embed:""`
	ledgerReceiptTransactionInputFlags  `embed:""`
}

func (c *LedgerReceiptTransactionsUpdateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	receipt, err := r.getLedgerReceiptForMutation(ctx, c.ReceiptID)
	if err != nil {
		return err
	}
	index := ledgerReceiptTransactionIndex(receipt.Transactions, c.TransactionID)
	if index < 0 {
		return invalidUsage("transaction %d not found on ledger receipt %d", c.TransactionID, c.ReceiptID)
	}
	changed, err := c.apply(&receipt.Transactions[index])
	if err != nil {
		return err
	}
	if !changed {
		return invalidUsage("provide at least one transaction field to update")
	}
	return r.updateLedgerReceiptFromMutation(ctx, c.ReceiptID, c.CreateReconciliation, receipt)
}

type LedgerReceiptTransactionsRemoveCmd struct {
	ledgerReceiptTransactionTargetFlags `embed:""`
}

func (c *LedgerReceiptTransactionsRemoveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	receipt, err := r.getLedgerReceiptForMutation(ctx, c.ReceiptID)
	if err != nil {
		return err
	}
	index := ledgerReceiptTransactionIndex(receipt.Transactions, c.TransactionID)
	if index < 0 {
		return invalidUsage("transaction %d not found on ledger receipt %d", c.TransactionID, c.ReceiptID)
	}
	if len(receipt.Transactions) == 1 {
		return invalidUsage("ledger receipt must contain at least one transaction")
	}
	receipt.Transactions = append(receipt.Transactions[:index], receipt.Transactions[index+1:]...)
	return r.updateLedgerReceiptFromMutation(ctx, c.ReceiptID, c.CreateReconciliation, receipt)
}

func ledgerReceiptTransactionIndex(transactions []procountorapi.Transaction, transactionID int) int {
	for idx, transaction := range transactions {
		if id, ok := transaction.ID.Get(); ok && id == transactionID {
			return idx
		}
	}
	return -1
}

func (r *runner) getLedgerReceiptForMutation(ctx context.Context, receiptID int) (*procountorapi.LedgerReceipt, error) {
	receipt, err := r.client.GetLedgerReceipt(ctx, procountorapi.GetLedgerReceiptParams{ReceiptId: strconv.Itoa(receiptID)})
	if err != nil {
		return nil, wrapErr("get ledger receipt", err)
	}
	return receipt, nil
}

func (r *runner) updateLedgerReceiptFromMutation(ctx context.Context, receiptID int, createReconciliation *bool, receipt *procountorapi.LedgerReceipt) error {
	params := procountorapi.UpdateLedgerReceiptParams{ReceiptId: receiptID}
	if createReconciliation != nil {
		params.CreateReconciliation.SetTo(*createReconciliation)
	}
	res, err := r.client.UpdateLedgerReceipt(ctx, receipt, params)
	if err != nil {
		return wrapErr("update ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsApproveCmd struct {
	ReceiptID int `arg:"" help:"Ledger receipt ID." name:"receipt-id"`
}

func (c *LedgerReceiptsApproveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.ApproveLedgerReceipt(ctx, procountorapi.ApproveLedgerReceiptParams{ReceiptId: c.ReceiptID})
	if err != nil {
		return wrapErr("approve ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsInvalidateCmd struct {
	ReceiptID int `arg:"" help:"Ledger receipt ID." name:"receipt-id"`
}

func (c *LedgerReceiptsInvalidateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.MarkLedgerReceiptAsInvalidated(ctx, procountorapi.MarkLedgerReceiptAsInvalidatedParams{ReceiptId: c.ReceiptID})
	if err != nil {
		return wrapErr("invalidate ledger receipt", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsUnfinishedCmd struct {
	ReceiptID int `arg:"" help:"Ledger receipt ID." name:"receipt-id"`
}

func (c *LedgerReceiptsUnfinishedCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.MarkLedgerReceiptAsUnfinished(ctx, procountorapi.MarkLedgerReceiptAsUnfinishedParams{ReceiptId: c.ReceiptID})
	if err != nil {
		return wrapErr("set ledger receipt unfinished", err)
	}
	return r.writeOutput(res)
}

type LedgerReceiptsUpdateTransactionDimensionsCmd struct {
	ReceiptID              int        `arg:""                                                                                           help:"Ledger receipt ID."        name:"receipt-id"`
	TransactionID          int        `arg:""                                                                                           help:"Transaction ID."           name:"transaction-id"`
	DimensionItemValueList *JSONInput `help:"Full dimension item value list JSON document. Accepts inline JSON, @file, or - for stdin." name:"dimension-item-value-list" required:""`
}

func (c *LedgerReceiptsUpdateTransactionDimensionsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.DimensionItemValueList
	if err := decodeRequiredImportInput(newRequestInputReader(r), "dimension-item-value-list", c.DimensionItemValueList, req.Decode); err != nil {
		return err
	}
	res, err := r.client.UpdateLedgerReceiptTransactionDimensions(ctx, &req, procountorapi.UpdateLedgerReceiptTransactionDimensionsParams{ReceiptId: c.ReceiptID, TransactionId: c.TransactionID})
	if err != nil {
		return wrapErr("update transaction dimensions", err)
	}
	return r.writeOutput(res)
}
