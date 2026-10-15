<div align="center">
  <img src="icons/diggerterm-icon.svg" alt="DiggerTerm" width="88" height="88">
  <h1>DiggerTerm</h1>
  <p>A lightweight desktop workspace for multiple terminals</p>
  <p>
    <a href="README.md">简体中文</a> ·
    <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">Releases</a> ·
    <a href="CHANGELOG.md">Changelog</a>
  </p>
  <p>
    <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
    <a href="https://github.com/xswt442-cmd/DiggerTerm/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?color=657483"></a>
    <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
  </p>
</div>

## Download

[v0.0.4 · GitHub Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.4)

| Platform | Package |
| --- | --- |
| Windows x64 | [`.exe`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.4/DiggerTerm-v0.0.4-windows-amd64.exe) |
| Linux x64 | [`.AppImage`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.4/DiggerTerm-v0.0.4-linux-x86_64.AppImage) |
| Ubuntu / Debian x64 | [`.deb`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.4/DiggerTerm-v0.0.4-linux-amd64.deb) |

Checksums: [SHA256SUMS](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.4/SHA256SUMS).

## Features

- **Multiple terminals:** sidebar navigation, search, rename, nested horizontal/vertical splits, draggable dividers and layout restore; closing a pane keeps its session running.
- **Local and remote shells:** PowerShell, CMD, Git Bash (requires Git for Windows), WSL, Linux Bash, and SSH.
- **Persistent sessions:** the daemon keeps sessions running after the GUI exits; reconnect on restart.
- **Read-only discovery:** inspect external terminal and agent process metadata without taking control.
- **Preferences:** dark themes, fonts, sizes, language, default shell, and working directory.

Smart **Ctrl+C** copies a terminal selection or interrupts without one; full-screen applications retain native control keys. The terminal toolbar and context menu provide Copy, Paste and a read-only copy mode. Try **Shift-drag** in mouse-reporting applications (Option on macOS); use Copy mode if selection is unavailable. Settings → Keys records, validates and saves shortcuts to `DiggerTerm/keybindings.json` in the current user's home directory ([example](examples/keybindings.json)).

## Notes

Copy [example JSON/CSS files](examples/themes/) into `DiggerTerm/` under your OS user's home directory, then refresh Settings → Appearance. Appearance shows the resolved directory and can open it directly. JSON `name` supplies the display name; optional `custom.css` overrides all themes. Shortcut configuration is stored separately as `keybindings.json` in the same directory.

Windows requires WebView2; binaries are unsigned. Linux builds target Ubuntu 24.04 and may not run on every distribution. Sessions are not guaranteed to survive daemon crashes or system reboots.

Settings provide manual reconnection and session exit diagnostics. New controls require the matching daemon version; finish old sessions and explicitly update the daemon before using them. Recovery replays the latest 256 KiB of raw output, so full-screen content may be incomplete. The reported SSH idle disconnect still needs reproduction in the affected environment.

[简体中文](README.md) · [CHANGELOG](CHANGELOG.md) · [Release guide](RELEASE.md) · [MIT License](LICENSE)
