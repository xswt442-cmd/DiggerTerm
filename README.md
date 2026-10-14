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

### 界面外观与默认目录（仅 dev）

左侧栏与终端之间、**我的会话**与**外部观察**之间的分割线都可以拖动；也可以聚焦后用方向键微调。分栏比例会随界面偏好保存。

左下角 **设置** 使用约 304 px 的紧凑弹层，分为**外观 / 终端 / 会话 / 观察**四组。外观可选四种低饱和暗色主题（夜航、石墨、松影、暮蓝，其中松影已调整为更淡的深灰绿）、**独立的界面字号 10–16 px**、中英界面和分栏复位；系统托盘菜单会跟随当前界面语言同步切换；终端可配置字体（系统未安装时回退）、字号 11–20 px、行距、回滚行数、光标样式与闪烁、滚轮速度、右键选词和终端留白；会话可指定默认 Shell 类型、是否自动切换到新会话，以及**新建本地 Shell 的默认工作目录**；观察可关闭外部进程显示、设置手动/自动扫描间隔、切换 PID 和底部状态栏显示。

工作目录留空沿用原来的默认路径；填写现存的绝对路径后，只对后来创建的 PowerShell、CMD 或 Unix 本地 Shell 生效，不改变已运行的终端，也不把 Windows 路径传给 WSL 或远程 SSH。路径由 Daemon 验证，非法或不存在的目录会在新建时提示错误。隐藏外部观察会停止当前 GUI 的扫描轮询，**不会控制或停止任何外部进程**。滚动回放行数只决定当前 GUI 终端的历史缓存，并不扩大 Daemon 原有的输出回放上限。

这些设置保存在当前桌面 WebView 的本地存储中，并非 Session Daemon 配置；不会改变会话的所有权或授权。配色/字号切换时会自动重新计算当前终端的行列数。

**终端快捷键：**「设置 → 终端 → 快捷键模式」默认**智能**：普通 Shell 中 Ctrl+C 有选区则复制、无选区则中断；Ctrl+V 粘贴。nano/vim 等启用终端备用屏幕的全屏程序中，Ctrl+C、Ctrl+V 等会交还程序（例如 nano 的 Ctrl+V 翻页、Ctrl+O 保存、Ctrl+X 退出）；Ctrl+Shift+C / Ctrl+Shift+V 始终复制/粘贴。可改为「终端原生」（始终优先转交 Ctrl 组合键）或「桌面复制粘贴」（始终采用桌面剪贴板规则）。Ctrl+A/L/R/W 等控制键发送到 Shell，而非触发浏览器的全选、刷新或关闭。Windows ConPTY 已验证前台持续 ping 可以被 Ctrl+C 中断；WSL Ubuntu 的 GNU nano 已验证编辑、保存、搜索、剪切恢复、翻页和退出。

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
