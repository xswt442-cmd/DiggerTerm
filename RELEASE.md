# DiggerTerm v0.0.1 · Initial Release / 首次发布

> **Early preview · Windows x64 only / 早期预览版 · 仅 Windows x64**

## 简体中文

DiggerTerm v0.0.1 是首次公开预览。它将 PowerShell、CMD 和默认 WSL 会话集中在一个轻量桌面窗口中，使用独立 Go Daemon 持有终端进程。

**主要功能**
- 左侧会话管理、右侧完整交互式终端。
- 创建、切换、调整大小与显式终止 DiggerTerm 管理的会话。
- 关闭或退出 GUI 后，只要 Daemon 仍运行，终端可以继续运行；重新启动 GUI 可重新连接。
- 系统托盘支持隐藏窗口、只退出 GUI，且活跃会话会阻止 Daemon 被主动停止。

**下载与使用**
- 附件：`DiggerTerm-v0.0.1-windows-amd64.zip`（Windows x64 便携版）。
- 解压后启动 `diggerterm.exe`，无需安装程序。
- 需要 Microsoft Edge WebView2 Runtime；WSL 会话需要已安装并初始化的 WSL 发行版。
- 本版本没有代码签名，可能显示 Windows 未知发布者提示。
- 可用同名 `.sha256` 校验文件验证下载内容。

**已知限制**
- 尚无 SSH、AI Agent 后台进程观察、外部终端接管。
- 只有 Daemon 持续运行时会话才有保障；Daemon 意外终止、用户注销或关机后不保证会话存活。
- 有限 PTY 输出重播不等于全屏终端程序的完整状态恢复。
- Linux/macOS 尚未提供正式测试的发布包。

## English

DiggerTerm v0.0.1 is the first public preview: a lightweight desktop workspace for PowerShell, Command Prompt, and default WSL sessions, backed by an independent Go session daemon.

**Highlights**
- A session sidebar and a real interactive terminal.
- Create, switch, resize, and explicitly terminate DiggerTerm-managed sessions.
- Close or exit the GUI without ending terminal processes, provided the daemon is still running; reconnect after restarting the GUI.
- System tray hide / GUI-only quit. Stopping the daemon is refused while sessions remain active.

**Download and run**
- Attachment: `DiggerTerm-v0.0.1-windows-amd64.zip` (portable Windows x64 archive).
- Extract and launch `diggerterm.exe`; no installer required.
- Requires Microsoft Edge WebView2 Runtime; WSL sessions require an installed, initialized Linux distribution.
- The preview is unsigned and may show an unknown-publisher warning.
- Verify your download using the accompanying `.sha256` checksum file.

**Known limitations**
- SSH, AI agent background-process observation, and external-terminal takeover are not available.
- Session continuity depends on a running daemon; daemon failure, sign-out, or shutdown can end sessions.
- Bounded PTY output replay is not a complete snapshot of a full-screen terminal program.
- No officially tested Linux/macOS release packages.

---

[Source](https://github.com/xswt442-cmd/DiggerTerm) · [Documentation / 文档](https://github.com/xswt442-cmd/DiggerTerm/blob/v0.0.1/README.md) · [MIT License](https://github.com/xswt442-cmd/DiggerTerm/blob/v0.0.1/LICENSE)
