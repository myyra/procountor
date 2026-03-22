package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

type AbsoluteURL string

func (u *AbsoluteURL) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return invalidUsage("invalid --%s: %v", cliValueName(ctx), err)
	}
	if !parsed.IsAbs() || parsed.Host == "" {
		return invalidUsage("invalid --%s: must be an absolute URL", cliValueName(ctx))
	}
	*u = AbsoluteURL(parsed.String())
	return nil
}

type LocalFilePath string

func (p *LocalFilePath) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return invalidUsage("--%s cannot be empty", cliValueName(ctx))
	}
	*p = LocalFilePath(trimmed)
	return nil
}

type DateValue struct {
	time.Time
}

func (d *DateValue) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return invalidUsage("invalid --%s: expected YYYY-MM-DD", cliValueName(ctx))
	}
	d.Time = parsed
	return nil
}

type DateTimeValue struct {
	time.Time
}

func (d *DateTimeValue) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return invalidUsage("invalid --%s: expected RFC3339", cliValueName(ctx))
	}
	d.Time = parsed
	return nil
}

type PaginateValue struct {
	paginateSpec
}

func (p *PaginateValue) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	spec, err := parsePaginateValue(value)
	if err != nil {
		return err
	}
	p.paginateSpec = spec
	return nil
}

type JSONInput string

func (j *JSONInput) Decode(ctx *kong.DecodeContext) error {
	value, err := popRequiredCLIValue(ctx)
	if err != nil {
		return err
	}
	*j = JSONInput(value)
	return nil
}

func cliValueName(ctx *kong.DecodeContext) string {
	if ctx != nil && ctx.Value != nil && ctx.Value.Name != "" {
		return ctx.Value.Name
	}
	return "value"
}

func popCLIValue(ctx *kong.DecodeContext) (string, error) {
	var value string
	if err := ctx.Scan.PopValueInto("value", &value); err != nil {
		return "", fmt.Errorf("read %s: %w", cliValueName(ctx), err)
	}
	return strings.TrimSpace(value), nil
}

func popRequiredCLIValue(ctx *kong.DecodeContext) (string, error) {
	value, err := popCLIValue(ctx)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", invalidUsage("--%s cannot be empty", cliValueName(ctx))
	}
	return value, nil
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}
