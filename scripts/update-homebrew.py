#!/usr/bin/env python3
"""Render the tap formula from a completed Magic Hour CLI release."""

import re
import sys
from pathlib import Path
from string import Template


def update(version: str, checksums_path: Path, formula_path: Path) -> None:
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?", version):
        raise ValueError(f"invalid version: {version}")

    checksums = {}
    for line in checksums_path.read_text().splitlines():
        digest, name = line.split()
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            raise ValueError(f"invalid checksum for {name}")
        checksums[name] = digest

    name = formula_path.stem
    classes = {"mh": "Mh", "mh-dev": "MhDev"}
    if name not in classes:
        raise ValueError(f"unsupported formula: {name}")
    values = {"VERSION": version, "FORMULA_CLASS": classes[name], "BIN_NAME": name}
    template = Path(__file__).with_name("mh.rb.template").read_text()
    for platform in ("darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64"):
        archive = f"mh_{version}_{platform}.tar.gz"
        key = f"SHA_{platform.upper()}"
        if template.count("${" + key + "}") != 1:
            raise ValueError(f"expected one {key} placeholder")
        values[key] = checksums[archive]

    formula_path.write_text(Template(template).substitute(values))


if __name__ == "__main__":
    if len(sys.argv) != 4:
        raise SystemExit("usage: update-homebrew.py VERSION CHECKSUMS FORMULA")
    update(sys.argv[1], Path(sys.argv[2]), Path(sys.argv[3]))
