# DiggerTerm Agent Guidelines

## Architecture and safety

- DiggerTerm uses Go, Wails 3, Vue 3/TypeScript, and xterm.js. The GUI is an IPC client; the separate Session Daemon owns PTY, ConPTY, and SSH sessions.
- Closing, hiding, or restarting the GUI must not terminate sessions still managed by the Daemon. Sessions are not guaranteed to survive Daemon termination or operating system shutdown. Do not stop real user sessions for testing.
- Distinguish `Managed` sessions owned by the Daemon from `Observed` external processes. Observation is read-only and does not authorize terminal takeover, input injection, process termination, or other control.
- Keep Windows ConPTY, WSL, Unix PTY, and OpenSSH behavior behind appropriate platform boundaries. Document only platform features supported by the current implementation.

## Documentation and reviews

- Read current files before editing. Keep changes limited to the requested behavior, and describe final implementation rather than discarded attempts or debugging history.
- Public tracked files must not contain personal paths, machine-specific settings, private SSH aliases, local network details, temporary deployment records, unfinished plans, or acceptance-status notes.
- READMEs explain the product, available features, downloads, installation, usage, supported platforms, and actual limitations. Code comments explain currently relevant behavior and constraints; use Mermaid for diagrams when needed.
- Keep English and Simplified Chinese user-facing documentation factually consistent.
- `CHANGELOG.md` is the sole source of version changes, and GitHub Release notes are generated from the matching tagged version. PR descriptions must be bilingual and contain only completed changes and verification that actually passed. Track separate future work in Issues.
- Use the [Architecture](docs/development/ARCHITECTURE.en.md) and [Release runbook](docs/development/RELEASE.md) for their respective subjects. `CLAUDE.md` only points to this file.

## Verification and permissions

- Run checks relevant to the affected code, including `go test ./... -count=1`, `go vet ./...`, `npm --prefix frontend test`, and `npm --prefix frontend run build`. For releases, check `node scripts/verify-release.mjs vX.Y.Z` against the actual candidate version.
- Keep `testplace/` ignored and preserve user files and `icons/`. Do not stage untracked personal content or generated build artifacts.
- Commits, pushes, branch changes, resets, tags, releases, modifications to existing user assets, and external-service operations require explicit authorization for the current task.
