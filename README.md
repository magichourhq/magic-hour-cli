# Magic Hour CLI

Generate images, video, and audio from the terminal with `mh`.

Fresh implementation. The planned design combines Go command definitions generated from OpenAPI with shared upload, generation, polling, and download flows. Commands support human-readable and machine-readable output, with explicit piping for composition. No interactive prompts in v1.

Under development; no releases yet. The first slice supports `image generate`
and `image edit` request validation and dry runs. API execution follows separately.

```sh
go build -o mh ./cmd/mh
./mh image generate --prompt 'A mountain landscape' --dry-run
./mh image edit --image photo.png --prompt 'Make it sunset' --dry-run
./mh schema image edit
./mh completion zsh
```

Missing inputs fail with an example; commands never prompt or implicitly read
stdin. Repeat `--image` for multiple editing inputs. `--dry-run` prints JSON and
does not upload files or make API calls.

## Updating commands

`api/openapi.json` is the source spec snapshot. `api/cli.json` selects supported
operations and supplies names, file roles, and CLI defaults. Run `go generate ./...`
after changing either file; commit the generated catalog with the source changes.
Unsupported schema constructs and stale field overrides fail generation.

```sh
cp ../sdk-generator/openapi.json api/openapi.json
go generate ./...
go vet ./...
go build ./...
```

The generator currently supports required objects, scalar fields, and arrays of
strings used by the first two image commands. Add support deliberately as more
operations are enabled. Deprecated fields are excluded; optional API defaults
remain server-side. CLI count defaults to one.

See [the design](DESIGN.md) for the agreed foundation and remaining decisions.
