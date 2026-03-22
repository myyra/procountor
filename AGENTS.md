# Repository Guidelines

This repository provides a Go CLI for Procountor API. The CLI uses alecthomas/kong.

## Core commands

- Build: `just build`
- Run tests: `just test`
- Format: `just format`
- Lint: `just lint`
- Regenerate API client from OpenAPI spec: `just generate`

## Repository layout

- `cmd/procountor/main.go`: minimal binary entrypoint; only calls `cli.Execute(...)`.
- `internal/cli/`: the actual CLI implementation.
- `procountorapi/`: generated ogen client code. Do not hand-edit.
- `internal/apispec/procountor-api.yaml`: source OpenAPI spec.

## Testing

- Write unit tests for all new functionality.
- Before finishing your turn, run `just format`, `just build`, `just test`, and `just lint` and make sure they all pass.
