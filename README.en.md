<p align="center"><img src="icons/diggerterm-icon.svg" alt="DiggerTerm Logo" width="104" height="104"></p>
<h1 align="center">DiggerTerm</h1>
<p align="center"><strong>A lightweight desktop terminal workspace</strong></p>
<p align="center">Manage multiple terminals in one window; keep sessions alive in a separate daemon.</p>
<p align="center">
  <a href="README.md">简体中文</a> ·
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">Download</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>
<p align="center">
  <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases"><img alt="Release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?label=release&amp;color=657483"></a>
  <img alt="Windows x64" src="https://img.shields.io/badge/Windows-x64-657483">
  <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
</p>

---

> **v0.0.1 · Windows x64 preview**
> Windows is the only officially packaged target for now. The Linux/macOS GUIs have not received full on-device validation.

## Features

- **Multiple sessions:** a session sidebar and an interactive xterm.js terminal, with input, resizing, and switching.
- **Local shells:** Windows PowerShell, Command Prompt, and an installed default WSL distribution.
- **Session continuity:** a separate Go daemon owns the PTYs. Hiding or exiting the GUI does not automatically terminate sessions; reopening the GUI reconnects to the still-running daemon.
- **Explicit control:** terminate only sessions managed by DiggerTerm. Daemon shutdown is refused while any session is running.

## Download

Get `DiggerTerm-v0.0.1-windows-amd64.zip` from [GitHub Releases](https://github.com/xswt442-cmd/DiggerTerm/releases/latest), extract it, and run `diggerterm.exe`.

Requires **Windows x64** and **Microsoft Edge WebView2 Runtime**. To use WSL, install and initialize a Linux distribution first.

> This preview is unsigned and may trigger an unknown-publisher warning in Windows.

## Session lifecycle

| Action | Behavior |
| --- | --- |
| Switch sessions / close window | Sessions keep running; closing the window hides it in the tray |
| Tray: **Quit UI** | Exits the GUI only; the daemon stays alive |
| Launch the app again | Reconnects to the still-running daemon for the same OS user |
| Sidebar: **Terminate** | Explicitly ends a DiggerTerm-managed session |
| Tray: **Stop daemon and quit** | Allowed only when no session is running |

The architecture stays intentionally small:

```text
Wails GUI + xterm.js  <-- private local IPC -->  Go Session Daemon
                                               └─ PTY / ConPTY / WSL
```

**Limitations:** Session continuity depends on a running daemon; daemon failure, forced termination, sign-out, or shutdown can end sessions. Bounded output replay is not a complete VT screen snapshot. SSH and arbitrary external-terminal screen takeover are not implemented.

**On the dev branch only (not included in the v0.0.1 download):** The Windows sidebar can now **observe metadata read-only** for common same-user terminal/AI-agent processes and their descendants (PID, parent PID, process name, start time). Detection is heuristic and cannot find every hidden agent terminal, read another PTY's screen, or control external processes; Linux processes inside WSL are not enumerated yet. An already-running older daemon must be stopped after all managed sessions finish, then restarted to expose the new feature.

## Build from source

Requires Go 1.25+, Node.js 22+, and the [Wails 3 CLI](https://v3.wails.io/) (currently `v3.0.0-beta.29`).

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
wails3 dev
```

Build and test (the frontend must be built first because Go embeds `frontend/dist`):

```powershell
npm --prefix frontend run build
go test ./...
go vet ./...
wails3 build
```

The executable is generated at `bin/diggerterm.exe`. Once all sessions are finished, you can stop the daemon explicitly with `bin/diggerterm.exe --stop-daemon`.

## Project

[中文 README](README.md) · [Changelog / 更新日志](CHANGELOG.md) · [MIT License](LICENSE)

## LICENSE

[MIT](LICENSE)
