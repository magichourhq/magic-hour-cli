#!/usr/bin/env python3
"""Update the Scoop manifest from a completed Magic Hour CLI release."""

import json
import re
import sys
from pathlib import Path


def update(version: str, checksums_path: Path, manifest_path: Path) -> None:
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", version):
        raise ValueError(f"invalid version: {version}")

    checksums = {}
    for line in checksums_path.read_text().splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"invalid checksum for {name}")
        checksums[name] = digest

    if manifest_path.exists():
        manifest = json.loads(manifest_path.read_text())
    elif manifest_path.stem == "mh-dev":
        manifest = json.loads(Path(__file__).with_name("mh-dev.json.template").read_text())
    else:
        raise FileNotFoundError(manifest_path)
    manifest["version"] = version
    for architecture, goarch in (("64bit", "amd64"), ("arm64", "arm64")):
        archive = f"mh_{version}_windows_{goarch}.zip"
        entry = manifest["architecture"][architecture]
        entry["url"] = f"https://github.com/magichourhq/magic-hour-cli/releases/download/v{version}/{archive}"
        entry["hash"] = checksums[archive]

    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: update-scoop.py VERSION CHECKSUMS MANIFEST")
    update(sys.argv[1], Path(sys.argv[2]), Path(sys.argv[3]))
