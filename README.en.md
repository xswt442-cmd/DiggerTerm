<p align="center"><img src="icons/diggerterm-icon.svg" alt="DiggerTerm Logo" width="104" height="104"></p>
<h1 align="center">DiggerTerm</h1>
<p align="center"><strong>A focused, lightweight desktop terminal workspace</strong></p>
<p align="center">Multiple terminals · background sessions · SSH · read-only process discovery</p>
<p align="center">
  <a href="README.md">简体中文</a> ·
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">Downloads</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>
<p align="center">
  <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases"><img alt="Release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?label=release&amp;color=657483"></a>
  <img alt="Windows x64" src="https://img.shields.io/badge/Windows-x64-657483">
  <img alt="Linux x64" src="https://img.shields.io/badge/Linux-x64-657483">
  <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
</p>

---

> **v0.0.2 · Windows / Linux x64 preview.** Windows packages are unsigned. Linux builds are primarily tested on Ubuntu 24.04; compatibility with other distributions requires testing.

## Download and install

Download the appropriate package from [v0.0.2 Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.2):

| Platform | File | Usage |
| --- | --- | --- |
| Windows x64 | `DiggerTerm-v0.0.2-windows-amd64.exe` | Run directly; Microsoft Edge WebView2 Runtime required |
| Linux x64 | `DiggerTerm-v0.0.2-linux-x86_64.AppImage` | `chmod +x *.AppImage`, then execute; host graphics dependencies may be required |
| Ubuntu/Debian x64 | `DiggerTerm-v0.0.2-linux-amd64.deb` | `sudo apt install ./DiggerTerm-v0.0.2-linux-amd64.deb` |

The Release also includes `SHA256SUMS`. On Linux run `sha256sum -c SHA256SUMS` in the download directory; on Windows compare `Get-FileHash .\DiggerTerm-v0.0.2-windows-amd64.exe -Algorithm SHA256` in PowerShell against the published hash.

The historical v0.0.1 Windows ZIP remains on its [original Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.1). **v0.0.2 ships only these three packages and the checksum file.**

## Features

- **Multi-session terminal:** Vue 3 + xterm.js, sidebar switching/search/rename and draggable layout separators.
- **Local shells / WSL:** Windows PowerShell and CMD, installed WSL distributions, and local Linux shells.
- **SSH:** interactive sessions through system OpenSSH, with simple `~/.ssh/config` Host aliases and optional ports. OpenSSH handles fingerprint, key, agent and password authentication; DiggerTerm does not store SSH credentials.
- **Background daemon:** a separate Go process owns DiggerTerm-created PTYs/ConPTYs. Closing the window normally minimizes to the tray; exiting the GUI does not terminate managed sessions, and a new GUI can reconnect.
- **Read-only discovery:** limited process metadata for current-user Windows terminals and agents, and selected processes for the default Linux user in **already-running WSL distributions**. No external terminal screen capture, takeover or process control.
- **Persistent preferences:** four muted dark themes; independent UI (10–16 px) and terminal (11–20 px) sizes, fonts, line spacing, cursors, scrollback, language, default local shell and working directory, observation refresh settings. Native tray labels follow the UI language.

### Terminal shortcuts

Choose **Settings → Terminal → Shortcuts**:

| Mode | Ctrl+C | Ctrl+V |
| --- | --- | --- |
| **Smart (default)**: normal shell | Copy a selection, otherwise interrupt | Paste clipboard text |
| **Smart**: nano/vim alternate screen | Hand over to the program | Hand over (page down in nano) |
| **Terminal native** | Hand over to the program | Hand over to the program |
| **Desktop clipboard** | Copy a selection, otherwise interrupt | Always paste |

`Ctrl+Shift+C` and `Ctrl+Shift+V` always copy/paste (macOS has separate Cmd conventions; no macOS build is published yet). Nano receives `Ctrl+O/W/K/U/X`; shell control keys such as Ctrl+A/L/R are forwarded. Automated Windows ConPTY interrupt and real GNU nano Linux PTY tests pass; test GUI clipboard behavior on your own desktop.

### Session lifetime and boundaries

| Action | Result |
| --- | --- |
| Switch sessions / close the GUI window | Sessions remain running; close normally hides to tray |
| Tray “Quit UI” | Exits only the GUI, leaving the daemon and its sessions running |
| Start DiggerTerm again | Reconnects to a daemon still running under the same OS user |
| Terminate a managed session | Explicitly stops that DiggerTerm-owned process only |
| Tray “Stop daemon” | **Allowed only when no managed sessions remain running** |

```text
Wails 3 GUI + xterm.js  <-- private per-user IPC --> Go Session Daemon
                                                └─ PTY / ConPTY / WSL / SSH
```

**Limitations:** Sessions are **not guaranteed to survive daemon crashes, OS sign-out or reboot**. Bounded output replay is not a full terminal-screen snapshot. SSH disconnection or remote shell exit can still end remote work; use tmux on remote servers for long-running jobs.

Discovery is heuristic and may miss processes; Windows and Linux PIDs are in different namespaces. A WSL running-state check is repeated before scanning but a small check-to-execution race remains, so a previously stopped distribution might still wake up. DiggerTerm **does not take over external PTYs or inspect their command lines, environments or screen content**.

Configure advanced SSH options (IPv6, ProxyJump, identity files) through local `~/.ssh/config`. Host suggestions only parse literal aliases, not `Include` directives. Default startup directories apply to **new local shells only**, not existing sessions, WSL or SSH. Preferences live in local WebView storage, not daemon configuration.

## Build from source

Requires **Go 1.25+**, **Node.js 24** (used by CI) and [Wails 3 CLI v3.0.0-beta.29](https://v3.wails.io/):

```shell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
npm --prefix frontend run test:preferences
npm --prefix frontend run test:shortcuts
npm --prefix frontend run build
go test ./...
go vet ./...
```

On Windows run `wails3 build` from the project root to produce `bin/diggerterm.exe`. On Linux (Ubuntu 24.04), install GTK4/WebKitGTK 6 development dependencies (see the [Release workflow](.github/workflows/release.yml)), then use `CGO_ENABLED=1 go build -tags production -o bin/diggerterm .`. See [RELEASE.md](RELEASE.md) for final packaging.

Once all managed sessions have exited, `diggerterm --stop-daemon` explicitly shuts down the idle daemon.

[中文 README](README.md) · [Changelog](CHANGELOG.md) · [Release process (Chinese)](RELEASE.md) · [MIT License](LICENSE)
