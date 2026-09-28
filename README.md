# Magic Hour CLI

Generate images, videos, and speech from your terminal with `mh`. It uploads
local inputs, waits for the project to finish, and downloads the result.

## Install

### Homebrew: macOS and Linux

With [Homebrew](https://brew.sh) installed:

```sh
brew install magichourhq/tap/mh
```

### Scoop: Windows

With [Scoop](https://scoop.sh) installed:

```powershell
scoop bucket add magic-hour https://github.com/magichourhq/scoop-bucket
scoop install magic-hour/mh
```

### Manual install

Download the current version from
[Releases](https://github.com/magichourhq/magic-hour-cli/releases).

#### Release archive: macOS, Linux, Windows

Download the archive for your OS and CPU, plus `checksums.txt`. Compare the
archive's SHA-256 with its entry in `checksums.txt`, extract it, and put `mh`
(`mh.exe` on Windows) on your `PATH`. Run `mh version` to check the install.

Archives include completion scripts for bash, zsh, fish, and PowerShell.

#### Linux packages

Download the `.deb` or `.rpm` for your CPU from
[Releases](https://github.com/magichourhq/magic-hour-cli/releases). On Debian or
Ubuntu:

```sh
sudo apt install ./magic-hour-cli_*.deb
```

On Fedora or RHEL:

```sh
sudo dnf install ./magic-hour-cli-*.rpm
```

These packages install shell completion but do not add an APT or DNF repository.
Download a newer package to upgrade.

## Log in

```sh
mh login
```

Approve the request in your browser. The CLI saves an API key for later commands.
If the browser does not open, visit the URL printed in your terminal.

To paste an existing [API key](https://magichour.ai/developer), run
`mh login --interactive`. Your input is hidden. For scripts and agents, set
`MAGIC_HOUR_API_KEY`; it takes precedence over a saved key. You can also pipe a
key from a secret manager into `mh login --key-stdin`.

`mh whoami` shows your account, tier, and credits. `mh logout` removes the saved
key, but does not revoke it. Revoke it in the Developer Hub if needed. An
environment key stays active after logout.

## Generate and edit images

```sh
mh image generate --prompt 'A mountain landscape at sunrise' --output landscape.png
mh image edit --image landscape.png --prompt 'Make it sunset' --output sunset.png
```

`--image` accepts a local path or URL. Local files upload automatically. Without
`--output`, files go to the current directory as `mh-<project-id>-<number>.<ext>`.
For multiple images, use `--count`. If you also set `--output`, it must point to
an existing directory. The CLI never overwrites output files.

## Generate video

```sh
mh video generate --prompt 'A corgi running through a field' --aspect-ratio 16:9
mh video generate --image photo.png --prompt 'Gentle camera movement' --duration 5
```

Without `--image`, `mh` generates video from text, requires `--prompt`, and
defaults to a 16:9 aspect ratio. `--aspect-ratio` only applies to text-to-video.
With `--image`, it animates the image; `--prompt` is optional. Both modes default
to 5 seconds. Supported durations vary by model. Use `--end-image` with `--image`
for an optional last frame. Model completion uses the image-to-video list when
`--image` is set, or both lists before a mode is chosen. The API chooses a model
and resolution if you omit them.

## Generate speech

```sh
mh audio generate --prompt 'Hello from Magic Hour' --voice 'Morgan Freeman'
mh audio clone --sample voice.mp3 --prompt 'Hello in my voice'
```

`generate` speaks text with a named voice. Complete `--voice` with Tab; names
must match an available voice. `clone` speaks text using a sample audio file or
URL. Local samples upload automatically. Both commands download the finished
audio by default. These endpoints currently return WAV files; use `.wav` for a
custom `--output` filename.

Commands wait and download by default. `--no-wait` returns the project ID
immediately; `--no-download` waits and returns output URLs. Text output shows
files or URLs first, then credits charged and the project ID. It shows status
when no completed output is available.
`--format json` returns structured fields for scripts. `--dry-run` prints the
request without calling the API or checking local input files.

## Manage projects

```sh
mh image generate --prompt 'A mountain landscape' --no-wait --format json
mh image wait PROJECT_ID
mh image download PROJECT_ID --output landscape.png
mh image get PROJECT_ID --format json
mh image delete PROJECT_ID
mh video get PROJECT_ID
mh video wait PROJECT_ID
mh video download PROJECT_ID
mh video delete PROJECT_ID
mh audio get PROJECT_ID
mh audio wait PROJECT_ID
mh audio download PROJECT_ID
mh audio delete PROJECT_ID
```

`wait` checks for completion but does not download. `download` requires a
completed project. `delete` takes effect immediately. Use `--timeout` to set a
command time limit.

## Chain commands

```sh
set -o pipefail
mh image generate --prompt 'A mountain landscape' --no-download --format json |
  mh video generate --image - --prompt 'Slowly pan across the landscape'
```

`--image -` reads one completed `mh --format json` result from stdin. JSON goes
to stdout; progress goes to stderr. Pass required flags explicitly in scripts.
Generation commands do not prompt or read stdin unless you pass `-` to an input
flag.

## Explore commands

```sh
mh --help
mh image generate --help
mh video generate --help
mh audio generate --help
mh audio clone --help
mh schema image edit
mh schema video generate
mh schema audio generate
mh completion zsh
```

`schema` prints machine-readable command definitions. Shell completion includes
model names. Homebrew and Linux packages install completion automatically;
direct-download archives include the scripts.

For contributors, `make spec` fetches the latest OpenAPI file and
`go generate ./...` rebuilds the command catalog from it and `api/cli.json`.
