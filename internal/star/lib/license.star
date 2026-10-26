"""Detect a repo's license, update its copyright year and set the license field in package manifests."""

load("./jsonfile.star", _jget = "get", _jset = "set")
load("./toml.star", _tget = "get", _tset = "set")

NAMES = ["LICENSE", "LICENSE.md", "LICENSE.txt", "LICENCE", "COPYING", "COPYING.md"]

# Phrases that identify common licenses, checked in order.
_SIGNATURES = [
    ("Apache-2.0", ["Apache License", "Version 2.0"]),
    ("MPL-2.0", ["Mozilla Public License Version 2.0"]),
    ("AGPL-3.0", ["GNU AFFERO GENERAL PUBLIC LICENSE", "Version 3"]),
    ("LGPL-3.0", ["GNU LESSER GENERAL PUBLIC LICENSE", "Version 3"]),
    ("GPL-3.0", ["GNU GENERAL PUBLIC LICENSE", "Version 3"]),
    ("GPL-2.0", ["GNU GENERAL PUBLIC LICENSE", "Version 2"]),
    ("BSD-3-Clause", ["Redistribution and use", "Neither the name"]),
    ("BSD-2-Clause", ["Redistribution and use"]),
    ("ISC", ["Permission to use, copy, modify, and/or distribute"]),
    ("MIT", ["Permission is hereby granted, free of charge"]),
    ("Unlicense", ["This is free and unencumbered software"]),
]

def find(repo):
    """Finds the repo's license file.

    Returns:
        The path of the first of LICENSE, LICENSE.md, LICENSE.txt, LICENCE,
        COPYING and COPYING.md that exists, or None.
    Example:
        before LICENSE.md:
            MIT License
        call:
            find(repo)
        returns:
            "LICENSE.md"
    """
    for name in NAMES:
        if repo.exists(name):
            return name
    return None

def detect(repo):
    """Identifies the license from the license file's text.

    Returns:
        An SPDX identifier, such as `MIT` or `Apache-2.0`, or None when
        there is no license file or its text isn't recognized.
    Example:
        before LICENSE:
            MIT License

            Copyright (c) 2021 Acme

            Permission is hereby granted, free of charge, to any person obtaining a copy
        call:
            detect(repo)
        returns:
            "MIT"
    """
    path = find(repo)
    if path == None:
        return None
    text = repo.read(path)
    for spdx, phrases in _SIGNATURES:
        if all([p in text for p in phrases]):
            return spdx
    return None

def set_year(repo, year):
    """Extends the copyright year in the license file to `year`.

    `Copyright (c) 2021 Acme` becomes `Copyright (c) 2021-2026 Acme`, and an
    existing range's end year is moved. Notices already at `year` and the
    Apache template's `[yyyy]` placeholder are left alone. gh fleet has no
    clock, so pass the year, for example from `env`.

    Args:
        year: the year to extend to, as a number or string.
    Returns:
        True when the file changed.
    Example:
        before LICENSE:
            MIT License

            Copyright (c) 2021-2024 Acme Inc.
        call:
            set_year(repo, 2026)
        after LICENSE:
            MIT License

            Copyright (c) 2021-2026 Acme Inc.
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    year = str(year)
    content = repo.read(path)
    lines = content.split("\n")
    for i, line in enumerate(lines):
        g = re.groups(r"^(.*\bCopyright\b(?:\s*\([cC]\)|\s*©)?\s*)(\d{4})(?:\s*[-–]\s*(\d{4}))?(.*)$", line)
        if not g:
            continue
        first, last = g[1], g[2] or g[1]
        if last >= year:
            continue
        lines[i] = g[0] + first + "-" + year + g[3]
    updated = "\n".join(lines)
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def set_spdx(repo, spdx):
    """Sets the license field in package manifests that already have one.

    Covers `license` in package.json and composer.json, `license` in
    pyproject.toml's `[project]` table when it is a string, and `license`
    in Cargo.toml's `[package]` table.

    Args:
        spdx: the SPDX expression, such as `Apache-2.0` or `MIT OR Apache-2.0`.
    Returns:
        The list of paths that changed.
    Example:
        before package.json:
            {
              "name": "web",
              "license": "ISC"
            }
        before Cargo.toml:
            [package]
            name = "core"
            license = "MIT"
        call:
            set_spdx(repo, "Apache-2.0")
        after package.json:
            {
              "name": "web",
              "license": "Apache-2.0"
            }
        after Cargo.toml:
            [package]
            name = "core"
            license = "Apache-2.0"
        returns:
            ["package.json", "Cargo.toml"]
    """
    changed = []
    for path in ["package.json", "composer.json"]:
        if repo.exists(path) and _jget(repo, path, "license") != None and _jset(repo, path, "license", spdx):
            changed.append(path)
    for path, table in [("pyproject.toml", "project"), ("Cargo.toml", "package")]:
        current = _tget(repo, path, table, "license")
        if type(current) == "string" and not current.startswith("{") and _tset(repo, path, table, "license", spdx):
            changed.append(path)
    return changed
