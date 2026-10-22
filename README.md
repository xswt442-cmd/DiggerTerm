<div align="center">
  <img src="icons/diggerterm-icon.svg" alt="DiggerTerm" width="88" height="88">
  <h1>DiggerTerm</h1>
  <p>简洁的桌面多终端工作区</p>
  <p>
    <a href="README.en.md">English</a> ·
    <a href="https://github.com/xswt442-cmd/DiggerTerm/releases">Releases</a> ·
    <a href="CHANGELOG.md">更新日志</a>
  </p>
  <p>
    <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/check.yml/badge.svg?branch=main"></a>
    <a href="https://github.com/xswt442-cmd/DiggerTerm/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?color=657483"></a>
    <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
  </p>
</div>

## 下载

[v0.0.6 · GitHub Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.6)

| 平台 | 安装包 |
| --- | --- |
| Windows x64 | [`.exe`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.6/DiggerTerm-v0.0.6-windows-amd64.exe) |
| Linux x64 | [`.AppImage`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.6/DiggerTerm-v0.0.6-linux-x86_64.AppImage) |
| Ubuntu / Debian x64 | [`.deb`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.6/DiggerTerm-v0.0.6-linux-amd64.deb) |

下载校验：[SHA256SUMS](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.6/SHA256SUMS)。

## 功能

- **多终端管理**：侧栏切换、搜索、重命名；支持左右/上下嵌套分屏、拖动比例与布局恢复。关闭面板保留后台会话。
- **本地与远程 Shell**：PowerShell、CMD、Windows Git Bash（需安装 Git for Windows）、WSL、Linux Bash 和 SSH。
- **后台会话**：退出 GUI 后，Daemon 继续托管会话；重新打开可连接。
- **Dig 进程树**：按 Windows/WSL 及发行版查看已发现的 Shell、Agent 与后代，支持展开、搜索和筛选；详情只读，不接管或控制外部进程。
- **个性化**：暗色配色、字号、字体、语言、默认 Shell 和工作目录等设置。

智能模式的 **Ctrl+C** 有选区复制、无选区中断；nano 等全屏应用保留原生控制键。终端工具栏和右键入口提供复制、粘贴和“复制模式”；鼠标报告模式下可尝试 **Shift 拖选**（macOS 用 Option），无效时使用复制模式。在“设置 → 快捷键”录制、校验和保存到当前用户主目录下的 `DiggerTerm/keybindings.json`（[示例](examples/keybindings.json)）。

## 说明

支持用户主题：将 [示例 JSON/CSS](examples/themes/) 复制到当前用户主目录下的 `DiggerTerm/`，在“设置 → 外观”刷新并选择。外观页显示实际目录，并可直接打开该目录。主题名称读取 JSON 的 `name`；`custom.css` 提供可关闭的全局覆盖。快捷键配置也保存在该目录的 `keybindings.json` 中。

Windows 需要 WebView2，安装包尚未签名。Linux 安装包基于 Ubuntu 24.04 构建，其他发行版可能需要相应的 GTK4、WebKitGTK 6 等系统依赖。会话不保证在 Daemon 崩溃或系统重启后存活。

会话设置提供手动重连与退出诊断。运行中会话仅允许一个 GUI 控制；其他 GUI 可以查看，但不能终止已由另一个 GUI 控制的会话。已被终端接收的输入会先于视图分离排队发送。IPC 请求有超时限制；若创建或终止请求超时，操作结果可能仍在后台生效，请先刷新会话列表再重试。

新功能需要配套版本的 Daemon；更新时，先完成已有会话，再停止旧 Daemon 并打开新版应用。重新连接时会恢复最近的终端输出，长时间运行的全屏界面可能不完整。

## 许可证

MIT
