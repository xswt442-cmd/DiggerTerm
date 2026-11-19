# Review policy

## Branch flow

- Develop on `dev` in focused, reviewable commits.
- Before `main` is advanced, open a `dev` to `main` pull request so CI and
  review cover the exact integration diff.
- Keep `main` linear and advance it only after required CI and review issues
  are resolved. Do not treat a bot comment as authorization to merge, push or
  release.

## CodeRabbit

CodeRabbit is an advisory reviewer configured by [`.coderabbit.yaml`](../.coderabbit.yaml).
It focuses on correctness, security boundaries, platform behavior and CI
regressions; it does not replace local tests, required CI or maintainer review.

- Request a review on `dev` to `main` pull requests. Because repository size
  and service plans can affect automatic review, use `@coderabbitai review`
  when no review starts automatically.
- Prefer one full review and one incremental review after actionable fixes.
  Do not repeatedly request reviews after a rate-limit, quota, access or file
  limit error.
- Inspect each actionable finding, fix genuine defects with focused commits,
  and explain or resolve findings that do not apply with evidence.
- Keep CodeRabbit advisory: `commit_status` and request-changes workflows stay
  disabled so an unavailable bot cannot block a valid PR indefinitely.

## Pull requests

Follow the [pull request template](PULL_REQUEST_TEMPLATE.md). State only
completed work and checks that actually passed. Required CI and unresolved
correctness or security findings block completion regardless of bot status.
