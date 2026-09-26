# CLI design

Status: initial direction agreed; implementation details remain open.

## Audience and scope

Creators, developers, and agents should be able to complete a generation with
one command. Human-readable and machine-readable output share the same command
definitions and execution logic.

Start with API orchestration and local file handling. No interactive prompts in
v1: missing required flags produce errors with examples. A persistent terminal
workspace and local media editing are outside the initial scope.

## Agreed foundation

- Go for standalone binaries across macOS, Linux, and Windows.
- Cobra for command parsing, help, and shell completion.
- A Go command catalog generated from OpenAPI plus a small amount of explicit
  CLI metadata. Generate definitions, not duplicated execution handlers.
- Shared execution logic for upload, generation, polling, and download.
- Handwritten behavior for workflows the API schema cannot describe.

```text
OpenAPI + CLI metadata
          ↓
 generated Go catalog
          ↓
    Cobra commands
          ↓
 shared workflow runner
 upload → create → wait → download
```

The catalog separates API definitions from command presentation and execution.
The runner should not depend on Cobra or an interactive prompt library.

## Responsibility boundaries

| Source | Responsibility |
| --- | --- |
| OpenAPI | API paths, methods, request fields, types, required fields, enums, descriptions |
| CLI metadata | Stable command names, file input roles, project kinds, common options, examples |
| Cobra adapter | Flags, help, completion, invocation |
| Shared runner | Input preparation, validation, upload, create, wait, download |
| Terminal adapter | Progress display on stderr |
| Output adapter | Human-readable results and machine-readable results |
| Workflow-specific code | Behaviors such as face detection and face selection |

Do not infer workflow behavior or project kinds from prose descriptions. Store
those decisions explicitly. Add abstractions only when the first working command
needs them; these boundaries do not require separate packages immediately.

## Command layout

| Group | Commands |
| --- | --- |
| `image` | `generate`, `edit`, `gif`, `upscale`, `remove-background`, `colorize`, `face-swap`, `head-swap`, `body-swap`, `edit-face`, `change-clothes`, `headshot`, `meme`, `qr-code` |
| `video` | `generate`, `from-image`, `from-audio`, `edit`, `restyle`, `animate`, `talking-photo`, `lip-sync`, `face-swap`, `replace-character`, `translate`, `subtitle` |
| `audio` | `speech`, `clone-voice` |
| `faces` | `detect`, `get`, `wait` |
| `files` | `upload` |
| `saved-items` | `list` |
| `account` | `get` |
| `auth` | `login`, `status`, `logout` |

Each of `image`, `video`, and `audio` also has `get ID`, `wait ID`,
`download ID`, and `delete ID`. The group determines the project API route;
there is no separate `projects` group or required `--type` flag. GIFs use image
project management. Register shared management handlers with the group's type.

Top-level utility commands: `completion`, `schema`, and `version`.

## Workflows and composition

Each generation command runs validation, local file uploads, job creation,
polling, and download through the shared runner. By default it waits and downloads.
`--no-wait` returns the project ID and type immediately; `--no-download` waits but
returns output URLs instead of downloading.

Support composition through explicit stdin flags such as `--image -`:

```sh
mh image generate --prompt "A mountain landscape" --no-download --format json |
  mh video from-image --image - --prompt "Slow camera pan"
```

The machine-readable result must carry project identity, media type, and output
references (URLs or local paths). Progress goes to stderr. Validate input media
types and reject ambiguous output selection. An unfinished `--no-wait` result
cannot serve as finished media input. Scripts should enable `pipefail` to retain
upstream failures. Exact result fields and selection syntax remain to be designed.

Keep the runner callable without Cobra, terminal input, or parsing human-readable
output. A future recipe executor can call those same operations and pass their
structured results between steps. Recipes may reference earlier step outputs,
but recipe syntax, execution, retries, and resume are deferred. Do not build a
recipe engine or speculative extension framework in v1.

## Spec updates

Spec updates should produce a reviewable pull request, followed by a CLI release.
Existing installed binaries do not change when the source spec changes.
Regenerate Go definitions as part of the update; CI should detect stale generated
files and unsupported schema features before release.

Compatible API fields should flow through the catalog without handwritten flag
code. New workflows and incompatible changes may need CLI metadata or custom
logic changes. Spec updates do not automatically guarantee a usable new command.

## Decisions still open

- Final flag conventions and output/error contracts.
- Exact structured stdin format and selection behavior for multiple outputs.
- Authentication and output filename behavior for the first command.

## First implementation slice

Implement one image-generation command end to end: flags, required-input
handling, API call, polling, download, and JSON output. Use it to validate the
catalog and runner boundaries before expanding API coverage. Do not copy the old
prototype wholesale.

## Reference

[Stripe CLI](https://github.com/stripe/stripe-cli) separates generated operation
definitions from shared command execution and handwritten workflows. Follow that
separation with build-time Go generation and a shared generation runner.
