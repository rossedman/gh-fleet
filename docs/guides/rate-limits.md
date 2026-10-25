---
title: Rate limits
---

# Rate limits

Changing hundreds of repositories means hundreds of writes, which is exactly what GitHub's abuse protection watches for. gh fleet shares one HTTP transport across all its workers, and that transport enforces the policy below so a large campaign finishes instead of getting throttled.

## What gh fleet does

| Situation | Behavior |
| --- | --- |
| Writes (creating or updating PRs, labels, reviewers, GraphQL mutations) | Limited to **1 per second** across all workers by default. Reads are not throttled. |
| A secondary rate limit (`403` or `429` with `Retry-After`) | **Every worker pauses** until the wait is over, then the request is retried. |
| A `500`, `502`, `503` or `504` | Retried with exponential backoff and jitter, starting at 1 second, up to **5 attempts** in total. |
| Primary quota running low | gh fleet keeps **10% headroom**. Once a resource is down to that reserve, new requests fail with an error naming the resource and when it resets, instead of using the last of your quota. |
| Code search quota (resets every minute) | Waits for the reset instead of failing. |

When the headroom is reached, the run stops cleanly. State is saved, and running `retry` after the reset picks up what is left.

## Tuning

Change the write rate with `--write-rate`, in writes per second:

```sh
gh fleet apply --write-rate 0.5   # one write every two seconds
```

Set it permanently with `GH_FLEET_WRITE_RATE` or `write-rate` in the [config file](../configuration.md#cli-settings).

`--concurrency` (or `limits.concurrency`, default 8) sets how many repos are processed in parallel. It affects clone and script time, not the write rate: more workers share the same writes-per-second budget.

## Keeping an eye on quota

`gh fleet status` prints the remaining core and GraphQL quota after its table:

```text
API quota: core 4,871/5,000 · graphql 4,962/5,000
```

Reads are batched to stay cheap: org listings are one paginated GraphQL query each and are cached for 10 minutes, and file checks and PR lookups run 50 repos per query.
