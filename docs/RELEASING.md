# Releasing

The repository is prepared for public distribution. Publishing repository
visibility and pushing release tags are separate maintainer actions.

## Validate a release candidate

```sh
go generate ./...
git diff --exit-code
make check
goreleaser check
goreleaser release --snapshot --clean
```

Use GoReleaser 2.18.2, pinned in the release workflow. Snapshot mode builds and
packages locally without publishing. Inspect `dist/` for six standalone binaries
(macOS/Linux/Windows, amd64/arm64), checksums, shell completions, source archive,
Scoop manifest, and Linux deb/rpm/apk packages.

Generate a Homebrew formula for the snapshot version listed in `dist/metadata.json`:

```sh
python3 scripts/homebrew.py 0.0.1-dev
```

The formula builds from the exact checksummed source archive. This supports both
macOS and Linux, requires Go only at installation time, and avoids unsigned-cask
quarantine workarounds. Prebuilt binaries remain available as release downloads.

## Publish

1. Merge the reviewed PRs in dependency order and ensure main's checks pass.
2. Confirm the MIT license and make the repository public when ready.
3. Push a reviewed semantic-version tag, for example `v0.1.0`.
4. The release workflow builds archives/packages, publishes the GitHub release,
   and updates `Formula/mh.rb` and `bucket/mh.json` on main for stable versions.
5. Verify archive checksums and installation on the supported platforms.

The release workflow uses the repository's `GITHUB_TOKEN`; no separate tap,
bucket, or cross-repository publishing secret is required. Its token must be able
to update the generated installer paths on main. If branch rules disallow that,
open a PR containing `dist/mh.rb` → `Formula/mh.rb` and `dist/scoop/mh.json` →
`bucket/mh.json` after the release instead of permitting direct installer updates.
Prerelease tags publish archives but do not replace the stable installer entries.

After the first public stable release:

```sh
brew tap magichourhq/cli https://github.com/magichourhq/cli
brew install magichourhq/cli/mh

scoop bucket add magichour https://github.com/magichourhq/cli
scoop install magichour/mh
```

Installers are not signed/notarized yet. Checksums accompany every release.
Homebrew builds locally; users of prebuilt macOS archives may need the normal
macOS approval flow. No release step disables operating-system security checks.

## Updating the API snapshot

The upstream `sdk-generator` workflow can open a `spec-update` PR after a labeled
API change merges. Grant its existing Magic Hour SDK Bot installation access to
this repository. New endpoint metadata still requires review; generation failures
are recorded in the PR instead of hiding the spec change.

Copy the upstream spec, regenerate, inspect the changed commands, and open a PR:

```sh
cp ../sdk-generator/openapi.json api/openapi.json
go generate ./...
make check
```

New POST endpoints fail generation until their command metadata or an explicit
utility exclusion is added. API changes are reviewed and released; installed
binaries never fetch or silently apply a new command schema.
