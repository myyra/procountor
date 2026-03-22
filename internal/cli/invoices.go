package cli

import (
	"context"
	"errors"
	"io"

	"github.com/myyra/procountor/procountorapi"
)

type InvoicesCmd struct {
	Search                InvoicesSearchCmd                `cmd:"" help:"Search invoices."`
	Get                   InvoicesGetCmd                   `cmd:"" help:"Get one invoice by ID."`
	Save                  InvoicesSaveCmd                  `cmd:"" help:"Create a new invoice."`
	AddNotes              InvoicesAddNotesCmd              `cmd:"" help:"Add or update invoice notes."                              name:"add-notes"`
	Approve               InvoicesApproveCmd               `cmd:"" help:"Approve an invoice."`
	Verify                InvoicesVerifyCmd                `cmd:"" help:"Verify an invoice."`
	Reject                InvoicesRejectCmd                `cmd:"" help:"Set invoice to rejected or payment denied state."`
	Invalidate            InvoicesInvalidateCmd            `cmd:"" help:"Invalidate an invoice."`
	Unfinished            InvoicesUnfinishedCmd            `cmd:"" help:"Move an invoice back to unfinished."`
	Send                  InvoicesSendCmd                  `cmd:"" help:"Send a sales invoice to the customer."`
	SendToCirculation     InvoicesSendToCirculationCmd     `cmd:"" help:"Send an invoice to verification and approval circulation." name:"send-to-circulation"`
	Transactions          InvoicesTransactionsCmd          `cmd:"" help:"List invoice transactions."`
	Image                 InvoicesImageCmd                 `cmd:"" help:"Download invoice image."`
	PaymentEvents         InvoicePaymentEventsCmd          `cmd:"" help:"List and mutate invoice payment events."                   name:"payment-events"`
	Comments              InvoiceCommentsCmd               `cmd:"" help:"List and mutate invoice discussion comments."`
	PersonalApprovals     InvoicesPersonalApprovalsCmd     `cmd:"" help:"List invoices waiting for your approval."                  name:"personal-approvals"`
	PersonalVerifications InvoicesPersonalVerificationsCmd `cmd:"" help:"List invoices waiting for your verification."              name:"personal-verifications"`
}

type InvoicesSearchCmd struct {
	Status                []string       `help:"Invoice statuses. Repeat the flag for multiple values." name:"status"`
	StartDate             *DateValue     `help:"Billing date start."                                    name:"start-date"`
	EndDate               *DateValue     `help:"Billing date end."                                      name:"end-date"`
	CreatedStartDate      *DateTimeValue `help:"Created start timestamp."                               name:"created-start-date"`
	CreatedEndDate        *DateTimeValue `help:"Created end timestamp."                                 name:"created-end-date"`
	VersionStartDate      *DateTimeValue `help:"Version start timestamp."                               name:"version-start-date"`
	VersionEndDate        *DateTimeValue `help:"Version end timestamp."                                 name:"version-end-date"`
	DueStartDate          *DateValue     `help:"Due date start."                                        name:"due-start-date"`
	DueEndDate            *DateValue     `help:"Due date end."                                          name:"due-end-date"`
	Types                 []string       `help:"Invoice types. Repeat the flag for multiple values."    name:"types"`
	BusinessPartnerID     *int           `help:"Filter by business partner ID."                         name:"business-partner-id"`
	FactoringContractID   *int           `help:"Filter by factoring contract ID."                       name:"factoring-contract-id"`
	OriginalInvoiceNumber *string        `help:"Filter by original invoice number."                     name:"original-invoice-number"`
	InvoiceNumber         *int           `help:"Filter by invoice number."                              name:"invoice-number"`
	SearchTerm            *string        `help:"Search term."                                           name:"search-term"`
	OrderByID             *string        `enum:"ASC,DESC"                                               help:"Order by ID."                                               name:"order-by-id"`
	OrderByDate           *string        `enum:"ASC,DESC"                                               help:"Order by date."                                             name:"order-by-date"`
	OrderByCreated        *string        `enum:"ASC,DESC"                                               help:"Order by created date."                                     name:"order-by-created"`
	OrderByVersion        *string        `enum:"ASC,DESC"                                               help:"Order by version date."                                     name:"order-by-version"`
	InvoiceChannel        []string       `help:"Invoice channels. Repeat the flag for multiple values." name:"invoice-channel"`
	Paginate              PaginateValue  `default:"0:200"                                               help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *InvoicesSearchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := buildSearchInvoiceParams(c)
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.InvoiceBasicInfo, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.SearchInvoice(ctx, params)
		if err != nil {
			return nil, wrapErr("search invoices", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

func buildSearchInvoiceParams(c *InvoicesSearchCmd) procountorapi.SearchInvoiceParams {
	var params procountorapi.SearchInvoiceParams
	if len(c.Status) > 0 {
		params.Status = make([]procountorapi.SearchInvoiceStatusItem, 0, len(c.Status))
		for _, value := range c.Status {
			params.Status = append(params.Status, procountorapi.SearchInvoiceStatusItem(value))
		}
	}
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
	if c.DueStartDate != nil {
		params.DueStartDate.SetTo(c.DueStartDate.Time)
	}
	if c.DueEndDate != nil {
		params.DueEndDate.SetTo(c.DueEndDate.Time)
	}
	if len(c.Types) > 0 {
		params.Types = make([]procountorapi.SearchInvoiceTypesItem, 0, len(c.Types))
		for _, value := range c.Types {
			params.Types = append(params.Types, procountorapi.SearchInvoiceTypesItem(value))
		}
	}
	if c.BusinessPartnerID != nil {
		params.BusinessPartnerId.SetTo(*c.BusinessPartnerID)
	}
	if c.FactoringContractID != nil {
		params.FactoringContractId.SetTo(*c.FactoringContractID)
	}
	if c.OriginalInvoiceNumber != nil {
		params.OriginalInvoiceNumber.SetTo(*c.OriginalInvoiceNumber)
	}
	if c.InvoiceNumber != nil {
		params.InvoiceNumber.SetTo(*c.InvoiceNumber)
	}
	if c.SearchTerm != nil {
		params.SearchTerm.SetTo(*c.SearchTerm)
	}
	if c.OrderByID != nil {
		params.OrderById.SetTo(procountorapi.SearchInvoiceOrderById(*c.OrderByID))
	}
	if c.OrderByDate != nil {
		params.OrderByDate.SetTo(procountorapi.SearchInvoiceOrderByDate(*c.OrderByDate))
	}
	if c.OrderByCreated != nil {
		params.OrderByCreated.SetTo(procountorapi.SearchInvoiceOrderByCreated(*c.OrderByCreated))
	}
	if c.OrderByVersion != nil {
		params.OrderByVersion.SetTo(procountorapi.SearchInvoiceOrderByVersion(*c.OrderByVersion))
	}
	if len(c.InvoiceChannel) > 0 {
		params.InvoiceChannel = make([]procountorapi.SearchInvoiceInvoiceChannelItem, 0, len(c.InvoiceChannel))
		for _, value := range c.InvoiceChannel {
			params.InvoiceChannel = append(params.InvoiceChannel, procountorapi.SearchInvoiceInvoiceChannelItem(value))
		}
	}
	return params
}

type InvoicesGetCmd struct {
	InvoiceID string `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetInvoice(ctx, procountorapi.GetInvoiceParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("get invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesSaveCmd struct {
	AddCollectionPenalCosts *bool      `help:"Automatically add collection or penal costs."                            name:"add-collection-penal-costs"`
	Invoice                 *JSONInput `help:"Full invoice JSON document. Accepts inline JSON, @file, or - for stdin." name:"invoice"                    required:""`
}

func (c *InvoicesSaveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.Invoice
	if err := decodeRequiredImportInput(newRequestInputReader(r), "invoice", c.Invoice, req.Decode); err != nil {
		return err
	}
	var params procountorapi.SaveInvoiceParams
	if c.AddCollectionPenalCosts != nil {
		params.AddCollectionPenalCosts.SetTo(*c.AddCollectionPenalCosts)
	}
	res, err := r.client.SaveInvoice(ctx, &req, params)
	if err != nil {
		return wrapErr("save invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesAddNotesCmd struct {
	InvoiceID int     `arg:""                help:"Invoice ID." name:"invoice-id"`
	Notes     *string `help:"Invoice notes." name:"notes"`
}

func (c *InvoicesAddNotesCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.InvoiceNotes
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("notes", "notes", c.Notes, scalarInputString, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.AddNotesToInvoice(ctx, &req, procountorapi.AddNotesToInvoiceParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("add notes to invoice", err)
	}
	return r.writeOutput(res)
}

type checkingEventInputFlags struct {
	Comment *string `help:"Checking event comment." name:"comment"`
}

func (f *checkingEventInputFlags) parse() procountorapi.OptCheckingEvent {
	var req procountorapi.OptCheckingEvent
	if f.Comment == nil {
		return req
	}
	var body procountorapi.CheckingEvent
	body.Comment.SetTo(*f.Comment)
	req.SetTo(body)
	return req
}

type InvoicesApproveCmd struct {
	InvoiceID               int   `arg:""                   help:"Invoice ID."      name:"invoice-id"`
	UpdateInventory         *bool `help:"Update inventory." name:"update-inventory"`
	checkingEventInputFlags `embed:""`
}

func (c *InvoicesApproveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := procountorapi.ApproveInvoiceParams{InvoiceId: c.InvoiceID}
	if c.UpdateInventory != nil {
		params.UpdateInventory.SetTo(*c.UpdateInventory)
	}
	req := c.parse()
	res, err := r.client.ApproveInvoice(ctx, req, params)
	if err != nil {
		return wrapErr("approve invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesVerifyCmd struct {
	InvoiceID               int `arg:""   help:"Invoice ID." name:"invoice-id"`
	checkingEventInputFlags `embed:""`
}

func (c *InvoicesVerifyCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	req := c.parse()
	res, err := r.client.VerifyInvoice(ctx, req, procountorapi.VerifyInvoiceParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("verify invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesRejectCmd struct {
	InvoiceID               int `arg:""   help:"Invoice ID." name:"invoice-id"`
	checkingEventInputFlags `embed:""`
}

func (c *InvoicesRejectCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	req := c.parse()
	res, err := r.client.RejectInvoice(ctx, req, procountorapi.RejectInvoiceParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("reject invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesInvalidateCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesInvalidateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.MarkInvoiceAsInvalidated(ctx, procountorapi.MarkInvoiceAsInvalidatedParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("invalidate invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesUnfinishedCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesUnfinishedCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.MarkInvoiceAsUnfinished(ctx, procountorapi.MarkInvoiceAsUnfinishedParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("set invoice unfinished", err)
	}
	return r.writeOutput(res)
}

type InvoicesSendCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesSendCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.Send(ctx, procountorapi.SendParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("send invoice", err)
	}
	return r.writeOutput(res)
}

type InvoicesSendToCirculationCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesSendToCirculationCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.SendToCirculation(ctx, procountorapi.SendToCirculationParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("send invoice to circulation", err)
	}
	return r.writeOutput(res)
}

type InvoicesTransactionsCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoicesTransactionsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetInvoiceTransactions(ctx, procountorapi.GetInvoiceTransactionsParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("get invoice transactions", err)
	}
	return r.writeOutput(res)
}

type InvoicesImageCmd struct {
	InvoiceID int     `arg:""                                                  help:"Invoice ID."   name:"invoice-id"`
	Page      *int    `help:"Invoice page number."                             name:"page"`
	Format    *string `enum:"PNG,JPG,JPEG"                                     help:"Image format." name:"format"`
	Out       string  `help:"Write binary response to file instead of stdout." name:"out"`
}

func (c *InvoicesImageCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := procountorapi.GetInvoiceImageParams{InvoiceId: c.InvoiceID}
	if c.Page != nil {
		params.Page.SetTo(*c.Page)
	}
	if c.Format != nil {
		params.Format.SetTo(procountorapi.GetInvoiceImageFormat(*c.Format))
	}
	res, err := r.client.GetInvoiceImage(ctx, params)
	if err != nil {
		return wrapErr("get invoice image", err)
	}
	reader, ok := res.(io.Reader)
	if !ok {
		return errors.New("invoice image response is not binary")
	}
	return r.writeReaderOutput(reader, c.Out)
}

type InvoicePaymentEventsCmd struct {
	Search   InvoicePaymentEventsSearchCmd   `cmd:"" help:"List payment events for an invoice."`
	Get      InvoicePaymentEventsGetCmd      `cmd:"" help:"Get one payment event by ID."`
	MarkPaid InvoicePaymentEventsMarkPaidCmd `cmd:"" help:"Mark invoice paid elsewhere."              name:"mark-paid"`
	Remove   InvoicePaymentEventsRemoveCmd   `cmd:"" help:"Remove one payment event from an invoice."`
}

type InvoicePaymentEventsSearchCmd struct {
	InvoiceID int           `arg:""          help:"Invoice ID."                                                name:"invoice-id"`
	OrderByID *string       `enum:"ASC,DESC" help:"Order by ID."                                               name:"order-by-id"`
	Paginate  PaginateValue `default:"0:200" help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *InvoicePaymentEventsSearchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := procountorapi.SearchPaymentEventsParams{InvoiceId: c.InvoiceID}
	if c.OrderByID != nil {
		params.OrderById.SetTo(procountorapi.SearchPaymentEventsOrderById(*c.OrderByID))
	}
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.PaymentEvent, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.SearchPaymentEvents(ctx, params)
		if err != nil {
			return nil, wrapErr("search payment events", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

type InvoicePaymentEventsGetCmd struct {
	InvoiceID      int `arg:"" help:"Invoice ID."       name:"invoice-id"`
	PaymentEventID int `arg:"" help:"Payment event ID." name:"payment-event-id"`
}

func (c *InvoicePaymentEventsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetPaymentEventByIdAndInvoiceId(ctx, procountorapi.GetPaymentEventByIdAndInvoiceIdParams{InvoiceId: c.InvoiceID, PaymentEventId: c.PaymentEventID})
	if err != nil {
		return wrapErr("get payment event", err)
	}
	return r.writeOutput(res)
}

type InvoicePaymentEventsMarkPaidCmd struct {
	InvoiceID         int     `arg:""                                  help:"Invoice ID."         name:"invoice-id"`
	AddPenalExpense   *bool   `help:"Automatically add penal expense." name:"add-penal-expense"`
	PaymentDate       *string `help:"Payment date (YYYY-MM-DD)."       name:"payment-date"`
	Amount            *string `help:"Payment amount."                  name:"amount"`
	Currency          *string `help:"Payment currency."                name:"currency"`
	Description       *string `help:"Payment description."             name:"description"`
	PaymentMethodType *string `help:"Payment method type."             name:"payment-method-type"`
}

func (c *InvoicePaymentEventsMarkPaidCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.MarkInvoiceAsPaid
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("paymentDate", "payment-date", c.PaymentDate, scalarInputDate, false); err != nil {
		return err
	}
	if err := builder.SetScalar("amount", "amount", c.Amount, scalarInputFloat, false); err != nil {
		return err
	}
	if err := builder.SetScalar("currency", "currency", c.Currency, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("description", "description", c.Description, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("paymentMethodType", "payment-method-type", c.PaymentMethodType, scalarInputEnum, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	params := procountorapi.MarkInvoiceAsPaidParams{InvoiceId: c.InvoiceID}
	if c.AddPenalExpense != nil {
		params.AddPenalExpense.SetTo(*c.AddPenalExpense)
	}
	res, err := r.client.MarkInvoiceAsPaid(ctx, &req, params)
	if err != nil {
		return wrapErr("mark invoice as paid", err)
	}
	return r.writeOutput(res)
}

type InvoicePaymentEventsRemoveCmd struct {
	InvoiceID      int `arg:"" help:"Invoice ID."       name:"invoice-id"`
	PaymentEventID int `arg:"" help:"Payment event ID." name:"payment-event-id"`
}

func (c *InvoicePaymentEventsRemoveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.RemovePaymentEvent(ctx, procountorapi.RemovePaymentEventParams{InvoiceId: c.InvoiceID, PaymentEventId: c.PaymentEventID})
	if err != nil {
		return wrapErr("remove payment event", err)
	}
	return r.writeOutput(res)
}

type InvoiceCommentsCmd struct {
	List          InvoiceCommentsListCmd          `cmd:"" help:"List invoice discussion comments."`
	Get           InvoiceCommentsGetCmd           `cmd:"" help:"Get one invoice comment."`
	Create        InvoiceCommentsCreateCmd        `cmd:"" help:"Post a new invoice discussion comment."`
	SetRead       InvoiceCommentsSetReadCmd       `cmd:"" help:"Mark an invoice comment as read."              name:"set-read"`
	TaggableUsers InvoiceCommentsTaggableUsersCmd `cmd:"" help:"List users you can tag in invoice discussion." name:"taggable-users"`
}

type InvoiceCommentsListCmd struct {
	InvoiceID            int   `arg:""                              help:"Invoice ID."              name:"invoice-id"`
	ReadByAllTaggedUsers *bool `help:"Filter by tagged-read state." name:"read-by-all-tagged-users"`
}

func (c *InvoiceCommentsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := procountorapi.GetInvoiceCommentsParams{InvoiceId: c.InvoiceID}
	if c.ReadByAllTaggedUsers != nil {
		params.ReadByAllTaggedUsers.SetTo(*c.ReadByAllTaggedUsers)
	}
	res, err := r.client.GetInvoiceComments(ctx, params)
	if err != nil {
		return wrapErr("get invoice comments", err)
	}
	return r.writeOutput(res)
}

type InvoiceCommentsGetCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
	CommentID int `arg:"" help:"Comment ID." name:"comment-id"`
}

func (c *InvoiceCommentsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetCommentById(ctx, procountorapi.GetCommentByIdParams{InvoiceId: c.InvoiceID, CommentId: c.CommentID})
	if err != nil {
		return wrapErr("get comment", err)
	}
	return r.writeOutput(res)
}

type taggedUsersInfoFlags struct {
	NumTaggedUsers       *string `help:"Number of tagged users."                  name:"num-tagged-users"`
	ReadByAllTaggedUsers *string `help:"All tagged users have read (true/false)." name:"read-by-all-tagged-users"`
}

func (f *taggedUsersInfoFlags) apply(builder *requestBodyBuilder, fieldName, prefix string) error {
	return builder.SetObject(fieldName, func(nested *requestBodyBuilder) error {
		if err := nested.SetScalar("numTaggedUsers", prefix+"num-tagged-users", f.NumTaggedUsers, scalarInputInt, false); err != nil {
			return err
		}
		return nested.SetScalar("readByAllTaggedUsers", prefix+"read-by-all-tagged-users", f.ReadByAllTaggedUsers, scalarInputBool, false)
	})
}

type InvoiceCommentsCreateCmd struct {
	InvoiceID       int                  `arg:""                              help:"Invoice ID."          name:"invoice-id"`
	CommentID       *string              `help:"Comment ID."                  name:"comment-id"`
	Author          *string              `help:"Comment author."              name:"author"`
	DateTime        *string              `help:"Comment timestamp (RFC3339)." name:"date-time"`
	Comment         *string              `help:"Comment text."                name:"comment"`
	TaggedUsersInfo taggedUsersInfoFlags `embed:""                            prefix:"tagged-users-info."`
}

func (c *InvoiceCommentsCreateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.Comment
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("id", "comment-id", c.CommentID, scalarInputInt, false); err != nil {
		return err
	}
	if err := builder.SetScalar("author", "author", c.Author, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("dateTime", "date-time", c.DateTime, scalarInputDateTime, false); err != nil {
		return err
	}
	if err := builder.SetScalar("comment", "comment", c.Comment, scalarInputString, false); err != nil {
		return err
	}
	if err := c.TaggedUsersInfo.apply(builder, "taggedUsersInfo", "tagged-users-info."); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.SaveInvoiceComment(ctx, &req, procountorapi.SaveInvoiceCommentParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("save invoice comment", err)
	}
	return r.writeOutput(res)
}

type InvoiceCommentsSetReadCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
	CommentID int `arg:"" help:"Comment ID." name:"comment-id"`
}

func (c *InvoiceCommentsSetReadCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.SetCommentReadByUser(ctx, procountorapi.SetCommentReadByUserParams{InvoiceId: c.InvoiceID, CommentId: c.CommentID})
	if err != nil {
		return wrapErr("set comment read", err)
	}
	return r.writeOutput(res)
}

type InvoiceCommentsTaggableUsersCmd struct {
	InvoiceID int `arg:"" help:"Invoice ID." name:"invoice-id"`
}

func (c *InvoiceCommentsTaggableUsersCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetTaggeableUsers(ctx, procountorapi.GetTaggeableUsersParams{InvoiceId: c.InvoiceID})
	if err != nil {
		return wrapErr("get taggable users", err)
	}
	return r.writeOutput(res)
}

type InvoicesPersonalApprovalsCmd struct {
	Types string `help:"Invoice type." name:"types" required:""`
}

func (c *InvoicesPersonalApprovalsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetInvoicesForPersonalApprovals(ctx, procountorapi.GetInvoicesForPersonalApprovalsParams{Types: procountorapi.GetInvoicesForPersonalApprovalsTypes(c.Types)})
	if err != nil {
		return wrapErr("get invoices for personal approvals", err)
	}
	return r.writeOutput(res)
}

type InvoicesPersonalVerificationsCmd struct {
	Types string `help:"Invoice type." name:"types" required:""`
}

func (c *InvoicesPersonalVerificationsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetInvoicesForPersonalVerifications(ctx, procountorapi.GetInvoicesForPersonalVerificationsParams{Types: procountorapi.GetInvoicesForPersonalVerificationsTypes(c.Types)})
	if err != nil {
		return wrapErr("get invoices for personal verifications", err)
	}
	return r.writeOutput(res)
}
