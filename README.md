# DiggerTerm

[简体中文](#简体中文) · [English](#english)

一个轻量、开源的桌面终端会话管理器。**左侧会话栏、右侧真实终端**，后台由独立的 Go Daemon 持有终端进程，GUI 可以退出后再次连接。

A lightweight open-source desktop terminal workspace, with a **session sidebar and an interactive terminal**. A separate Go daemon owns terminal processes so the desktop UI can exit and reconnect.

> **v0.0.1 · Early release** — Windows x64 is the only published and tested distribution. Features listed under “Planned” are not implemented.

## 简体中文

### 功能

- 在 Windows 上创建和切换 **PowerShell、CMD 和默认 WSL** 交互式终端。
- 使用 xterm.js 显示终端输出，支持输入、窗口尺寸调整、显式终止以及移除已退出会话。
- **GUI / Daemon 分离**：GUI 退出（包括意外结束）后，若 Daemon 仍运行，原有会话继续存活；重新打开 GUI 可连接原会话。
- 关闭主窗口默认隐藏到系统托盘；托盘可选择仅退出 GUI，或在没有运行中会话时停止 Daemon。
- 本地 IPC 不暴露 HTTP 端口。Windows 使用当前用户专属的 Named Pipe；Unix 平台有私有 Socket 实现，但尚未经过桌面实机验证。

### 下载与运行

从 [Releases](https://github.com/xswt442-cmd/DiggerTerm/releases) 下载 `DiggerTerm-v0.0.1-windows-amd64.zip`，解压到可写目录后运行 `diggerterm.exe`。无需安装器；需要 Windows x64 和可用的 **Microsoft Edge WebView2 Runtime**。使用 WSL 功能前，请自行安装和初始化 WSL 发行版。

首次启动 GUI 时会自动连接或创建同一用户的后台 Daemon。请勿在有正在运行的终端时删除或替换正在使用的程序文件。此版本**尚未代码签名**，Windows 可能显示未知发布者提示。

| 操作 | 行为 |
| --- | --- |
| 切换会话 / 关闭主窗口 | 不结束终端；关闭主窗口只隐藏到托盘 |
| 托盘：Quit UI | 完全退出 GUI，Daemon 和其会话继续运行 |
| 在侧栏显式 Terminate | 终止选中的、由 DiggerTerm 创建的会话 |
| 托盘：Stop daemon and quit | 仅当没有运行中的会话时才停止 Daemon |
| 再次运行程序 | 重新连接同一用户仍在运行的 Daemon |

**已知限制：** 不支持 SSH、外部终端发现、AI Agent 子进程观察或直接接管任意现有终端。Daemon 崩溃、被强制结束、用户注销或关机时不保证会话继续运行；历史输出仅保留有限字节流，并非完整终端屏幕快照。Linux/macOS 未提供正式安装包。

### 开发

依赖 Go 1.25+、Node.js 22+ 和 Wails 3 CLI（本项目采用 `v3.0.0-beta.29`），在 Windows 上执行：

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
wails3 dev
```

验证并构建：

```powershell
go test ./...
go vet ./...
npm --prefix frontend run build
wails3 build
```

构建产物为 `bin/diggerterm.exe`。只在所有会话结束后，如需显式停服，可运行 `bin/diggerterm.exe --stop-daemon`。可选的 `DIGGERTERM_IPC_NAMESPACE` 用于隔离测试实例。

架构：`Wails GUI ↔ 本地私有 IPC ↔ DiggerTerm --daemon ↔ PTY / ConPTY / WSL`。只有 Daemon 拥有 PTY；外部进程今后即使被发现也默认只读，不能未经逐次授权就被控制。

## English

### Features

- Create and switch interactive **PowerShell, Command Prompt, and default WSL** sessions on Windows.
- Render full terminal I/O through xterm.js, resize terminals, explicitly terminate managed sessions, and remove completed sessions.
- **Separate GUI and daemon**: sessions remain alive while the daemon lives, even when the GUI exits or is force-closed. Opening the GUI again reconnects to that same daemon.
- Closing the main window hides it in the system tray. The tray offers **Quit UI**, or a daemon shutdown option that refuses to run if any session is active.
- Local IPC does not expose a network HTTP port. Windows uses a named pipe restricted to the current OS user. Unix socket code exists but has not received complete desktop testing.

### Download and run

Download `DiggerTerm-v0.0.1-windows-amd64.zip` from [GitHub Releases](https://github.com/xswt442-cmd/DiggerTerm/releases), extract it to a writable directory, and launch `diggerterm.exe`. No installer is needed. You need **Windows x64 and Microsoft Edge WebView2 Runtime**; WSL usage requires an installed and initialized distribution.

The GUI connects to or launches a per-user session daemon automatically. Avoid replacing or deleting the executable while its daemon is running. **This preview is unsigned**, so Windows may display an unknown-publisher warning.

| Action | Behavior |
| --- | --- |
| Switch sessions / close window | Does not end sessions; closing the window hides it in the tray |
| Tray: Quit UI | Exits only the GUI; the daemon and its sessions remain |
| Explicit Terminate in sidebar | Ends the selected DiggerTerm-managed session |
| Tray: Stop daemon and quit | Refuses to stop while any session is running |
| Launch GUI again | Reconnects to the same running daemon for the current user |

**Known limitations:** SSH, external terminal discovery, AI agent process monitoring, and arbitrary terminal takeover are not implemented. Sessions are not guaranteed to survive daemon failure, forced daemon termination, OS sign-out, or reboot. Limited raw-output replay is **not** a complete terminal-screen snapshot. There are no tested Linux/macOS release packages yet.

### Build from source

Install Go 1.25+, Node.js 22+, and Wails 3 CLI (`v3.0.0-beta.29` for this project):

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
wails3 dev
```

Tests and production build:

```powershell
go test ./...
go vet ./...
npm --prefix frontend run build
wails3 build
```

The build generates `bin/diggerterm.exe`. Use `bin/diggerterm.exe --stop-daemon` only when all sessions have finished. The optional `DIGGERTERM_IPC_NAMESPACE` environment variable isolates developer/test instances.

Architecture: `Wails GUI ↔ private local IPC ↔ DiggerTerm --daemon ↔ PTY / ConPTY / WSL`. Only the daemon owns terminal processes; any future external-process observer will remain read-only by default.

---

[Changelog](CHANGELOG.md) · [v0.0.1 Release Notes](RELEASE.md) · [MIT License](LICENSE)
