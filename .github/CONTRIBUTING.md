# Contributing

## Branches

Pull requests target `dev`. CI runs for pushes to `dev` and `main` and for pull requests. It includes a full-history Gitleaks scan. Maintainers fast-forward `main` before a release; merging, tagging, and publishing are maintainer actions.

## Pull request descriptions

A description states the change, the work completed, and the checks actually run, in a Simplified Chinese section and a matching English section. Deferred work belongs in an issue.

## Commit messages

Start the subject with a Conventional Commit type: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `perf:`, `build:`, `ci:`, `chore:`, or `release:`. Add a scope in parentheses when it clarifies the area (`fix(paths):`). Follow the colon with one sentence describing the change. One commit carries one type; split unrelated changes into separate commits.

## Documentation

`README.md` and `README.en.md` change together. `docs/development/ARCHITECTURE.zh-CN.md` and `.en.md` change together, and `CHANGELOG.md` keeps its `简体中文` and `English` sections in step. Tracked files contain no personal paths, credentials, private host details, or one-off logs.

## Verification

Run `go test ./... -count=1`, `go vet ./...`, `npm --prefix frontend test`, `npm --prefix frontend run build`, `node --test scripts/check-docs.test.mjs`, `node scripts/check-docs.mjs`, and `node .github/engineering/scripts/skills.mjs --check .`. CI also scans the full Git history with Gitleaks. A release also runs `node scripts/verify-release.mjs vX.Y.Z` against the candidate version. State which checks were not run.

## Version history

Release notes derive from the section for the tagged version in `CHANGELOG.md`. `docs/development/RELEASE.md` records the maintainer steps.

## Security reports

Security findings follow [SECURITY.md](SECURITY.md).
