package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	ht "github.com/ogen-go/ogen/http"

	"github.com/myyra/procountor/procountorapi"
)

type AttachmentsCmd struct {
	Delete  AttachmentDeleteCmd  `cmd:"" help:"Delete an attachment."`
	Content AttachmentContentCmd `cmd:"" help:"Download attachment file content."`
	Rename  AttachmentRenameCmd  `cmd:"" help:"Rename an attachment."`
	Save    AttachmentSaveCmd    `cmd:"" help:"Upload a new attachment."`
}

type AttachmentDeleteCmd struct {
	AttachmentID int `arg:"" help:"Attachment ID." name:"attachment-id"`
}

func (c *AttachmentDeleteCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	if err := r.client.DeleteAttachment(ctx, procountorapi.DeleteAttachmentParams{AttachmentId: c.AttachmentID}); err != nil {
		return wrapErr("delete attachment", err)
	}
	return r.writeOutput(map[string]bool{"ok": true})
}

type AttachmentContentCmd struct {
	AttachmentID int    `arg:""                                                  help:"Attachment ID." name:"attachment-id"`
	Out          string `help:"Write binary response to file instead of stdout." name:"out"`
}

func (c *AttachmentContentCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}
	res, err := r.client.GetAttachmentContent(ctx, procountorapi.GetAttachmentContentParams{AttachmentId: c.AttachmentID})
	if err != nil {
		return wrapErr("get attachment content", err)
	}
	reader, ok := res.(io.Reader)
	if !ok {
		return errors.New("attachment content response is not binary")
	}
	return r.writeReaderOutput(reader, c.Out)
}

type AttachmentRenameCmd struct {
	AttachmentID int     `arg:""                                help:"Attachment ID." name:"attachment-id"`
	Name         *string `help:"Attachment name."               name:"name"`
	QueryName    *string `help:"Optional query parameter name." name:"query-name"`
}

func (c *AttachmentRenameCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}

	var req procountorapi.AttachmentRename
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("name", "name", c.Name, scalarInputString, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, req.Decode); err != nil {
		return err
	}

	params := procountorapi.RenameAttachmentParams{AttachmentId: c.AttachmentID}
	if c.QueryName != nil {
		params.Name.SetTo(*c.QueryName)
	}

	res, err := r.client.RenameAttachment(ctx, &req, params)
	if err != nil {
		return wrapErr("rename attachment", err)
	}
	return r.writeOutput(res)
}

type AttachmentSaveCmd struct {
	File            LocalFilePath `help:"Path to attachment file to upload."                               name:"file"              required:""`
	Name            *string       `help:"Attachment name."                                                 name:"name"`
	ReferenceType   *string       `help:"Attachment reference type."                                       name:"reference-type"`
	ReferenceID     *string       `help:"Attachment reference ID."                                         name:"reference-id"`
	MimeType        *string       `help:"Attachment MIME type."                                            name:"mime-type"`
	SendWithInvoice *string       `help:"Whether to send the attachment with the invoice (true or false)." name:"send-with-invoice"`
}

func (c *AttachmentSaveCmd) Run(ctx context.Context, r *runner) error {
	if err := r.requireAPI(ctx); err != nil {
		return err
	}

	var meta procountorapi.Attachment
	builder := newRequestBodyBuilder(newRequestInputReader(r))
	if err := builder.SetScalar("name", "name", c.Name, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("referenceType", "reference-type", c.ReferenceType, scalarInputEnum, false); err != nil {
		return err
	}
	if err := builder.SetScalar("referenceId", "reference-id", c.ReferenceID, scalarInputInt, false); err != nil {
		return err
	}
	if err := builder.SetScalar("mimeType", "mime-type", c.MimeType, scalarInputString, false); err != nil {
		return err
	}
	if err := builder.SetScalar("sendWithInvoice", "send-with-invoice", c.SendWithInvoice, scalarInputBool, false); err != nil {
		return err
	}
	if err := decodeJSONNode(builder.body, meta.Decode); err != nil {
		return err
	}

	filePath := string(c.File)
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open --file: %w", err)
	}
	defer func() {
		_ = fileHandle.Close()
	}()

	fileInfo, err := fileHandle.Stat()
	if err != nil {
		return fmt.Errorf("stat --file: %w", err)
	}

	req := procountorapi.SaveAttachmentReq{
		Meta: meta,
		File: ht.MultipartFile{
			Name: filepath.Base(filePath),
			File: fileHandle,
			Size: fileInfo.Size(),
		},
	}
	optReq := procountorapi.NewOptSaveAttachmentReq(req)

	res, err := r.client.SaveAttachment(ctx, optReq)
	if err != nil {
		return wrapErr("save attachment", err)
	}
	return r.writeOutput(res)
}
