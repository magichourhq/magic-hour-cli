#!/usr/bin/env python3
"""Update the tap formula from a completed Magic Hour CLI release."""

import re
import sys
from pathlib import Path


def update(version: str, checksums_path: Path, formula_path: Path) -> None:
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", version):
        raise ValueError(f"invalid version: {version}")

    checksums = {}
    for line in checksums_path.read_text().splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"invalid checksum for {name}")
        checksums[name] = digest

    formula = formula_path.read_text()
    formula, count = re.subn(r'(?m)^  version "[^"]+"$', f'  version "{version}"', formula)
    if count != 1:
        raise ValueError("expected one formula version")

    for platform in ("darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64"):
        archive = f"mh_{version}_{platform}.tar.gz"
        digest = checksums[archive]
        url = f"https://github.com/magichourhq/magic-hour-cli/releases/download/v{version}/{archive}"
        pattern = re.compile(
            rf'(?m)^([ \t]*)url "https://github\.com/magichourhq/magic-hour-cli/releases/download/v[^\"]+/mh_[^\"]+_{platform}\.tar\.gz"\n\1sha256 "[0-9a-f]{{64}}"$'
        )
        formula, count = pattern.subn(lambda match: f'{match[1]}url "{url}"\n{match[1]}sha256 "{digest}"', formula)
        if count != 1:
            raise ValueError(f"expected one URL and checksum for {platform}")

    formula_path.write_text(formula)


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: update-homebrew.py VERSION CHECKSUMS FORMULA")
    update(sys.argv[1], Path(sys.argv[2]), Path(sys.argv[3]))
