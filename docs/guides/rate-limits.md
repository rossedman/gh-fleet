---
title: Rate limits
---

# Rate limits

Changing hundreds of repositories means hundreds of writes, which is exactly what GitHub's abuse protection watches for. gh fleet shares one HTTP transport across all its workers, and that transport follows [GitHub's rate limit rules](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api), so a large campaign finishes instead of getting throttled or blocked.

## What gh fleet does

| Situation | Behavior |
| --- | --- |
| Writes (creating or updating PRs, labels, reviewers, comments, GraphQL mutations) | Limited to **1 per second** across all workers by default, and kept within GitHub's content creation limits: **80 a minute and 500 an hour**, counted over a rolling window. Reads are not throttled. |
| A secondary rate limit with `Retry-After` | **Every worker pauses** for as long as GitHub asks, then the request is retried. |
| A secondary rate limit without `Retry-After` (a `429`, or a `403` saying so) | Every worker pauses for **1 minute**, doubling each time it happens again in a row, up to 15 minutes. |
| Primary quota spent (`x-ratelimit-remaining: 0` on a `403` or `429`, or a GraphQL `RATE_LIMITED` error) | Nothing is retried before the reset. Search quotas reset within a minute, so gh fleet waits; core and GraphQL quotas reset hourly, so the run stops with an error naming the resource and when it resets. |
| Primary quota running low | gh fleet keeps **10% headroom**. Once a resource is down to that reserve, new requests fail with the same error, instead of using the last of your quota. |
| A `500`, `502`, `503` or `504` | Retried with exponential backoff and jitter, starting at 1 second, up to **5 attempts** in total. |

Every wait is reported on stderr with how long it is, so a long silence is never a mystery. When the headroom is reached, the run stops cleanly. State is saved, and running `retry` after the reset picks up what is left.

## Tuning

Change the write rate with `--write-rate`, in writes per second:

```sh
gh fleet apply --write-rate 0.5   # one write every two seconds
```

Set it permanently with `GH_FLEET_WRITE_RATE` or `write-rate` in the [config file](../configuration.md#cli-settings).

`--concurrency` (or `limits.concurrency`, default 8) sets how many repos are processed in parallel. It affects clone and script time, not the write rate: more workers share the same writes-per-second budget.

The content creation limits are GitHub's and can't be raised. A campaign that opens a PR and adds labels and reviewers makes about three writes per repo, so past roughly 150 repos an `apply` is paced by the 500-an-hour limit. Waves (`limits.waves`) and `retry` let a long rollout span several sessions.

## Keeping an eye on quota

`gh fleet status` prints the remaining core and GraphQL quota after its table:

```text
API quota: core 4,871/5,000 · graphql 4,962/5,000
```

Reads are batched to stay cheap: org listings are one paginated GraphQL query each and are cached for 10 minutes, and file checks and PR lookups run 50 repos per query.
