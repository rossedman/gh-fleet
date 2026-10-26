---
title: license.star
---

# license.star

Detect a repo's license, update its copyright year and set the license field in package manifests.

```python
load("@fleet//license.star", "find", "detect", "set_year", "set_spdx")
```

| Function | What it does |
| --- | --- |
| [`find`](#find) | Finds the repo's license file. |
| [`detect`](#detect) | Identifies the license from the license file's text. |
| [`set_year`](#set_year) | Extends the copyright year in the license file to `year`. |
| [`set_spdx`](#set_spdx) | Sets the license field in package manifests that already have one. |

## find

```python
find(repo)
```

Finds the repo's license file.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The path of the first of LICENSE, LICENSE.md, LICENSE.txt, LICENCE,
COPYING and COPYING.md that exists, or None.

**Example**

```python
find(repo)
# returns "LICENSE.md"
```

=== "Before"

    `LICENSE.md`

    ```markdown
    MIT License
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/license.star#L23)

## detect

```python
detect(repo)
```

Identifies the license from the license file's text.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** An SPDX identifier, such as `MIT` or `Apache-2.0`, or None when
there is no license file or its text isn't recognized.

**Example**

```python
detect(repo)
# returns "MIT"
```

=== "Before"

    `LICENSE`

    ```text
    MIT License

    Copyright (c) 2021 Acme

    Permission is hereby granted, free of charge, to any person obtaining a copy
    ```

**Used in:** [`license-year`](https://github.com/rossedman/gh-fleet/tree/main/examples/license-year)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/license.star#L42)

## set_year

```python
set_year(repo, year)
```

Extends the copyright year in the license file to `year`.

`Copyright (c) 2021 Acme` becomes `Copyright (c) 2021-2026 Acme`, and an
existing range's end year is moved. Notices already at `year` and the
Apache template's `[yyyy]` placeholder are left alone. gh fleet has no
clock, so pass the year, for example from `env`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `year` | The year to extend to, as a number or string. |

**Returns:** True when the file changed.

**Example**

```python
set_year(repo, 2026)
# returns True
```

=== "Before"

    `LICENSE`

    ```text
    MIT License

    Copyright (c) 2021-2024 Acme Inc.
    ```

=== "After"

    `LICENSE`

    ```text
    MIT License

    Copyright (c) 2021-2026 Acme Inc.
    ```

**Used in:** [`license-year`](https://github.com/rossedman/gh-fleet/tree/main/examples/license-year)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/license.star#L69)

## set_spdx

```python
set_spdx(repo, spdx)
```

Sets the license field in package manifests that already have one.

Covers `license` in package.json and composer.json, `license` in
pyproject.toml's `[project]` table when it is a string, and `license`
in Cargo.toml's `[package]` table.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `spdx` | The SPDX expression, such as `Apache-2.0` or `MIT OR Apache-2.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_spdx(repo, "Apache-2.0")
# returns ["package.json", "Cargo.toml"]
```

=== "Before"

    `package.json`

    ```json
    {
      "name": "web",
      "license": "ISC"
    }
    ```

    `Cargo.toml`

    ```toml
    [package]
    name = "core"
    license = "MIT"
    ```

=== "After"

    `package.json`

    ```json
    {
      "name": "web",
      "license": "Apache-2.0"
    }
    ```

    `Cargo.toml`

    ```toml
    [package]
    name = "core"
    license = "Apache-2.0"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/license.star#L115)
