package cli

import (
	"context"

	"github.com/myyra/procountor/procountorapi"
)

type DimensionsCmd struct {
	List       DimensionsListCmd       `cmd:"" help:"List dimensions."`
	Get        DimensionsGetCmd        `cmd:"" help:"Get one dimension by ID."`
	AddItem    DimensionsAddItemCmd    `cmd:"" help:"Add an item to a dimension." name:"add-item"`
	Update     DimensionsUpdateCmd     `cmd:"" help:"Update a dimension name."`
	UpdateItem DimensionsUpdateItemCmd `cmd:"" help:"Update a dimension item."    name:"update-item"`
}

type DimensionsListCmd struct {
	Name     *string `help:"Filter by dimension name."      name:"name"`
	CodeName *string `help:"Filter by dimension item name." name:"code-name"`
}

func (c *DimensionsListCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var params procountorapi.GetDimensionsParams
	if c.Name != nil {
		params.Name.SetTo(*c.Name)
	}
	if c.CodeName != nil {
		params.CodeName.SetTo(*c.CodeName)
	}

	res, err := r.client.GetDimensions(ctx, params)
	if err != nil {
		return wrapErr("get dimensions", err)
	}
	return r.writeOutput(res)
}

type DimensionsGetCmd struct {
	DimensionID int `arg:"" help:"Dimension ID." name:"dimension-id"`
}

func (c *DimensionsGetCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetDimension(ctx, procountorapi.GetDimensionParams{DimensionId: c.DimensionID})
	if err != nil {
		return wrapErr("get dimension", err)
	}
	return r.writeOutput(res)
}

type DimensionsAddItemCmd struct {
	DimensionID int     `arg:""                             help:"Dimension ID." name:"dimension-id"`
	ID          *string `help:"Dimension item ID."          name:"id"`
	CodeName    *string `help:"Dimension item code name."   name:"code-name"`
	Status      *string `help:"Dimension item status."      name:"status"`
	Description *string `help:"Dimension item description." name:"description"`
}

func (c *DimensionsAddItemCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.DimensionItem
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("id", "id", c.ID, scalarInputInt, false); err != nil {
		return err
	}
	if err := builder.SetScalar("codeName", "code-name", c.CodeName, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("status", "status", c.Status, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("description", "description", c.Description, scalarInputString, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.AddDimensionItem(ctx, &req, procountorapi.AddDimensionItemParams{DimensionId: c.DimensionID})
	if err != nil {
		return wrapErr("add dimension item", err)
	}
	return r.writeOutput(res)
}

type DimensionsUpdateCmd struct {
	DimensionID int     `arg:""                 help:"Dimension ID." name:"dimension-id"`
	Name        *string `help:"Dimension name." name:"name"`
}

func (c *DimensionsUpdateCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.DimensionName
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("name", "name", c.Name, scalarInputString, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.UpdateDimension(ctx, &req, procountorapi.UpdateDimensionParams{DimensionId: c.DimensionID})
	if err != nil {
		return wrapErr("update dimension", err)
	}
	return r.writeOutput(res)
}

type DimensionsUpdateItemCmd struct {
	DimensionID int     `arg:""                             help:"Dimension ID." name:"dimension-id"`
	ID          *string `help:"Dimension item ID."          name:"id"`
	CodeName    *string `help:"Dimension item code name."   name:"code-name"`
	Status      *string `help:"Dimension item status."      name:"status"`
	Description *string `help:"Dimension item description." name:"description"`
}

func (c *DimensionsUpdateItemCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	var req procountorapi.DimensionItem
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("id", "id", c.ID, scalarInputInt, false); err != nil {
		return err
	}
	if err := builder.SetScalar("codeName", "code-name", c.CodeName, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("status", "status", c.Status, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("description", "description", c.Description, scalarInputString, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}
	res, err := r.client.UpdateDimensionItem(ctx, &req, procountorapi.UpdateDimensionItemParams{DimensionId: c.DimensionID})
	if err != nil {
		return wrapErr("update dimension item", err)
	}
	return r.writeOutput(res)
}
