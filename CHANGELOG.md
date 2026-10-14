# Changelog / 更新日志

本项目的版本变更记录以 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) 为参考，按语义化版本标识。
Changes are recorded in a format inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased] / 未发布

### 简体中文
- 暂无已宣布的变更。

### English
- No announced changes yet.

## [0.0.1] - 2026-10-14

### 简体中文

首次公开预览版本，仅发布 Windows x64 便携包。

**新增**
- Go + Wails 3 + Vue 3 / TypeScript + xterm.js 桌面工作区与会话侧栏。
- Windows PowerShell、CMD、默认 WSL 的 PTY/ConPTY 交互、会话切换、窗口调整、显式终止。
- 独立 Go Session Daemon、当前用户专属本地 IPC，以及 GUI 退出和重开后的会话重新连接。
- 系统托盘隐藏与仅退出 GUI；Daemon 在存在活跃会话时拒绝停止。

**已知限制**
- 尚未实现 SSH、外部进程观察与任意已运行终端接管。
- Daemon 本身或操作系统结束进程后不保证恢复；历史输出重放不等于完整屏幕状态。
- Linux/macOS 未经过完整 GUI 实机测试，没有正式发布包。

### English

First public preview, released as a Windows x64 portable archive.

**Added**
- Go + Wails 3 + Vue 3 / TypeScript + xterm.js desktop workspace with a session sidebar.
- Interactive PTY/ConPTY for PowerShell, CMD, and default WSL, including switching, resizing, and explicit termination.
- Separate Go Session Daemon, local per-user IPC, and reconnection after closing or restarting the GUI.
- System tray hide / GUI-only exit; daemon shutdown is refused while sessions remain active.

**Known limitations**
- SSH, external process observation, and arbitrary running-terminal takeover are not implemented.
- Sessions are not guaranteed to recover from daemon termination or an OS shutdown; raw-output replay is not a full screen-state snapshot.
- No fully tested Linux/macOS desktop builds or release packages.

[Unreleased]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.1...dev
[0.0.1]: https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.1
