# Security policy

## Supported versions

Only the latest release of gh-fleet gets security fixes. Upgrade with
`gh extension upgrade fleet`.

## What counts

gh fleet runs your change scripts in clones of many repositories and pushes
branches and opens pull requests with your GitHub token. Please report
anything that could make it do more than you asked, for example:

- a way for repository content, a PR body or an API response to run code,
  inject into a git or shell command, or escape the clone directory
- a campaign or flag combination that pushes to a branch other than the
  campaign branch, or force-pushes over someone else's commits
- your token being sent anywhere other than the GitHub host gh is
  configured for, or written to logs or state files
- a release artifact that verifies with the commands below but was not
  built by this repository's release workflow

## Reporting a vulnerability

Do not open a public issue. Report it privately through GitHub's private
vulnerability reporting:

**<https://github.com/rossedman/gh-fleet/security/advisories/new>**

Include the version (`gh fleet version`), what an attacker could do, and steps
or a campaign file that reproduce it.

## What to expect

- An acknowledgement within 3 business days.
- A fix, or a plan with a timeline, within 30 days.
- Credit in the published advisory, unless you would rather stay anonymous.

## Verifying releases

Every release is built by `.github/workflows/release.yml` and its
`checksums.txt` is signed keylessly with [cosign](https://docs.sigstore.dev/)
through GitHub's OIDC identity. Build provenance is attested too. To verify a
download, fetch the binary, `checksums.txt` and `checksums.txt.sigstore.json`
from the release, then:

```sh
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github.com/rossedman/gh-fleet/.github/workflows/release.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
sha256sum --ignore-missing -c checksums.txt
gh attestation verify gh-fleet_v1.0.0_linux-amd64 --repo rossedman/gh-fleet
```

Use the file names of the release you downloaded. `gh extension install`
downloads the binary for your platform from the same release.
