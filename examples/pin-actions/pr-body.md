Pins every third-party action in `{{ .Repo.FullName }}` to a full commit SHA,
keeping the tag as a trailing comment:

```yaml
uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # v4.3.0
```

A tag can be moved to point at different code; a SHA cannot. Dependabot
keeps pinned SHAs and their comments up to date.

Opened by the `{{ .Campaign.Name }}` campaign with [gh fleet](https://github.com/rossedman/gh-fleet).
