# DiggerTerm · Agent notes

- This is an early-stage **Go + Wails 3** desktop terminal project. Keep changes small; do not invent unfinished features.
- The same Go binary runs GUI or `--daemon`; **only the daemon owns PTYs**. The GUI is an IPC client; keep this two-process boundary small.
- Distinguish **managed sessions** from **observed external processes**. Observation is read-only; external control requires explicit per-action user confirmation and OS permission.
- GUI hide/quit/restart must never terminate daemon sessions. IPC is private to the OS user; daemon stop is explicit and forbidden while sessions run. Never claim survival of daemon crashes/reboots or arbitrary PTY takeover.
- Keep OS-specific PTY/ConPTY/WSL logic behind narrow interfaces. Never assume Linux process behavior on Windows.
- Check relevant code with `go test ./...` and frontend build; keep dependencies, build outputs, and secrets untracked; generated Wails TS bindings are versioned.
- Do not switch branches, commit, push, reset, or alter existing user assets unless explicitly asked.
