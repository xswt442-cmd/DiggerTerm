# 更新日志 / Changelog

记录已交付的桌面功能、修复、平台支持及兼容性变化。GitHub Release 从对应 Tag 的本文件生成。

This file records shipped desktop features, fixes, platform support, and compatibility changes. Release notes are generated from the matching tagged version.

## [Unreleased]

### 简体中文

- 新增终端缓冲区搜索、键盘快速选择和命令面板，支持在工作区内定位并执行可用操作。
- 新增 Dig 进程观察的变化提示与关注状态，保持不同 Windows/WSL 来源及进程身份的边界。
- 新增会话输出活动提示，改善终端搜索、选区操作和外部链接处理。
- 改进终端工作区控件布局、焦点边框和快捷键相关设置。

### English

- Added terminal-buffer search, keyboard-driven Quick Select, and a command palette for finding and invoking available workspace actions.
- Added Dig process-observation change indicators and watch state while preserving Windows/WSL source and process-identity boundaries.
- Added session-output activity indicators and improved terminal search, text selection, and external-link handling.
- Refined terminal workspace controls, focused-pane borders, and shortcut settings.

## [0.0.5] - 2026-10-16

### 简体中文

- Dig 新增 Windows 及已运行 WSL 的只读进程树，提供终端/Agent 名称标记、父链推断、搜索、筛选和元数据详情。
- 搜索进程时自动展开命中项的祖先，保留手动折叠偏好，并在进程身份变化或消失后清理旧状态。
- 托盘退出菜单区分停止空闲 Daemon 和确认后终止全部托管会话；批量终止前重新检查目标会话。
- 慢速只读进程扫描不会阻塞同一 GUI 连接的会话请求；并发扫描有限制，客户端断开时取消扫描。

### English

- Dig adds a read-only Windows/WSL process tree with terminal/Agent name labels, inferred parent chains, search, filters, and metadata details.
- Search expands matching ancestors, preserves manual fold preferences, and clears stale state when process identities change or disappear.
- Tray quit actions distinguish stopping an idle daemon from confirming termination of all managed sessions; targets are rechecked before bulk termination.
- Slow read-only process scans no longer block session requests over the same GUI connection; scans are concurrency-limited and canceled on disconnect.

## [0.0.4] - 2026-10-16

### 简体中文

- 设置增加帮助页，展示主题 JSON/CSS、全局 `custom.css` 和快捷键配置路径，并提供版本检查。
- 快捷键支持逐项恢复默认值，快捷键模式移入快捷键设置。
- 优化首页空状态和托盘退出菜单；无会话时分屏后隐藏欢迎层。
- 主题与快捷键配置保存在当前用户主目录下的 `DiggerTerm/`。

### English

- Added a Help page with theme JSON/CSS, global `custom.css`, keybinding locations, and version checks.
- Added per-shortcut reset to defaults and moved keyboard mode into shortcut settings.
- Improved the home empty state and tray quit actions; the welcome view hides after an empty workspace is split.
- Theme and keybinding files share `DiggerTerm/` under the current user's home directory.

## [0.0.3] - 2026-10-15

### 简体中文

- 新增用户 JSON/CSS 配色主题、外观刷新及快捷键录制、冲突检测和保存。
- 新增嵌套终端分屏与布局恢复；每个托管会话由一个 GUI 控制端持有，关闭面板仅分离视图。
- 改进全屏程序选区复制，增加工具栏、右键操作和只读复制模式；Windows 剪贴板具备原生后备处理。
- 新增 Daemon 手动重连、会话退出码、PTY 流诊断和输出截断提示。
- Windows 新增 Git Bash，未安装时不会自动转到 WSL；Linux/macOS 使用本机 Bash。
- 终端重连回放最近 256 KiB 原始输出，全屏程序的历史画面可能无法精确还原。SSH 空闲连接依赖远端主机及网络策略。

### English

- Added user JSON/CSS themes, refresh, and recording, conflict detection, and persistence for shortcuts.
- Added nested terminal panes and layout recovery; only one GUI controls each managed session, while closing a pane detaches the view.
- Improved fullscreen-program selection copying with toolbar, context menu, read-only copy mode, and a native Windows clipboard fallback.
- Added manual daemon reconnect, session exit codes, PTY stream diagnostics, and output-truncation warnings.
- Added Git Bash on Windows without falling back to WSL if not installed; Linux/macOS use native Bash.
- Reconnection replays the latest 256 KiB of raw output, so historical fullscreen views may not be restored exactly. Idle SSH sessions depend on remote host and network policy.

## [0.0.2] - 2026-10-15

### 简体中文

- Windows/WSL 支持同用户外部终端与 Agent 进程的只读元信息观察，只扫描已经运行的 WSL 发行版；不接管外部 PTY。
- 使用系统 OpenSSH 建立 Daemon 托管的 SSH 会话，支持 ssh_config 别名、端口及系统原生认证方式；不存储 SSH 凭据。
- 支持托管会话重命名与侧栏搜索，从本机 OpenSSH 配置提供 Host 别名建议。
- 优化界面布局和设置：分栏拖动、终端/界面字号、字体、回滚行数、光标、滚动、Shell 和工作路径等。
- 改进 Ctrl+C、Ctrl+V 与终端程序的快捷键处理，修复 Windows ConPTY 前台进程接收 Ctrl+C 的问题。

### English

- Added read-only metadata observation for same-user external terminal/Agent processes on Windows and running WSL distributions, without external PTY control.
- Added daemon-managed SSH sessions through system OpenSSH, supporting ssh_config aliases, ports, and native authentication without storing credentials.
- Added managed-session renaming, sidebar search, and OpenSSH Host alias suggestions.
- Improved layout and settings for splitters, UI/terminal fonts, scrollback, cursor, scrolling, Shell, and working directory.
- Improved Ctrl+C/Ctrl+V shortcut handling for terminal programs and fixed foreground Ctrl+C delivery for Windows ConPTY.

## [0.0.1] - 2026-10-14

### 简体中文

首个公开预览版本，提供 Windows x64 便携包。

- 基于 Go、Wails 3、Vue 3/TypeScript 和 xterm.js 的桌面终端工作区与侧栏。
- 支持 Windows PowerShell、CMD 和默认 WSL 的 PTY/ConPTY 交互、切换、调整大小与显式终止。
- 独立 Session Daemon、当前用户专属本地 IPC、GUI 退出后重新连接，以及托盘隐藏功能。
- Daemon 有活跃会话时拒绝停止；Daemon 或操作系统终止后不保证恢复会话，原始输出回放无法完整恢复屏幕状态。

### English

Initial public preview with a Windows x64 portable build.

- Added a Go/Wails 3 desktop terminal workspace with Vue 3/TypeScript and xterm.js.
- Supported PowerShell, CMD, and default WSL PTY/ConPTY interaction, switching, resizing, and explicit termination.
- Added a separate Session Daemon, user-private local IPC, GUI reconnection, and system tray hiding.
- Daemon refuses to stop while managed sessions remain active. Sessions do not survive daemon or OS termination, and raw replay is not a full screen snapshot.

[Unreleased]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.5...dev
[0.0.5]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.1
