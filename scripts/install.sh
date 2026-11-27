#!/bin/sh
set -eu

fail() {
    printf 'mh installer: %s\n' "$*" >&2
    exit 1
}

main() {
    os=$(uname -s)
    case "$os" in
        Darwin) os=darwin ;;
        Linux) os=linux ;;
        *) fail "Unsupported operating system: $os" ;;
    esac

    arch=$(uname -m)
    case "$arch" in
        x86_64) arch=amd64 ;;
        arm64|aarch64) arch=arm64 ;;
        *) fail "Unsupported CPU architecture: $arch" ;;
    esac

    bin_dir=${HOME:?HOME must be set}/.local/bin
    mkdir -p "$bin_dir"
    tmp_dir=$(mktemp -d "$bin_dir/.mh-install.XXXXXX")
    trap 'rm -rf "$tmp_dir"' 0
    trap 'exit 1' HUP INT TERM
    cd "$tmp_dir"

    releases=https://github.com/magichourhq/magic-hour-cli/releases
    curl -q -fsSL --proto '=https' --retry 3 -o checksums.txt "$releases/latest/download/checksums.txt"

    # Read the release version from the archive name so both downloads refer to the same release.
    checksum=$(awk -v suffix="_${os}_${arch}" '
        length($1) == 64 && $1 ~ /^[0-9a-f]+$/ &&
        $2 ~ ("^mh_[0-9]+[.][0-9]+[.][0-9]+" suffix "[.]tar[.]gz$") {
            print $1 "  " $2
            exit
        }
    ' checksums.txt)
    [ -n "$checksum" ] || fail "No release archive for $os/$arch"
    archive=${checksum##* }
    version=${archive#mh_}
    version=${version%%_*}
    curl -q -fsSL --proto '=https' --retry 3 -o "$archive" "$releases/download/v$version/$archive"

    if command -v sha256sum >/dev/null 2>&1; then
        printf '%s\n' "$checksum" | sha256sum -c >/dev/null || fail "Checksum verification failed"
    elif command -v shasum >/dev/null 2>&1; then
        printf '%s\n' "$checksum" | shasum -a 256 -c >/dev/null || fail "Checksum verification failed"
    else
        fail "Install sha256sum or shasum to verify the download"
    fi

    tar -xOzf "$archive" mh > mh
    [ -s mh ] || fail "Release archive contains an empty binary"
    [ ! -d "$bin_dir/mh" ] || fail "$bin_dir/mh is a directory"
    chmod 755 mh
    mv -f mh "$bin_dir/mh"

    printf 'Installed mh %s to %s\n' "$version" "$bin_dir/mh"
    case ":${PATH-}:" in
        *":$bin_dir:"*) ;;
        *) printf '%s\n' 'Add mh to your current shell:' '  export PATH="$HOME/.local/bin:$PATH"' ;;
    esac
}

main
