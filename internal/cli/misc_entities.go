package cli

import (
	"context"

	"github.com/myyra/procountor/procountorapi"
)

type ChartOfAccountsCmd struct{}

func (c *ChartOfAccountsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetChartOfAccount(ctx)
	if err != nil {
		return wrapErr("get chart of accounts", err)
	}
	return r.writeOutput(res)
}

type CompanyCmd struct{}

func (c *CompanyCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetCompanyInfo(ctx)
	if err != nil {
		return wrapErr("get company info", err)
	}
	return r.writeOutput(res)
}

type CostCentersCmd struct {
	Get CostCentersGetCmd `cmd:"" help:"Get cost center info for one ledger receipt."`
}

type CostCentersGetCmd struct {
	LedgerReceiptID int `arg:"" help:"Ledger receipt ID." name:"ledger-receipt-id"`
}

func (c *CostCentersGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetCostCenter(ctx, procountorapi.GetCostCenterParams{LedgerReceiptId: c.LedgerReceiptID})
	if err != nil {
		return wrapErr("get cost center", err)
	}
	return r.writeOutput(res)
}

type CostReceiptsCmd struct {
	Search CostReceiptsSearchCmd `cmd:"" help:"Search cost receipts."`
	Get    CostReceiptsGetCmd    `cmd:"" help:"Get one cost receipt by ID."`
	Create CostReceiptsCreateCmd `cmd:"" help:"Create a cost receipt."`
}

type CostReceiptsSearchCmd struct {
	CreatedStartDate *DateValue    `help:"Filter start date." name:"created-start-date"`
	Paginate         PaginateValue `default:"0:200"           help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *CostReceiptsSearchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var params procountorapi.SearchCostReceiptsParams
	if c.CreatedStartDate != nil {
		params.CreatedStartDate.SetTo(c.CreatedStartDate.Time)
	}
	res, err := r.client.SearchCostReceipts(ctx, params)
	if err != nil {
		return wrapErr("search cost receipts", err)
	}
	return r.writeOutput(sliceByPaginate(res.Results, c.Paginate.paginateSpec))
}

type CostReceiptsGetCmd struct {
	CostReceiptID int `arg:"" help:"Cost receipt ID." name:"cost-receipt-id"`
}

func (c *CostReceiptsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetCostReceipt(ctx, procountorapi.GetCostReceiptParams{CostReceiptId: c.CostReceiptID})
	if err != nil {
		return wrapErr("get cost receipt", err)
	}
	return r.writeOutput(res)
}

type CostReceiptsCreateCmd struct {
	CostReceipt *JSONInput `help:"Full cost receipt JSON document. Accepts inline JSON, @file, or - for stdin." name:"cost-receipt" required:""`
}

func (c *CostReceiptsCreateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.CostReceipt
	if err := decodeRequiredImportInput(newRequestInputReader(r), "cost-receipt", c.CostReceipt, req.Decode); err != nil {
		return err
	}
	res, err := r.client.CreateCostReceipt(ctx, &req)
	if err != nil {
		return wrapErr("create cost receipt", err)
	}
	return r.writeOutput(res)
}

type FiscalYearsCmd struct {
	List FiscalYearsListCmd `cmd:"" help:"List fiscal years in the active environment."`
}

type FiscalYearsListCmd struct{}

func (c *FiscalYearsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetFiscalYears(ctx)
	if err != nil {
		return wrapErr("get fiscal years", err)
	}
	return r.writeOutput(res)
}

type PersonsCmd struct {
	List   PersonsListCmd   `cmd:"" help:"List persons."`
	Get    PersonsGetCmd    `cmd:"" help:"Get one person by ID."`
	Update PersonsUpdateCmd `cmd:"" help:"Update a person record."`
}

type PersonsListCmd struct {
	PersonNumber    *string       `help:"Filter by person number."     name:"person-number"`
	MainPersonGroup *string       `help:"Filter by main person group." name:"main-person-group"`
	PersonGroupID   *int          `help:"Filter by person group ID."   name:"person-group-id"`
	IncludeInactive *bool         `help:"Include inactive persons."    name:"include-inactive"`
	OrderByID       *string       `enum:"ASC,DESC"                     help:"Order by ID."                                               name:"order-by-id"`
	Paginate        PaginateValue `default:"0:200"                     help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *PersonsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := buildGetPersonsParams(c)
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.PersonBasicInfo, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.GetPersons(ctx, params)
		if err != nil {
			return nil, wrapErr("get persons", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

func buildGetPersonsParams(c *PersonsListCmd) procountorapi.GetPersonsParams {
	var params procountorapi.GetPersonsParams
	if c.PersonNumber != nil {
		params.PersonNumber.SetTo(*c.PersonNumber)
	}
	if c.MainPersonGroup != nil {
		params.MainPersonGroup.SetTo(*c.MainPersonGroup)
	}
	if c.PersonGroupID != nil {
		params.PersonGroupId.SetTo(*c.PersonGroupID)
	}
	if c.IncludeInactive != nil {
		params.IncludeInactive.SetTo(*c.IncludeInactive)
	}
	if c.OrderByID != nil {
		params.OrderById.SetTo(procountorapi.GetPersonsOrderById(*c.OrderByID))
	}
	return params
}

type PersonsGetCmd struct {
	ID int `arg:"" help:"Person ID." name:"id"`
}

func (c *PersonsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetPersonById(ctx, procountorapi.GetPersonByIdParams{ID: c.ID})
	if err != nil {
		return wrapErr("get person", err)
	}
	return r.writeOutput(res)
}

type PersonsUpdateCmd struct {
	ID     int        `arg:""                                                                        help:"Person ID." name:"id"`
	Person *JSONInput `help:"Full person JSON document. Accepts inline JSON, @file, or - for stdin." name:"person"     required:""`
}

func (c *PersonsUpdateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.Person
	if err := decodeRequiredImportInput(newRequestInputReader(r), "person", c.Person, req.Decode); err != nil {
		return err
	}
	res, err := r.client.UpdatePerson(ctx, &req, procountorapi.UpdatePersonParams{ID: c.ID})
	if err != nil {
		return wrapErr("update person", err)
	}
	return r.writeOutput(res)
}

type ProductsCmd struct {
	List ProductsListCmd `cmd:"" help:"List products in one product group."`
}

type ProductsListCmd struct {
	Group string `help:"Product group, for example expense, travel, or a custom group." name:"group" required:""`
}

func (c *ProductsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.ListProducts(ctx, procountorapi.ListProductsParams{Group: c.Group})
	if err != nil {
		return wrapErr("list products", err)
	}
	return r.writeOutput(res)
}

type VATsCmd struct {
	Country  VATsCountryCmd  `cmd:"" help:"List VAT percentages for one country."`
	Company  VATsCompanyCmd  `cmd:"" help:"List VAT percentages and statuses for the active company."`
	Settings VATsSettingsCmd `cmd:"" help:"Show VAT defaults for the active company."`
}

type VATsCountryCmd struct {
	CountryCode string `help:"Country code in ISO 3166-1 alpha-2 format." name:"country-code" required:""`
}

func (c *VATsCountryCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetVatStatusesForCountry(ctx, procountorapi.GetVatStatusesForCountryParams{CountryCode: c.CountryCode})
	if err != nil {
		return wrapErr("get VAT statuses for country", err)
	}
	return r.writeOutput(res)
}

type VATsCompanyCmd struct{}

func (c *VATsCompanyCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetVatStatusesForCompany(ctx)
	if err != nil {
		return wrapErr("get VAT statuses for company", err)
	}
	return r.writeOutput(res)
}

type VATsSettingsCmd struct{}

func (c *VATsSettingsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetVatSettings(ctx)
	if err != nil {
		return wrapErr("get VAT settings", err)
	}
	return r.writeOutput(res)
}
