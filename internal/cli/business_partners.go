package cli

import (
	"context"

	"github.com/myyra/procountor/procountorapi"
)

type BusinessPartnersCmd struct {
	Search          BusinessPartnersSearchCmd          `cmd:"" help:"Search business partners."`
	Get             BusinessPartnersGetCmd             `cmd:"" help:"Get one business partner by ID."`
	DefaultAccounts BusinessPartnersDefaultAccountsCmd `cmd:"" help:"Show default posting accounts for a business partner." name:"default-accounts"`
	PersonalDetails BusinessPartnersPersonalDetailsCmd `cmd:"" help:"Get current user personal details."                    name:"personal-details"`
	Put             BusinessPartnersPutCmd             `cmd:"" help:"Replace a business partner record."`
	Patch           BusinessPartnersPatchCmd           `cmd:"" help:"Update selected business partner fields."`
	Groups          BusinessPartnerGroupsCmd           `cmd:"" help:"Manage business partner groups."`
}

type BusinessPartnersSearchCmd struct {
	IdentifierType *string        `help:"Filter by identifier type." name:"identifier-type"`
	CustomerNumber *string        `help:"Filter by customer number." name:"customer-number"`
	MainGroup      *string        `help:"Filter by main group."      name:"main-group"`
	PartnerGroup   *string        `help:"Filter by partner group."   name:"partner-group"`
	Type           *string        `help:"Filter by partner type."    name:"type"`
	OrderByID      *string        `enum:"ASC,DESC"                   help:"Order by ID."                                               name:"order-by-id"`
	Active         *bool          `help:"Filter by active status."   name:"active"`
	VersionStart   *DateTimeValue `help:"Version start timestamp."   name:"version-start-date"`
	VersionEnd     *DateTimeValue `help:"Version end timestamp."     name:"version-end-date"`
	Paginate       PaginateValue  `default:"0:200"                   help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *BusinessPartnersSearchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := buildSearchBusinessPartnersParams(c)
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.BusinessPartnerBasicInfo, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.SearchBusinessPartners(ctx, params)
		if err != nil {
			return nil, wrapErr("search business partners", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

func buildSearchBusinessPartnersParams(c *BusinessPartnersSearchCmd) procountorapi.SearchBusinessPartnersParams {
	var params procountorapi.SearchBusinessPartnersParams
	if c.IdentifierType != nil {
		params.IdentifierType.SetTo(*c.IdentifierType)
	}
	if c.CustomerNumber != nil {
		params.CustomerNumber.SetTo(*c.CustomerNumber)
	}
	if c.MainGroup != nil {
		params.MainGroup.SetTo(*c.MainGroup)
	}
	if c.PartnerGroup != nil {
		params.PartnerGroup.SetTo(*c.PartnerGroup)
	}
	if c.Type != nil {
		params.Type.SetTo(procountorapi.SearchBusinessPartnersType(*c.Type))
	}
	if c.OrderByID != nil {
		params.OrderById.SetTo(procountorapi.SearchBusinessPartnersOrderById(*c.OrderByID))
	}
	if c.Active != nil {
		params.Active.SetTo(*c.Active)
	}
	if c.VersionStart != nil {
		params.VersionStartDate.SetTo(c.VersionStart.Time)
	}
	if c.VersionEnd != nil {
		params.VersionEndDate.SetTo(c.VersionEnd.Time)
	}
	return params
}

type BusinessPartnersGetCmd struct {
	ID string `arg:"" help:"Business partner ID." name:"id"`
}

func (c *BusinessPartnersGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetBusinessPartner(ctx, procountorapi.GetBusinessPartnerParams{ID: c.ID})
	if err != nil {
		return wrapErr("get business partner", err)
	}
	return r.writeOutput(res)
}

type BusinessPartnersDefaultAccountsCmd struct {
	ID int `arg:"" help:"Business partner ID." name:"id"`
}

func (c *BusinessPartnersDefaultAccountsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetDefaultAccounts(ctx, procountorapi.GetDefaultAccountsParams{ID: c.ID})
	if err != nil {
		return wrapErr("get default accounts", err)
	}
	return r.writeOutput(res)
}

type BusinessPartnersPersonalDetailsCmd struct{}

func (c *BusinessPartnersPersonalDetailsCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetPersonalDetails(ctx)
	if err != nil {
		return wrapErr("get personal details", err)
	}
	return r.writeOutput(res)
}

type BusinessPartnersPutCmd struct {
	ID              int        `arg:""                                                                                  help:"Business partner ID." name:"id"`
	BusinessPartner *JSONInput `help:"Full business partner JSON document. Accepts inline JSON, @file, or - for stdin." name:"business-partner"     required:""`
}

func (c *BusinessPartnersPutCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.BusinessPartner
	if err := decodeRequiredImportInput(newRequestInputReader(r), "business-partner", c.BusinessPartner, req.Decode); err != nil {
		return err
	}
	res, err := r.client.PutBusinessPartner(ctx, &req, procountorapi.PutBusinessPartnerParams{ID: c.ID})
	if err != nil {
		return wrapErr("put business partner", err)
	}
	return r.writeOutput(res)
}

type BusinessPartnersPatchCmd struct {
	ID              int        `arg:""                                                                    help:"Business partner ID." name:"id"`
	QueryVersion    *string    `help:"Last modification timestamp query parameter."                       name:"query-version"`
	BusinessPartner *JSONInput `help:"Business partner JSON. Accepts inline JSON, @file, or - for stdin." name:"business-partner"     required:""`
}

func (c *BusinessPartnersPatchCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}

	var req procountorapi.BusinessPartner
	if err := decodeRequiredImportInput(newRequestInputReader(r), "business-partner", c.BusinessPartner, req.Decode); err != nil {
		return err
	}

	params := procountorapi.UpdateBusinessPartnerParams{ID: c.ID}
	if c.QueryVersion != nil {
		params.Version.SetTo(*c.QueryVersion)
	}
	if err := r.client.UpdateBusinessPartner(ctx, &req, params); err != nil {
		return wrapErr("patch business partner", err)
	}
	return r.writeOutput(map[string]bool{"ok": true})
}

type BusinessPartnerGroupsCmd struct {
	List   BusinessPartnerGroupsListCmd   `cmd:"" help:"List business partner groups."`
	Get    BusinessPartnerGroupsGetCmd    `cmd:"" help:"Get one business partner group."`
	Create BusinessPartnerGroupsCreateCmd `cmd:"" help:"Create a business partner group."`
	Update BusinessPartnerGroupsUpdateCmd `cmd:"" help:"Update a business partner group."`
}

type BusinessPartnerGroupsListCmd struct {
	Name     *string       `help:"Filter by group name."    name:"name"`
	Type     *string       `help:"Filter by group type."    name:"type"`
	Active   *bool         `help:"Filter by active status." name:"active"`
	Paginate PaginateValue `default:"0:200"                 help:"Result range: <from>:<limit>, <limit>, all, or <from>:all." name:"paginate"`
}

func (c *BusinessPartnerGroupsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	params := buildGetPartnerGroupsParams(c)
	results, err := collectPageRange(c.Paginate.paginateSpec, func(page, size int) ([]procountorapi.BusinessPartnerGroup, error) {
		params.Size.SetTo(size)
		params.Page.SetTo(page)
		res, err := r.client.GetPartnerGroups(ctx, params)
		if err != nil {
			return nil, wrapErr("get partner groups", err)
		}
		return res.Results, nil
	})
	if err != nil {
		return err
	}
	return r.writeOutput(results)
}

func buildGetPartnerGroupsParams(c *BusinessPartnerGroupsListCmd) procountorapi.GetPartnerGroupsParams {
	var params procountorapi.GetPartnerGroupsParams
	if c.Name != nil {
		params.Name.SetTo(*c.Name)
	}
	if c.Type != nil {
		params.Type.SetTo(procountorapi.GetPartnerGroupsType(*c.Type))
	}
	if c.Active != nil {
		params.Active.SetTo(*c.Active)
	}
	return params
}

type BusinessPartnerGroupsGetCmd struct {
	ID int `arg:"" help:"Group ID." name:"id"`
}

func (c *BusinessPartnerGroupsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetPartnerGroup(ctx, procountorapi.GetPartnerGroupParams{ID: c.ID})
	if err != nil {
		return wrapErr("get partner group", err)
	}
	return r.writeOutput(res)
}

type businessPartnerGroupInputFlags struct {
	GroupID *string `help:"Business partner group ID."          name:"group-id"`
	Name    *string `help:"Business partner group name."        name:"name"`
	Type    *string `help:"Business partner group type."        name:"type"`
	Active  *string `help:"Business partner group active flag." name:"active"`
}

func (f *businessPartnerGroupInputFlags) apply(builder *requestBodyBuilder) error {
	if err := builder.SetScalar("id", "group-id", f.GroupID, scalarInputInt, false); err != nil {
		return err
	}
	if err := builder.SetScalar("name", "name", f.Name, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("type", "type", f.Type, scalarInputEnum, false); err != nil {
		return err
	}
	if err := builder.SetScalar("active", "active", f.Active, scalarInputBool, false); err != nil {
		return err
	}
	return nil
}

type BusinessPartnerGroupsCreateCmd struct {
	businessPartnerGroupInputFlags `embed:""`
}

func (c *BusinessPartnerGroupsCreateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.BusinessPartnerGroup
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := c.apply(builder); err != nil {
		return err
	}
	if err := builder.DecodeExplicitRequired(req.Decode); err != nil {
		return err
	}
	res, err := r.client.CreateBusinessPartnerGroup(ctx, &req)
	if err != nil {
		return wrapErr("create partner group", err)
	}
	return r.writeOutput(res)
}

type BusinessPartnerGroupsUpdateCmd struct {
	ID                             int `arg:""   help:"Group ID." name:"id"`
	businessPartnerGroupInputFlags `embed:""`
}

func (c *BusinessPartnerGroupsUpdateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.BusinessPartnerGroup
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := c.apply(builder); err != nil {
		return err
	}
	if err := builder.DecodeExplicitRequired(req.Decode); err != nil {
		return err
	}
	res, err := r.client.UpdateBusinessPartnerGroup(ctx, &req, procountorapi.UpdateBusinessPartnerGroupParams{ID: c.ID})
	if err != nil {
		return wrapErr("update partner group", err)
	}
	return r.writeOutput(res)
}
