"""Bump Composer package constraints and the PHP version in composer.json.

Edits keep composer.json's key order and indentation. composer.lock isn't
re-resolved; follow with a `run` step such as `composer update <package>
--no-install --no-scripts` when the lockfile is committed.
"""

load("./jsonfile.star", _get = "get", _set = "set")

def manifests(repo):
    """Lists the repo's composer.json files.

    Returns:
        The sorted paths of every composer.json outside vendor directories.
    Example:
        before composer.json:
            {}
        before packages/lib/composer.json:
            {}
        before vendor/x/composer.json:
            {}
        call:
            manifests(repo)
        returns:
            ["composer.json", "packages/lib/composer.json"]
    """
    return repo.glob("**/composer.json", exclude = ["**/vendor/**"])

def set_require(repo, package, constraint):
    """Sets a package's constraint wherever `require` or `require-dev` lists it.

    A newer version already there is kept, so a campaign never lowers one.

    Packages a manifest doesn't require are left alone.

    Args:
        package: the package, such as `laravel/framework`.
        constraint: the new constraint, such as `^11.0`.
    Returns:
        The list of paths that changed.
    Example:
        before composer.json:
            {
                "require": {
                    "php": "^8.1",
                    "laravel/framework": "^10.10"
                },
                "require-dev": {
                    "phpunit/phpunit": "^10.1"
                }
            }
        call:
            set_require(repo, "laravel/framework", "^11.0")
        after composer.json:
            {
                "require": {
                    "php": "^8.1",
                    "laravel/framework": "^11.0"
                },
                "require-dev": {
                    "phpunit/phpunit": "^10.1"
                }
            }
        returns:
            ["composer.json"]
    """
    changed = []
    for path in manifests(repo):
        dirty = False
        for section in ("require", "require-dev"):
            current = _get(repo, path, [section, package])
            if current != None and current != constraint and not semver.lowers(current, constraint) and _set(repo, path, [section, package], constraint):
                dirty = True
        if dirty:
            changed.append(path)
    return changed

def set_php(repo, constraint, platform = None):
    """Sets the PHP version a project requires, and optionally its `config.platform.php`.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        constraint: the `require.php` constraint, such as `^8.2`.
        platform: the PHP version Composer resolves against, such as
            `8.2.0`; set only where `config.platform.php` already exists.
    Returns:
        The list of paths that changed.
    Example:
        before composer.json:
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
        call:
            set_php(repo, "^8.2", platform = "8.2.0")
        after composer.json:
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
        returns:
            ["composer.json"]
    """
    changed = []
    for path in manifests(repo):
        have = _get(repo, path, ["require", "php"])
        a = have != None and not semver.lowers(have, constraint) and _set(repo, path, ["require", "php"], constraint)
        have = _get(repo, path, ["config", "platform", "php"]) if platform != None else None
        b = have != None and not semver.lowers(have, platform) and _set(repo, path, ["config", "platform", "php"], platform)
        if a or b:
            changed.append(path)
    return changed
