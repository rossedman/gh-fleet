---
title: php.star
---

# php.star

Bump Composer package constraints and the PHP version in composer.json.

Edits keep composer.json's key order and indentation. composer.lock isn't
re-resolved; follow with a `run` step such as `composer update <package>
--no-install --no-scripts` when the lockfile is committed.

```python
load("@fleet//php.star", "manifests", "set_require", "set_php")
```

| Function | What it does |
| --- | --- |
| [`manifests`](#manifests) | Lists the repo's composer.json files. |
| [`set_require`](#set_require) | Sets a package's constraint wherever `require` or `require-dev` lists it. |
| [`set_php`](#set_php) | Sets the PHP version a project requires, and optionally its `config.platform.php`. |

## manifests

```python
manifests(repo)
```

Lists the repo's composer.json files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every composer.json outside vendor directories.

**Example**

```python
manifests(repo)
# returns ["composer.json", "packages/lib/composer.json"]
```

=== "Before"

    `composer.json`

    ```json
    {}
    ```

    `packages/lib/composer.json`

    ```json
    {}
    ```

    `vendor/x/composer.json`

    ```json
    {}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/php.star#L10)

## set_require

```python
set_require(repo, package, constraint)
```

Sets a package's constraint wherever `require` or `require-dev` lists it.

A newer version already there is kept, so a campaign never lowers one.

Packages a manifest doesn't require are left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `package` | The package, such as `laravel/framework`. |
| `constraint` | The new constraint, such as `^11.0`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_require(repo, "laravel/framework", "^11.0")
# returns ["composer.json"]
```

=== "Before"

    `composer.json`

    ```json
    {
        "require": {
            "php": "^8.1",
            "laravel/framework": "^10.10"
        },
        "require-dev": {
            "phpunit/phpunit": "^10.1"
        }
    }
    ```

=== "After"

    `composer.json`

    ```json
    {
        "require": {
            "php": "^8.1",
            "laravel/framework": "^11.0"
        },
        "require-dev": {
            "phpunit/phpunit": "^10.1"
        }
    }
    ```

**Used in:** [`php-laravel-11`](https://github.com/rossedman/gh-fleet/tree/main/examples/php-laravel-11)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/php.star#L29)

## set_php

```python
set_php(repo, constraint, platform = None)
```

Sets the PHP version a project requires, and optionally its `config.platform.php`.

A newer version already there is kept, so a campaign never lowers one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `constraint` | The `require.php` constraint, such as `^8.2`. |
| `platform` | The PHP version Composer resolves against, such as `8.2.0`; set only where `config.platform.php` already exists. |

**Returns:** The list of paths that changed.

**Example**

```python
set_php(repo, "^8.2", platform = "8.2.0")
# returns ["composer.json"]
```

=== "Before"

    `composer.json`

    ```json
    {
        "require": {
            "php": "^8.1"
        },
        "config": {
            "platform": {
                "php": "8.1.0"
            }
        }
    }
    ```

=== "After"

    `composer.json`

    ```json
    {
        "require": {
            "php": "^8.2"
        },
        "config": {
            "platform": {
                "php": "8.2.0"
            }
        }
    }
    ```

**Used in:** [`php-laravel-11`](https://github.com/rossedman/gh-fleet/tree/main/examples/php-laravel-11)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/php.star#L78)
