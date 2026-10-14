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

[v0.0.2 · GitHub Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.2)

| Platform | Package |
| --- | --- |
| Windows x64 | [`.exe`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-windows-amd64.exe) |
| Linux x64 | [`.AppImage`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-linux-x86_64.AppImage) |
| Ubuntu / Debian x64 | [`.deb`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-linux-amd64.deb) |

Checksums: [SHA256SUMS](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/SHA256SUMS).

## Features

- **Multiple terminals:** sidebar navigation, search, rename, and resizable panes.
- **Local and remote shells:** PowerShell, CMD, WSL, Linux shells, and SSH.
- **Persistent sessions:** the daemon keeps sessions running after the GUI exits; reconnect on restart.
- **Read-only discovery:** inspect external terminal and agent process metadata without taking control.
- **Preferences:** dark themes, fonts, sizes, language, default shell, and working directory.

Keyboard shortcuts follow familiar Windows terminal behavior: **Ctrl+C** copies selected text or interrupts a running command; **Ctrl+V** pastes. Full-screen applications such as nano retain native control keys; **Ctrl+Shift+C/V** always copy/paste.

## Notes

Windows requires WebView2; binaries are unsigned. Linux builds target Ubuntu 24.04 and may not run on every distribution. Sessions are not guaranteed to survive daemon crashes or system reboots.

[简体中文](README.md) · [CHANGELOG](CHANGELOG.md) · [Release guide](RELEASE.md) · [MIT License](LICENSE)
