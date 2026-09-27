# Magic Hour CLI

Generate and edit images from your terminal with `mh`. This alpha supports image
generation, image editing, uploads, project management, and JSON output for scripts.

## Install

The CLI, tap, and bucket are private during alpha. Homebrew and Scoop installs
become available after those repositories are public. For private testing,
download a release while signed in to GitHub.

### macOS and Linux: Homebrew

With [Homebrew](https://brew.sh) installed:

```sh
brew install magichourhq/homebrew-tap/mh
```

### Windows: Scoop

With [Scoop](https://scoop.sh) installed:

```powershell
scoop bucket add magic-hour https://github.com/magichourhq/scoop-bucket
scoop install magic-hour/mh
```

### Linux: Debian/Ubuntu or Fedora/RHEL

If the release includes `.deb` or `.rpm` packages, download the one for your
CPU from [Releases](https://github.com/magichourhq/magic-hour-cli/releases),
then install it on Debian/Ubuntu:

```sh
sudo apt install ./magic-hour-cli_*.deb
```

Or on Fedora/RHEL:

```sh
sudo dnf install ./magic-hour-cli-*.rpm
```

These local installs do not add an APT or DNF repository. Download a newer
package to upgrade.

### Direct download: any OS

Download the archive for your OS and CPU, plus `checksums.txt`, from
[Releases](https://github.com/magichourhq/magic-hour-cli/releases). Compare the
archive's SHA-256 with its entry in `checksums.txt`, extract it, and put `mh`
(or `mh.exe` on Windows) on your `PATH`.

Run `mh version` to confirm the install.

## Authenticate

Set your [Magic Hour API key](https://magichour.ai/developer):

```sh
export MAGIC_HOUR_API_KEY='your-key'
mh auth status
```

To validate and save the key for future sessions, run `mh auth login`.
`mh auth logout` removes the saved key; an environment key remains active.
You can also pipe a key from a secret manager into `mh auth login --key-stdin`.

## Generate and edit

```sh
mh image generate --prompt 'A mountain landscape at sunrise' --output landscape.png
mh image edit --image landscape.png --prompt 'Make it sunset' --output sunset.png
```

`--image` accepts a local path or URL. Local files upload automatically. Use
`--count` for multiple images, and `--dry-run` to inspect a request without
calling the API.

Commands wait and download images by default. `--no-wait` returns a project ID
immediately; `--no-download` waits but returns output URLs. Output files are
never overwritten.

## Manage images

```sh
mh image generate --prompt 'A mountain landscape' --no-wait --format json
mh image wait PROJECT_ID
mh image download PROJECT_ID --output landscape.png
mh image get PROJECT_ID --format json
mh image delete PROJECT_ID
```

`wait` does not download. `delete` is immediate. Use `--timeout` to set a
command time limit.

## Chain commands

```sh
set -o pipefail
mh image generate --prompt 'A mountain landscape' --no-download --format json |
  mh image edit --image - --prompt 'Make it sunset' --output sunset.png
```

`--image -` reads one completed `mh --format json` result from stdin. JSON output
goes to stdout; progress goes to stderr. For scripts and agents, pass required
flags explicitly and use `--format json`. Missing inputs fail with an example;
commands do not prompt or read stdin implicitly.

## Explore commands

```sh
mh --help
mh image generate --help
mh schema image edit
mh completion zsh
```

`schema` prints machine-readable command definitions. Shell completion is
available for bash, zsh, fish, and PowerShell.

For contributors: `make spec` fetches the latest OpenAPI file, and
`go generate ./...` rebuilds the command catalog from it and `api/cli.json`.
See [design](DESIGN.md) for architecture.
