---
title: ruby.star
---

# ruby.star

Bump gem versions, the Ruby version and Bundler, and set RuboCop's target Ruby.

Edits are line-based, so comments and layout in Gemfiles and gemspecs
stay as they were. Gemfile.lock isn't re-resolved; follow with a `run`
step such as `bundle lock --update <gem>` when the lockfile is committed.

```python
load("@fleet//ruby.star", "gemfiles", "set_gem", "set_ruby", "set_required_ruby", "set_bundler", "set_rubocop_target")
```

| Function | What it does |
| --- | --- |
| [`gemfiles`](#gemfiles) | Lists the Gemfiles and gemspecs. |
| [`set_gem`](#set_gem) | Sets a gem's version requirement wherever a Gemfile or gemspec lists it. |
| [`set_ruby`](#set_ruby) | Sets the Ruby version a Gemfile pins with `ruby "x"`. |
| [`set_required_ruby`](#set_required_ruby) | Sets `required_ruby_version` in every gemspec that declares it. |
| [`set_bundler`](#set_bundler) | Sets the Bundler version a Gemfile.lock was `BUNDLED WITH`. |
| [`set_rubocop_target`](#set_rubocop_target) | Sets `AllCops: TargetRubyVersion` in .rubocop.yml. |

## gemfiles

```python
gemfiles(repo)
```

Lists the Gemfiles and gemspecs.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every Gemfile, gems.rb and *.gemspec outside
vendor directories.

**Example**

```python
gemfiles(repo)
# returns ["Gemfile", "acme.gemspec"]
```

=== "Before"

    `Gemfile`

    ```text
    source "https://rubygems.org"
    ```

    `acme.gemspec`

    ```text
    Gem::Specification.new
    ```

    `vendor/bundle/x/Gemfile`

    ```text
    x
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L13)

## set_gem

```python
set_gem(repo, name, requirement)
```

Sets a gem's version requirement wherever a Gemfile or gemspec lists it.

Covers `gem "x", "~> 1.0"` with any options after it, and gemspec
`add_dependency`, `add_runtime_dependency` and
`add_development_dependency`. Gems listed without a version are left
alone, since that is a deliberate choice to follow the lockfile.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The gem name. |
| `requirement` | The requirement, such as `~> 7.2`, or a list for several, such as `[">= 1.5", "< 2"]`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_gem(repo, "rails", "~> 7.2")
set_gem(repo, "puma", [">= 6", "< 7"])
# returns ["Gemfile"]
```

=== "Before"

    `Gemfile`

    ```text
    gem "rails", "~> 7.0.8"
    gem 'puma', '>= 5', require: false
    gem "pg"
    ```

=== "After"

    `Gemfile`

    ```text
    gem "rails", "~> 7.2"
    gem 'puma', '>= 6', '< 7', require: false
    gem "pg"
    ```

**Used in:** [`ruby-rails-bump`](https://github.com/rossedman/gh-fleet/tree/main/examples/ruby-rails-bump)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L33)

## set_ruby

```python
set_ruby(repo, version)
```

Sets the Ruby version a Gemfile pins with `ruby "x"`.

`ruby file: ".ruby-version"` lines are left alone; use
`runtime.set_version(repo, "ruby", ...)` for the version file.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Ruby version or requirement, such as `3.3.5` or `~> 3.3`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_ruby(repo, "3.3.5")
# returns ["Gemfile"]
```

=== "Before"

    `Gemfile`

    ```text
    source "https://rubygems.org"
    ruby "3.1.4"
    ```

=== "After"

    `Gemfile`

    ```text
    source "https://rubygems.org"
    ruby "3.3.5"
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L81)

## set_required_ruby

```python
set_required_ruby(repo, requirement)
```

Sets `required_ruby_version` in every gemspec that declares it.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `requirement` | The requirement, such as `>= 3.1`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_required_ruby(repo, ">= 3.1")
# returns ["acme.gemspec"]
```

=== "Before"

    `acme.gemspec`

    ```text
    Gem::Specification.new do |spec|
      spec.name = "acme"
      spec.required_ruby_version = ">= 2.7"
    end
    ```

=== "After"

    `acme.gemspec`

    ```text
    Gem::Specification.new do |spec|
      spec.name = "acme"
      spec.required_ruby_version = ">= 3.1"
    end
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L112)

## set_bundler

```python
set_bundler(repo, version)
```

Sets the Bundler version a Gemfile.lock was `BUNDLED WITH`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Bundler version, such as `2.5.22`. |

**Returns:** The list of paths that changed.

**Example**

```python
set_bundler(repo, "2.5.22")
# returns ["Gemfile.lock"]
```

=== "Before"

    `Gemfile.lock`

    ```text
    DEPENDENCIES
      rails

    BUNDLED WITH
       2.4.10
    ```

=== "After"

    `Gemfile.lock`

    ```text
    DEPENDENCIES
      rails

    BUNDLED WITH
       2.5.22
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L144)

## set_rubocop_target

```python
set_rubocop_target(repo, version)
```

Sets `AllCops: TargetRubyVersion` in .rubocop.yml.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The Ruby version RuboCop should target, such as `3.3`. |

**Returns:** True when the file changed; False when the repo has no .rubocop.yml.

**Example**

```python
set_rubocop_target(repo, "3.3")
# returns True
```

=== "Before"

    `.rubocop.yml`

    ```yaml
    AllCops:
      TargetRubyVersion: 3.0
      NewCops: enable
    ```

=== "After"

    `.rubocop.yml`

    ```yaml
    AllCops:
      TargetRubyVersion: 3.3
      NewCops: enable
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/ruby.star#L178)
