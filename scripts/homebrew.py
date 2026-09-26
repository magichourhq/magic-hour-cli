#!/usr/bin/env python3
"""Build a Homebrew tap formula from GoReleaser's exact archives."""

import hashlib
import json
import pathlib
import re

dist = pathlib.Path("dist")
metadata = json.loads((dist / "metadata.json").read_text())
version, tag = metadata["version"], metadata["tag"]
if not re.fullmatch(r"[A-Za-z0-9._-]+", version) or not re.fullmatch(r"v[0-9A-Za-z._-]+", tag):
    raise SystemExit("invalid release version or tag")


def archive(os_name, arch):
    name = f"mh_{version}_{os_name}_{arch}.tar.gz"
    digest = hashlib.sha256((dist / name).read_bytes()).hexdigest()
    return f'''      url "https://github.com/magichourhq/cli/releases/download/{tag}/{name}"
      sha256 "{digest}"'''


formula = f'''class Mh < Formula
  desc "Generate and edit images with Magic Hour"
  homepage "https://github.com/magichourhq/cli"
  version "{version}"
  license "MIT"

  on_macos do
    on_arm do
{archive("darwin", "arm64")}
    end
    on_intel do
{archive("darwin", "amd64")}
    end
  end

  on_linux do
    on_arm do
{archive("linux", "arm64")}
    end
    on_intel do
{archive("linux", "amd64")}
    end
  end

  def install
    bin.install "mh"
    bash_completion.install "completions/mh.bash" => "mh"
    zsh_completion.install "completions/mh.zsh" => "_mh"
    fish_completion.install "completions/mh.fish"
    doc.install "LICENSE", "THIRD_PARTY_NOTICES.txt"
  end
end
'''
(dist / "mh.rb").write_text(formula)
