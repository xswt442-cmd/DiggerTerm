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

**Limitations:** Session continuity depends on a running daemon; daemon failure, forced termination, sign-out, or shutdown can end sessions. Bounded output replay is not a complete VT screen snapshot. Arbitrary external-terminal screen takeover is not implemented.

**On the dev branch only (not in v0.0.1):** The sidebar **observes read-only process metadata** for current-user Windows terminal/agent processes and the default Linux user's processes within **already-running WSL distributions** (PID, parent PID, name, and start time). Linux and Windows PID namespaces are distinct. Classification is heuristic, not exhaustive. The scanner rechecks whether a WSL distribution is running before observing it, but the check and launch are not atomic, so an unexpected wake-up remains possible in a narrow race. It never reads command lines, environments, or external terminal screens, and cannot control external processes. Linux processes cannot yet be reliably linked to DiggerTerm-managed `wsl.exe` sessions. To use this feature with an older running daemon, finish existing sessions, explicitly stop it, then reopen the GUI.

### Appearance and shell startup directory (dev branch only)

Drag the separator between the sidebar and terminal, or the separator between **My sessions** and **Observed**. The focused separators also support arrow keys. Pane sizes are saved with local UI preferences.

The compact (~304 px) **Settings** popover has four tabs: **Appearance / Terminal / Sessions / Observed**. Appearance offers four muted dark palettes (Nightfall, Graphite, a softer grey-green Pine, Blue Hour), an independent **UI font size (10–16 px)**, UI language, and splitter reset. Native tray menu labels follow the UI's selected language. Terminal options include font family (falling back when not installed), size (11–20 px), line height, scrollback, cursor shape/blink, mouse wheel speed, right-click word selection and padding. Sessions lets you choose the default shell, whether new sessions become active, and a **default working directory for new local shells**. Observed controls discovery visibility, manual or periodic scanning, PID labels, and the bottom status bar.

An empty working directory uses the existing default. An existing absolute path applies only to new PowerShell, CMD or local Unix shells, not running sessions, WSL or remote SSH. The daemon validates the path. Hiding observed processes stops GUI scanning, **not any external process**. The scrollback option controls only the current xterm history and does not extend the daemon's bounded output replay.

Preferences are stored in the local desktop WebView, not in the session daemon. Theme and font changes resize terminal columns and rows without changing process ownership.

**Terminal keyboard shortcuts:** Under Settings → Terminal → Shortcuts, **Smart** (default) lets Ctrl+C copy when text is selected or interrupt the shell otherwise, while Ctrl+V pastes in a normal terminal buffer. In alternate-screen applications such as nano/vim, native Ctrl+C/V are forwarded instead (for example, Ctrl+V pages down in nano; Ctrl+O writes and Ctrl+X exits). Ctrl+Shift+C / Ctrl+Shift+V always copy/paste. You can choose **Terminal native** or **Desktop clipboard** to override the automatic split. Ctrl+A/L/R/W and other control keys reach the shell instead of triggering browser shortcuts. Windows ConPTY was tested with Ctrl+C interrupting continuous ping; GNU nano was tested under a real Ubuntu Linux PTY for save, search, cut/uncut, page down, and exit.

### SSH (dev branch only; not in the v0.0.1 download)

Choose **SSH** in the sidebar, enter a `~/.ssh/config` Host alias such as `prod` or a destination like `alice@server.example`, optionally override the port, then select **Connect**. The field suggests simple, literal `Host` aliases from the local config, without revealing credentials or expanding `Include`. OpenSSH handles first-connection host-key verification, keys/ssh-agent and password prompts in the terminal; DiggerTerm neither auto-accepts host keys nor stores credentials. Configure IPv6 destinations, ProxyJump and identity files in `~/.ssh/config`.

The sidebar also supports filtering by name/PID. The pencil button renames a managed session; its name is held by the daemon across GUI reconnects (not persisted after a daemon shutdown). **Read-only observed processes cannot be renamed or controlled.**

SSH sessions are daemon-managed: quitting the GUI does not itself disconnect the client. **SSH/network failure, a remote shell exit or daemon shutdown can still end remote work**; consider tmux for long-running tasks. This does not take over existing external SSH terminals.

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

## Links

[中文 README](README.md) · [Changelog / 更新日志](CHANGELOG.md) · [Release process (maintainer, Chinese)](RELEASE.md) · [MIT License](LICENSE)
