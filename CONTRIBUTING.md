# Contributing

Use the Go version in `go.mod`. Build with `make build`; run `make check` before
opening a PR. Keep changes focused and describe the user-visible behavior and
how you verified it. Dry runs and completion checks do not need credentials.

## Layout

- `api/openapi.json`: upstream API snapshot.
- `api/cli.json`: stable command names, flag aliases, file roles, and examples.
- `internal/generate`: compile request schemas into Go definitions.
- `internal/catalog`: generated data, validation, and input conventions.
- `internal/workflow`: API orchestration, file handling, polling, and downloads.
- `internal/cli`: Cobra commands, explicit stdin, and output formatting.
- `internal/api`: authenticated JSON calls and unauthenticated asset transfers.

Run `go generate ./...` after changing the spec or metadata. Commit both the
source change and generated result. CI rejects stale generated files, unsupported
schema keywords, missing operation mappings, and invalid file metadata. Never edit
`catalog_gen.go` directly.

For a new endpoint, add CLI metadata and verify its help, example dry run, and
schema output. Shared generation behavior belongs in the runner. Keep Cobra and
terminal streams out of the runner so future recipes can reuse it.

Real generations spend credits. Use an account you control, choose small jobs,
record project IDs, and delete only artifacts you created. Never commit API keys,
saved config, signed URLs, private prompts, or generated user media.

Dependency changes should also update `THIRD_PARTY_NOTICES.txt` when licenses or
versions change. Use `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` for a
dependency vulnerability check.
