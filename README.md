# Magic Hour CLI

Generate and edit images from the terminal with `mh`.

Commands are generated from OpenAPI and share upload, polling, and download
flows. Results support text and JSON output, with explicit piping between commands.

Alpha release. `image generate` and `image edit` support
API execution, local uploads, polling, downloads, and explicit JSON piping.

Tagged releases build static `mh` binaries for macOS, Linux, and Windows on
amd64 and arm64. Archives include shell completions and license notices; release
checksums are published alongside them. Download your archive from
[Releases](https://github.com/magichourhq/cli/releases), verify it against
`checksums.txt`, then put `mh` on your `PATH`. A release starts when a reviewed
PR changes `VERSION` on `main`. The workflow tags that commit, builds the
binaries, and creates the GitHub release. To build from source:

```sh
go build -o mh ./cmd/mh
./mh image generate --prompt 'A mountain landscape' --dry-run
./mh image edit --image photo.png --prompt 'Make it sunset' --dry-run
./mh schema image edit
./mh completion zsh
```

Set `MAGIC_HOUR_API_KEY`, or validate and save a key with `mh auth login`:

```sh
export MAGIC_HOUR_API_KEY='your-key'
./mh auth login
./mh auth status
./mh auth logout
```

For a key from a secret manager, pipe it to `mh auth login --key-stdin`. Login
validates the key before saving it. Neither command prompts or prints the key.
Saved keys live in `~/.config/magic-hour/config.json` on macOS/Linux and the
user config directory on Windows. `MAGIC_HOUR_CONFIG` overrides the config path;
`XDG_CONFIG_HOME` overrides the default directory. Environment credentials take
precedence, including after logout. You can also set the config file directly
with an `api_key` property.

```sh
./mh image generate --prompt 'A mountain landscape' --output landscape.png
./mh image generate --prompt 'A mountain landscape' --no-wait --format json
./mh image wait PROJECT_ID
./mh image download PROJECT_ID --output landscape.png
./mh image get PROJECT_ID --format json
./mh image delete PROJECT_ID
```

Generation waits and downloads by default. `--no-wait` returns the project ID;
`--no-download` waits and returns URLs. `wait` only waits; use `download` to fetch
files later. `--timeout` bounds execution (default 30 minutes). Ctrl-C stops local
work without canceling the server job. Creation requests are not retried.

Default output names are `mh-PROJECT_ID-1.EXT`, `mh-PROJECT_ID-2.EXT`, etc.
`--output` accepts a filename for one output or an existing directory for multiple
outputs. Files are never overwritten. Partial downloads are removed on failure.
Deleting a project is immediate and irreversible; no confirmation prompt is shown.

Progress goes to stderr; results go to stdout. `--format json` returns project
identity, status, and an `outputs` array containing URLs and absolute local paths
when downloaded. Project retrieval also includes the raw response as `project`.
Failures exit nonzero and print an error on stderr (a JSON error object in JSON
mode); a known project ID is preserved in stdout so the operation can be resumed.
In JSON mode, the stdout result also includes `error` so downstream commands
reject failed upstream work.

## Combining commands

```sh
set -o pipefail
./mh image generate --prompt 'A mountain landscape' --no-download --format json |
  ./mh image edit --image - --prompt 'Make it sunset'
```

`--image -` explicitly reads one `mh --format json` result from stdin. It prefers
downloaded paths when present, otherwise uses output URLs. A repeated-image input
accepts all outputs; single-file inputs require exactly one. Unfinished projects,
wrong media types, failed upstream commands, and empty outputs fail before upload
or generation. Stdin is never read implicitly. `--timeout` separately bounds
explicit stdin reading and execution, so waiting for an upstream command does
not consume the downstream generation limit.

File flags accept local paths, HTTP(S) URLs, or durable API file paths (such as
`api-assets/...`). Local files upload automatically; remote references pass
through. All local inputs are checked before uploading any files. Dry runs do
not upload or check local file contents.

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
