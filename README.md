<p align="center"><img src="icons/diggerterm-icon.svg" alt="DiggerTerm Logo" width="104" height="104"></p>
<h1 align="center">DiggerTerm</h1>
<p align="center"><strong>轻量、专注的桌面终端工作区</strong></p>
<p align="center">一个窗口管理多个终端；独立 Daemon 保持会话运行。</p>
<p align="center">
  <a href="README.en.md">English</a> ·
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">下载 Releases</a> ·
  <a href="CHANGELOG.md">更新日志</a>
</p>
<p align="center">
  <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
  <a href="https://github.com/xswt442-cmd/DiggerTerm/releases"><img alt="Release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?label=release&amp;color=657483"></a>
  <img alt="Windows x64" src="https://img.shields.io/badge/Windows-x64-657483">
  <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
</p>

---

> **v0.0.1 · Windows x64 预览版**
> 目前只正式提供 Windows 便携包；Linux/macOS 的 GUI 尚未完成实机验证。

## 功能

- **多终端工作区**：左侧会话列表、右侧 xterm.js 交互终端，支持输入、尺寸调整、会话切换。
- **本地 Shell**：Windows PowerShell、CMD，以及已安装的默认 WSL 发行版。
- **后台会话保活**：独立 Go Daemon 拥有 PTY；关闭或退出 GUI 不会自动终止会话，重新打开可接回仍在运行的 Daemon。
- **明确的终止权限**：仅显式终止由 DiggerTerm 管理的会话；存在运行中会话时拒绝停止 Daemon。

## 下载

前往 [GitHub Releases](https://github.com/xswt442-cmd/DiggerTerm/releases/latest) 下载 `DiggerTerm-v0.0.1-windows-amd64.zip`，解压后运行 `diggerterm.exe`。

运行要求：**Windows x64**、可用的 **Microsoft Edge WebView2 Runtime**。如需使用 WSL，请先安装并初始化发行版。

> 首发预览版未进行代码签名，Windows 可能提示未知发布者。

## 会话生命周期

| 操作 | 结果 |
| --- | --- |
| 切换会话 / 关闭窗口 | 会话继续运行；关闭窗口默认隐藏到托盘 |
| 托盘 **Quit UI** | 只退出 GUI，Daemon 继续运行 |
| 再次打开应用 | 重新连接到同一用户仍在运行的 Daemon |
| 侧栏 **Terminate** | 明确终止对应的自有会话 |
| 托盘 **Stop daemon and quit** | 仅在没有活跃会话时允许停止 |

架构保持简单：

```text
Wails GUI + xterm.js  <-- 本地用户专属 IPC -->  Go Session Daemon
                                                   └─ PTY / ConPTY / WSL
```

**边界：** Daemon 崩溃、被强制停止、注销或关机时，不保证终端存活；有限输出回放不等于完整 VT 屏幕快照。任意外部终端画面接管仍未实现。

**dev 开发预览（不包含在 v0.0.1 下载包）：** 侧栏可以**只读观察**当前 Windows 用户的常见终端 / Agent 进程，以及**已经运行的 WSL 发行版**内默认 Linux 用户的终端 / Agent 相关进程（PID、父 PID、进程名、启动时间）。WSL 的 Linux PID 和 Windows PID 属于不同命名空间；识别仍是进程名和父子树推断，不能保证发现所有隐式终端。不会为了扫描唤醒停止的 WSL 发行版，也不会读取命令行、环境变量或外部终端画面；外部进程不可控制。WSL 内部进程暂不能与自有 `wsl.exe` 会话可靠关联。升级后若仍连接旧 Daemon，请在自有会话结束后明确停服，再重新启动 GUI。

### SSH（仅 dev，尚未包含在 v0.0.1 发布包）

在左侧选择 **SSH**，填写 `prod`（`~/.ssh/config` 中的 Host 别名）或 `alice@server.example`，端口可以留空以采用 OpenSSH 配置，然后点击 **Connect**。输入框会建议配置文件中简单、明确的 `Host` 别名（不会读取或显示密钥、密码，也不展开 `Include`）。首次连接的主机指纹提示、私钥/ssh-agent 与密码交互均由系统 OpenSSH 在终端内处理，DiggerTerm 不自动接受主机密钥，也不保存凭据。IPv6 地址、ProxyJump、密钥路径等高级配置请放进 `~/.ssh/config`。

侧栏还支持按名称/PID 筛选会话和只读观察进程；自有会话右侧的铅笔按钮可重命名。名称属于 Daemon 会话元信息，跨 GUI 重连保持一致，但 Daemon 停止后不持久化。**外部观察进程没有重命名或控制权限。**

SSH 属于 DiggerTerm 自有会话：只退出 GUI 不断开 SSH；但 **SSH 网络断线、远端 Shell 退出或 Daemon 停止并不能保活远端任务**，长任务建议在服务器侧使用 tmux。SSH 只支持交互终端，不是对外部已存在 SSH 终端的接管。

## 从源码构建

需要 Go 1.25+、Node.js 22+、[Wails 3 CLI](https://v3.wails.io/)（项目当前使用 `v3.0.0-beta.29`）。

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
npm --prefix frontend ci
wails3 dev
```

构建与检查（前端先构建，因为 Go 会嵌入 `frontend/dist`）：

```powershell
npm --prefix frontend run build
go test ./...
go vet ./...
wails3 build
```

生成的程序位于 `bin/diggerterm.exe`。所有终端结束后，如需停服可执行 `bin/diggerterm.exe --stop-daemon`。

## 项目

[English README](README.en.md) · [更新日志 / Changelog](CHANGELOG.md) · [MIT License](LICENSE)

## LICENSE

[MIT](LICENSE)
