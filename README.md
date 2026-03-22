# procountor

Lightweight Go CLI for Procountor API operations.

The CLI is a thin wrapper around:
- `internal/apispec/procountor-api.yaml`
- generated ogen client in `procountorapi/`

External consumers can import the generated client directly:

```go
import "github.com/myyra/procountor/procountorapi"
```

All API operations are exposed as explicit Kong commands, grouped by OpenAPI tag.

## Build

```bash
go build ./...
```

Run:

```bash
go run ./cmd/procountor --help
```

## Authentication Model

This CLI uses persisted login state by default.

Run once:

```bash
procountor auth login
```

`auth login` uses `--username`/`--password` (or `PROCOUNTOR_USERNAME`/`PROCOUNTOR_PASSWORD`) and prompts when needed, performs the `/login/companies -> /login` flow, and stores tokens in:
- `$XDG_STATE_HOME/procountor/auth.json`, or
- `~/.local/state/procountor/auth.json` when `XDG_STATE_HOME` is not set.

The stored access token is automatically refreshed via `/token` when expired.

Useful auth commands:
- `procountor auth companies`
- `procountor auth login`
- `procountor auth refresh`
- `procountor auth status`
- `procountor auth logout`

## Global Flags

- `--base-url` or `PROCOUNTOR_BASE_URL`: API base URL.
  - Default: `https://nemo.procountor.com/api`
- `--auth-file` or `PROCOUNTOR_AUTH_FILE`: override persisted auth state path.
- `--timeout`: HTTP timeout (default `30s`).
- `--verbose` / `-v`: print raw HTTP requests and responses to stderr.
  - Verbose output is dimmed when stderr is a terminal.
- `--output`: output format (`human`, `json`, or `plain`, default `human`).

Login credential env vars:
- `PROCOUNTOR_USERNAME`: username for `auth companies` and `auth login`.
- `PROCOUNTOR_PASSWORD`: password for `auth companies` and `auth login`.

## Request Input

Commands with request bodies use one of two primary models:
- small or patch-shaped bodies use explicit flags
- document-shaped create/replace bodies use command-specific whole-object import flags such as `--invoice`, `--ledger-receipt`, `--business-partner`, and `--cost-receipt`

For flag-shaped commands:
- top-level scalar and enum fields use normal flags
- nested object fields use embedded sub-flags such as `--payment-info.bank-account` and `--options.receipt-name`

Complex object arrays are not assembled through repeatable JSON flags. If you need to author or replace a nested list of objects, use the command's whole-document import flag instead.

Whole-object import always uses an explicit command-specific flag. Bare stdin is never treated as an implicit whole request body.

Use per-command help for the exact flag contract:
- `procountor <group> <command> --help`

Natural single-resource IDs are positional. Examples:
- `procountor invoices get 12345`
- `procountor ledger-receipts transactions update 12345 67890 --account 3000`
- `procountor attachments content 555 --out attachment.pdf`

## Binary Output

Binary endpoints support `--out`:
- `attachments content`
- `invoices image`

If `--out` is omitted, binary bytes are written to stdout.

## Pagination

Paginated search/list commands use `--paginate` and return only the result array (no `meta`/`nemoMeta` envelope).

`--paginate` formats:
- `<limit>` (same as `0:<limit>`)
- `<from>:<limit>`
- `all`
- `<from>:all`

Default: `--paginate 0:200`

## Command Groups

- `auth`
- `attachments`
- `business-partners`
- `chart-of-accounts`
- `company`
- `cost-centers`
- `cost-receipts`
- `dimensions`
- `fiscal-years`
- `invoices`
- `ledger-receipts`
- `persons`
- `products`
- `reports`
- `vats`

Use per-command help for exact flags:

```bash
procountor <group> <command> --help
```

## Schema

Use `schema` with a command path to print that command's JSON request-body schema.

Commands that do not send an `application/json` request body print nothing.

```bash
procountor schema invoices save
procountor schema reports accounting
```

## Examples

```bash
# One-time login (prompts for credentials)
procountor auth login

# Non-interactive login
PROCOUNTOR_USERNAME=my-user PROCOUNTOR_PASSWORD=my-pass procountor auth login

# Search invoices using stored auth
procountor invoices search --status SENT --paginate 50

# Machine-readable JSON output
procountor invoices search --status SENT --paginate 50 --output json

# Get one invoice
procountor invoices get 12345

# Save invoice from command-specific import flag
procountor invoices save --invoice @./invoice.json

# Update one ledger receipt transaction without roundtripping JSON manually
procountor ledger-receipts transactions update 12345 67890 --account 3000

# Download invoice image
procountor invoices image 12345 --format PNG --out ./invoice.png

# Inspect company VAT defaults and a country VAT table
procountor vats settings
procountor vats country --country-code DK

# Show auth status / clear local auth state
procountor auth status
procountor auth logout
```
