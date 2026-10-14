# DiggerTerm · Agent notes

- This is an early-stage **Go + Wails 3** desktop terminal project. Keep changes small; do not invent unfinished features.
- Go owns session/process operations; the frontend owns presentation. Avoid unnecessary services or packages.
- Distinguish **managed sessions** from **observed external processes**. Observation is read-only; external control requires explicit per-action user confirmation and OS permission.
- Closing the UI must not silently terminate sessions. Do not claim persistence or arbitrary PTY takeover before it is implemented and tested.
- Keep OS-specific PTY/ConPTY/WSL logic behind narrow interfaces. Never assume Linux process behavior on Windows.
- Check relevant code with `go test ./...` and frontend build; keep dependencies, build outputs, and secrets untracked; generated Wails TS bindings are versioned.
- Do not switch branches, commit, push, reset, or alter existing user assets unless explicitly asked.
