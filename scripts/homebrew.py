#!/usr/bin/env python3
"""Generate a source Homebrew formula from the exact released source archive."""
import hashlib
import pathlib
import re
import sys

version = sys.argv[1]
if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?", version):
    raise SystemExit("expected a semantic version without the v prefix")
source = pathlib.Path("dist") / f"mh_{version}_source.tar.gz"
checksum = hashlib.sha256(source.read_bytes()).hexdigest()
formula = f'''class Mh < Formula
  desc "Generate images, video, and audio with Magic Hour"
  homepage "https://github.com/magichourhq/cli"
  url "https://github.com/magichourhq/cli/releases/download/v{version}/{source.name}"
  sha256 "{checksum}"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-trimpath", "-ldflags", "-s -w -X main.version=#{{version}}",
           "-o", bin/"mh", "./cmd/mh"
    generate_completions_from_executable(bin/"mh", "completion")
  end
end
'''
pathlib.Path("dist/mh.rb").write_text(formula)
