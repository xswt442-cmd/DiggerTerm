# Changelog / 更新日志

本项目的版本变更记录以 [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) 为参考，按语义化版本标识。
Changes are recorded in a format inspired by [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased] / 未发布

- Dig 新增 Windows/WSL 只读进程树，按名称标注终端和 Agent、展示父链推断，并支持搜索、筛选和元数据详情。
- Dig adds a read-only Windows/WSL process tree with name-based terminal/agent labels, parent-chain inference, search, filters, and metadata details.
- Dig 搜索会临时展开命中节点的祖先，清除筛选后恢复手动折叠状态。
- Dig search temporarily expands matching ancestors and restores manual folds when filters clear.
- 托盘菜单区分“停止空闲后台服务并退出”和“停止所有会话并退出”；批量终止会先确认会话数量，执行前重新核对列表，只终止 Daemon 管理的会话，并等待退出后再停止 Daemon。
- Tray actions distinguish stopping an idle daemon from explicitly terminating managed sessions. Bulk termination confirms the session count, revalidates the list, and stops the daemon only after the confirmed managed sessions exit.
- 慢速只读进程扫描不再阻塞同一 GUI 连接的会话请求；并发扫描受限，并会在客户端断开时取消。
- Slow read-only process scans no longer block session requests on the same GUI connection; scans are concurrency-limited and canceled when the client disconnects.

## [0.0.4] - 2026-10-16

### 简体中文

- 设置新增帮助页，列出主题 JSON、主题 CSS、全局 `custom.css` 和快捷键配置的实际路径，并提供版本检查。
- 快捷键配置支持逐项恢复默认值；快捷键模式移入快捷键设置。
- 优化首页空状态与托盘退出菜单；无会话时分屏后隐藏欢迎层。
- 主题与快捷键配置统一保存在当前用户主目录下的 `DiggerTerm/`。

### English

- Added a Help page with paths for theme JSON/CSS, global `custom.css`, and keybindings, plus update checks.
- Added per-shortcut default reset and moved keyboard mode into shortcut settings.
- Refined the home empty state and tray exit labels; the welcome panel hides after splitting an empty workspace.
- Theme and shortcut configuration now share `DiggerTerm/` under the current user's home directory.

## [0.0.3]

### 简体中文

- 新增用户 JSON/CSS 配色主题、外观刷新，以及快捷键录制、冲突检测和保存。
- 新增可嵌套终端分屏和布局恢复，限制每个托管会话只有一个 GUI 控制端；关闭面板仅分离视图。
- 改进全屏程序选区复制，新增工具栏、右键操作和只读复制模式；Windows 原生剪贴板增加后备处理。
- 新增手动 Daemon 重连、会话退出码和 PTY 流诊断、输出缺口检测与回放截断提示。
- Windows 新增 Git Bash，本机未安装时不会转到 WSL；Linux/macOS 使用本机 Bash。

### English

- Added user JSON/CSS themes, appearance refresh, shortcut recording, conflict checks, and persistence.
- Added nested terminal panes and layout restore with one GUI controller per managed session; closing a pane only detaches its view.
- Improved selection copy for full-screen applications, added toolbar/context-menu controls and read-only copy mode, and added a Windows native clipboard fallback.
- Added manual daemon reconnection, session exit/PTY stream diagnostics, output gap detection, and bounded replay warnings.
- Added Windows Git Bash without falling back to WSL when it is unavailable, plus native Bash on Unix.

画面恢复仍受限于最近 256 KiB 原始输出，不能保证还原完整的全屏终端状态。用户报告的 SSH 闲置断联尚未在现场复现。Windows 原生 WebView2 跨应用剪贴板和部分平台 GUI 矩阵仍未完成验收。

Recovery still uses the latest 256 KiB of raw output and cannot reproduce every full-screen terminal state. The reported idle SSH disconnect still needs field reproduction. Some native WebView2 clipboard and platform GUI checks remain unverified.

**下载 / Downloads**（Windows 未签名；Linux 主要针对 Ubuntu 24.04）:

- [Windows x64 EXE](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.3/DiggerTerm-v0.0.3-windows-amd64.exe)
- [Linux x86_64 AppImage](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.3/DiggerTerm-v0.0.3-linux-x86_64.AppImage)
- [Ubuntu/Debian amd64 DEB](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.3/DiggerTerm-v0.0.3-linux-amd64.deb)
- [SHA256SUMS](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.3/SHA256SUMS)

## [0.0.2] - 2026-10-15

### 简体中文
- Windows / WSL：新增同用户外部终端与 Agent 进程的只读元信息观察。扫描仅覆盖已运行的 WSL 发行版；没有外部 PTY 画面读取、输入或终止权限。
- SSH：通过系统 OpenSSH 创建 Daemon 托管的远程交互会话，支持 ssh_config 别名、可选端口、原生密钥/ssh-agent/主机指纹和口令交互；DiggerTerm 不存储认证信息。
- Windows 体验：支持托管会话重命名（Daemon 持有名称、跨 GUI 同步）、侧栏搜索过滤、从本地 OpenSSH 配置提供简单 Host 别名建议。
- 界面改进：紧凑侧栏、可拖动和键盘调节的左右/上下分栏、真实 Logo；四标签紧凑设置面板，支持暗色配色、独立的界面字号及终端字号、字体系列/行距、回滚行数、光标与滚动行为、默认 Shell/工作路径、新会话激活，以及只读观察显示与刷新选项，均可本地保存。松影配色调整为低饱和深灰绿，原生托盘菜单随界面语言切换。
- 终端输入：新增智能 / 终端原生 / 桌面剪贴板快捷键模式，修复 Ctrl+C 中断与复制、Ctrl+V 粘贴及 nano 原生 Ctrl+V 翻页冲突，支持 Ctrl+Shift+C/V、Ctrl+Insert、Shift+Insert；修复 Windows 后台 ConPTY 前台程序不能响应 Ctrl+C 的进程组问题。补充 Windows ping 与 Linux nano 真 PTY 回归测试。

### English
- Windows / WSL: added read-only metadata observation for same-user terminal/agent processes, including Linux processes in already-running WSL distributions. No external terminal screen capture or process control.
- SSH: added daemon-managed remote interactive sessions via system OpenSSH, with ssh_config host aliases, optional port, and native host-key/key-agent/password prompts. DiggerTerm does not persist credentials.
- Windows UX: rename managed sessions (daemon-owned, synchronized across GUIs), filter sidebar sessions/processes, and suggest literal OpenSSH Host aliases from the local config.
- UI: denser sidebar, draggable and keyboard-adjustable splitters, project logo, and a compact four-tab settings panel with saved dark palettes, separate UI/terminal font sizes, font family and line spacing, scrollback, cursor/scroll options, default shell and directory, new-session focus, and read-only observation display/refresh controls. Pine now uses a muted grey-green palette, and the native tray menu follows the UI language.
- Terminal input: added Smart / Terminal native / Desktop clipboard modes to resolve Ctrl+C interrupt versus copy and Ctrl+V paste versus nano page-down, with Ctrl+Shift+C/V and Insert clipboard alternatives. Fixed a Windows detached-daemon process-group issue that prevented ConPTY foreground commands from receiving Ctrl+C. Added real Windows ping and Linux GNU nano PTY regressions.

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

[Unreleased]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.4...dev
[0.0.4]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/xswt442-cmd/DiggerTerm/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.1
