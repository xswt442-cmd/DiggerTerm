<p align="center"><img src="icons/diggerterm-icon.svg" alt="DiggerTerm Logo" width="104" height="104"></p>
<h1 align="center">DiggerTerm</h1>
<p align="center"><strong>轻量、专注的桌面终端工作区</strong></p>
<p align="center">多终端管理 · 后台会话保活 · SSH · 外部终端只读观察</p>
<p align="center">
  <a href="README.en.md">English</a> ·
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">下载</a> ·
  <a href="CHANGELOG.md">更新日志</a>
</p>
<p align="center">
  <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases"><img alt="Release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?label=release&amp;color=657483"></a>
  <img alt="Windows x64" src="https://img.shields.io/badge/Windows-x64-657483">
  <img alt="Linux x64" src="https://img.shields.io/badge/Linux-x64-657483">
  <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
</p>

---

> **v0.0.2 · Windows / Linux x64 预览版。** Windows 包尚未代码签名；Linux 主要使用 Ubuntu 24.04 构建和测试，其他发行版兼容性需要实机确认。

## 下载与安装

从 [v0.0.2 Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.2) 下载对应安装包：

| 平台 | 文件 | 用法 |
| --- | --- | --- |
| Windows x64 | `DiggerTerm-v0.0.2-windows-amd64.exe` | 双击运行；需要 Edge WebView2 Runtime |
| Linux x64 | `DiggerTerm-v0.0.2-linux-x86_64.AppImage` | `chmod +x *.AppImage` 后执行；可能需要系统图形依赖 |
| Ubuntu/Debian x64 | `DiggerTerm-v0.0.2-linux-amd64.deb` | `sudo apt install ./DiggerTerm-v0.0.2-linux-amd64.deb` |

Release 还提供统一的 `SHA256SUMS`。Linux 用户可在同一下载目录执行 `sha256sum -c SHA256SUMS`；Windows 用户可使用 PowerShell `Get-FileHash .\DiggerTerm-v0.0.2-windows-amd64.exe -Algorithm SHA256` 与校验值对照。

v0.0.1 Windows ZIP 仍保留在[历史 Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.1) 中；**v0.0.2 正式资产只有上述三个安装包及校验文件**。

## 主要功能

- **多会话终端**：Vue 3 + xterm.js 交互界面；侧栏切换、搜索、重命名，支持窗口和分栏拖动调整。
- **本地 Shell / WSL**：Windows PowerShell、CMD、已安装的 WSL 发行版，及 Linux 本地 Shell。
- **SSH**：通过系统 OpenSSH 创建交互式会话，支持 `~/.ssh/config` 中的简单 Host 别名与可选端口；指纹、密钥、ssh-agent、密码由 OpenSSH 处理，应用不保存 SSH 凭据。
- **后台 Daemon**：独立 Go 进程拥有 DiggerTerm 创建的 PTY/ConPTY；关闭窗口默认隐藏到托盘，选择退出 GUI 后仍可重新连接运行中的会话。
- **只读观察**：查看同一 Windows 用户的常见终端与 Agent 进程，以及**已运行 WSL 发行版**内默认 Linux 用户的部分相关进程元信息。不读取外部终端画面，不操作或终止外部进程。
- **可保存的设置**：四种低饱和暗色主题、界面字号（10–16 px）与终端字号（11–20 px）、字体、行距、光标、回滚、语言、默认 Shell 与本地工作目录、观察扫描间隔等。系统托盘菜单随界面语言切换。

### 终端快捷键

在 **设置 → 终端 → 快捷键模式**选择：

| 模式 | Ctrl+C | Ctrl+V |
| --- | --- | --- |
| **智能（默认）**：普通 Shell | 有选区复制，否则发送中断 | 粘贴剪贴板文本 |
| **智能**：nano / vim 等备用屏幕程序 | 交给终端程序处理 | 交给程序处理（nano 中翻页） |
| **终端原生** | 交给程序处理 | 交给程序处理 |
| **桌面复制粘贴** | 有选区复制，否则中断 | 始终粘贴 |

`Ctrl+Shift+C` / `Ctrl+Shift+V` 始终可复制/粘贴（macOS 另有 Cmd 行为，当前未发布 macOS 包）。`Ctrl+O/W/K/U/X` 等组合键可用于 nano；Ctrl+A/L/R 等控制键会传给 Shell。Windows ConPTY 中断与 Linux GNU nano 的真实 PTY 用例已通过自动化测试，实际 GUI 剪贴板仍建议在目标机器验证。

### 会话生命周期与边界

| 操作 | 结果 |
| --- | --- |
| 切换会话、关闭 GUI 窗口 | 终端继续运行；关闭窗口通常隐藏到托盘 |
| 托盘「退出界面」 | 退出 GUI，Daemon 与自有会话继续运行 |
| 再次启动 DiggerTerm | 连接同一系统用户仍在运行的 Daemon |
| 终止某个自有会话 | 仅显式终止所选的 DiggerTerm 托管进程 |
| 托盘「停止后台服务」 | **只有没有运行中会话才允许** |

```text
Wails 3 GUI + xterm.js  <-- 用户专属本地 IPC --> Go Session Daemon
                                                  └─ PTY / ConPTY / WSL / SSH
```

**已知边界：** Daemon 崩溃、用户注销、机器重启后**不保证会话存活**；有限输出回放不等于完整终端屏幕快照。SSH 断线或远端 Shell 退出也可能结束任务，长期远程任务建议使用 tmux。

外部观察只依赖进程名/父子关系等启发式信息，可能漏报；Windows/WSL PID 互不对应。扫描会复核 WSL 发行版是否运行，但检查与扫描存在极短竞态，不能绝对保证不唤醒已停止的发行版。应用**不会接管外部 PTY，也不会读取其命令行、环境变量或屏幕**。

SSH 高级选项（IPv6、ProxyJump、身份文件等）建议通过本地 `~/.ssh/config` 配置；Host 建议仅识别简单别名，不展开 `Include`。默认工作目录只作用于**新建的本地 Shell**，不改变现有终端，也不会将 Windows 路径传入 WSL/SSH。设置存储于 GUI WebView，不属于 Daemon 配置。

## 从源码构建

需要 **Go 1.25+**、**Node.js 24**（CI 当前使用）、[Wails 3 CLI v3.0.0-beta.29](https://v3.wails.io/)：

```shell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
npm --prefix frontend run test:preferences
npm --prefix frontend run test:shortcuts
npm --prefix frontend run build
go test ./...
go vet ./...
```

Windows 在项目根目录运行 `wails3 build`，生成 `bin/diggerterm.exe`。Linux（Ubuntu 24.04）需先安装 GTK4、WebKitGTK 6 等开发依赖（见 [发布工作流](.github/workflows/release.yml)），再执行 `CGO_ENABLED=1 go build -tags production -o bin/diggerterm .`。正式三格式打包与校验命令见 [RELEASE.md](RELEASE.md)。

自有会话退出后，可执行 `diggerterm --stop-daemon` 显式停止空闲 Daemon。

[English README](README.en.md) · [更新日志](CHANGELOG.md) · [发布流程（维护者）](RELEASE.md) · [MIT License](LICENSE)
