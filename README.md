# Magic Hour CLI

Generate images, video, and audio from the terminal with `mh`.

Go command definitions generated from OpenAPI share upload, generation, polling,
and download flows. Commands support readable text and structured JSON, with
explicit piping for composition. No interactive prompts in v1.

Under development; no releases yet. All 28 image, video, and audio generation APIs
support execution, local uploads, polling, downloads, and explicit JSON piping.

## Install

Until the first release, build from this checkout:

```sh
go build -o mh ./cmd/mh
```

Release packaging includes standalone macOS/Linux/Windows binaries for Intel and
ARM, Linux deb/rpm/apk packages, Homebrew, and Scoop. See
[release and installation instructions](docs/RELEASING.md). A Magic Hour API key
is required for API calls; help, completion, schema, and dry runs work offline.

```sh
go build -o mh ./cmd/mh
./mh image generate --prompt 'A mountain landscape' --dry-run
./mh image edit --image photo.png --prompt 'Make it sunset' --dry-run
./mh schema image edit
./mh completion zsh
```

## Authentication and utilities

```sh
# Validate and save an environment key (no prompt or key argument).
mh auth login

# Or send a key from a secret manager through stdin.
secret-manager-command | mh auth login --key-stdin
mh auth status
mh auth logout

mh account get
mh saved-items list --type reference --limit 20 --format json
mh files upload portrait.png voice.wav
mh faces detect --file portrait.png --format json
mh faces get DETECTION_ID --format json
mh faces wait DETECTION_ID --format json
```

Login validates the key before saving it. Saved credentials use an atomic write
with owner-only file permissions on Unix. Logout removes saved credentials;
environment keys remain active. Windows uses the user's configuration directory.
Neither login nor any other command prompts interactively.

Saved items are paginated; pass `next_cursor` to `--cursor` for the next page.
Face detection returns face paths for `face_mappings.original_face` and preview
URLs. It waits by default, supports `--no-wait`, and does not download face crops.

Set `MAGIC_HOUR_API_KEY` or use `~/.config/magic-hour/config.json` with an `api_key`
property. `MAGIC_HOUR_CONFIG` overrides the config path; `XDG_CONFIG_HOME` overrides
the default configuration directory. Environment credentials take precedence.

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
files later. `--timeout` bounds execution (default 30 minutes). Explicit stdin has
its own timeout budget, so upstream rendering does not consume the next job's
execution time. Ctrl-C stops local work without canceling the server job.
Creation requests are not retried; polling retries up to three consecutive
transient failures and honors bounded Retry-After delays.

Default output names are `mh-PROJECT_ID-1.EXT`, `mh-PROJECT_ID-2.EXT`, etc.
`--output` accepts a filename for one output or an existing directory for multiple
outputs. Files are never overwritten. Partial downloads are removed on failure.
Deleting a project is immediate and irreversible; no confirmation prompt is shown.

Progress goes to stderr; results go to stdout. `--format json` returns project
identity, status, and an `outputs` array containing URLs and absolute local paths
when downloaded. Project retrieval also includes the raw response as `project`.
Failures exit nonzero and print an error on stderr (a JSON error object in JSON
mode); a known project ID is preserved in stdout so the operation can be resumed.
Partial failures include an `error` in JSON results; downstream commands reject
them even when the remote project completed. `outputs[].media_type` distinguishes
actual GIF/video/audio output from the project group used to manage it.

## Combining commands

```sh
set -o pipefail
./mh image generate --prompt 'A mountain landscape' --no-download --format json |
  ./mh image edit --image - --prompt 'Make it sunset'
```

`--image -` explicitly reads one `mh --format json` result from stdin. It prefers
downloaded paths when present, otherwise uses output URLs. A repeated-image input
accepts all outputs; single-file inputs require exactly one. Unfinished projects,
wrong media types, and empty outputs fail before upload or generation. Stdin is
never read implicitly. The operation timeout also bounds explicit stdin reads.

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

The generator supports the request schemas used by all current generation APIs:
objects, arrays, scalars, enums, required fields, numeric and length bounds,
patterns, and numeric steps. Deprecated fields are excluded; optional API defaults
remain server-side. CLI count defaults to one and clip start defaults to zero.

Nested arrays of objects accept JSON, for example:

```sh
./mh image face-swap --target scene.png \
  --face-mappings '[{"original_face":"api-assets/task/0-0.png","new_face":"face.png"}]'
./mh video generate --end 5 --prompt 'Animate @scene' \
  --references '[{"name":"scene","file_path":"scene.png"}]'
```

Local files inside face mappings and reference arrays upload automatically. File
or YouTube source modes are inferred when omitted. A prompt selects custom prompt
mode where the API otherwise requires a mode; face mappings select individual
face mode. Explicit conflicting inputs fail early. Clip duration/end must be
provided; v1 does not probe media duration or run FFmpeg.

Use `mh image --help`, `mh video --help`, `mh audio --help`, or `mh schema` to see
available commands. Each media group has its own `get/wait/download/delete`
commands, so project type is never guessed.

See [the design](DESIGN.md) for the architecture and scope.

See [contributing](CONTRIBUTING.md) for the code layout and update flow, and the
[agent skill](skills/magic-hour/SKILL.md) for machine-oriented usage. Licensed
under [MIT](LICENSE); dependency licenses are in [third-party notices](THIRD_PARTY_NOTICES.txt).
