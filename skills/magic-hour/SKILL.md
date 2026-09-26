---
name: magic-hour
description: Generate and transform images, videos, and audio with the mh CLI.
---

Use `mh` to call Magic Hour. Commands are noninteractive; no `--no-input` flag is
needed. Use `--format json` for execution results and check the exit status.

1. Discover commands with `mh --help` and `mh image|video|audio --help`.
2. Inspect generation inputs with `mh schema GROUP COMMAND`; do not guess model
   names, voice names, enum values, or required inputs.
3. Use `--dry-run` to validate generation inputs without API calls or uploads.
4. Execute with explicitly supplied flags. Local media uploads automatically.
5. Use the project ID with `mh image|video|audio get`, `wait`, or `download` to
   inspect or resume an existing job. Do not create a duplicate job after an
   ambiguous timeout; creation requests can incur charges even if the response
   was lost.

`MAGIC_HOUR_API_KEY` overrides saved configuration. Do not print keys or put them
in command arguments. Use `mh auth status` to check authentication.

Generation waits and downloads by default. `--no-wait` returns an unfinished
project; `--no-download` waits but returns output URLs. Results include `id`,
`type`, `status`, and `outputs`; each output has a URL, optional local path, and
optional `media_type`. A result may also have `error` after partial failure.
Reject failed results even when the API project's status is `complete`.

Use explicit stdin file flags for piping. Single-file flags require exactly one
upstream output; repeated-file inputs consume all compatible outputs.

```sh
set -o pipefail
mh image generate --prompt 'A mountain landscape' --no-download --format json |
  mh image edit --image - --prompt 'Make it sunset' --format json
```

Progress and errors go to stderr. Never mix them into piped stdout. Input streams
must close when complete. Recipes and local media editing are outside v1.
