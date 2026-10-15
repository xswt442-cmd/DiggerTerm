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

[v0.0.2 · GitHub Release](https://github.com/xswt442-cmd/DiggerTerm/releases/tag/v0.0.2)

| 平台 | 安装包 |
| --- | --- |
| Windows x64 | [`.exe`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-windows-amd64.exe) |
| Linux x64 | [`.AppImage`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-linux-x86_64.AppImage) |
| Ubuntu / Debian x64 | [`.deb`](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/DiggerTerm-v0.0.2-linux-amd64.deb) |

下载校验：[SHA256SUMS](https://github.com/xswt442-cmd/DiggerTerm/releases/download/v0.0.2/SHA256SUMS)。

## 功能

- **多终端管理**：侧栏切换、搜索、重命名，可拖动分栏。
- **本地与远程 Shell**：PowerShell、CMD、WSL、Linux Shell 和 SSH。
- **后台会话**：退出 GUI 后，Daemon 继续托管会话；重新打开可连接。
- **只读观察**：查看外部终端与 Agent 进程信息，不接管、不控制。
- **个性化**：暗色配色、字号、字体、语言、默认 Shell 和工作目录等设置。

快捷键遵循 Windows 终端习惯：**Ctrl+C** 有选区复制、无选区中断；**Ctrl+V** 粘贴。nano 等全屏应用保留原生控制键；**Ctrl+Shift+C/V** 可随时复制/粘贴。

## 说明

开发分支支持用户主题：将 [示例 JSON/CSS](examples/themes/) 复制到用户主目录下的 `user/DiggerTerm/`，在“设置 → 外观”刷新并选择。主题名称读取 JSON 的 `name`；`custom.css` 提供可关闭的全局覆盖。外观页显示实际目录和配置错误。

Windows 需要 WebView2，安装包尚未签名。Linux 主要针对 Ubuntu 24.04 构建，其他发行版兼容性待验证。会话不保证在 Daemon 崩溃或系统重启后存活。

[English](README.en.md) · [CHANGELOG](CHANGELOG.md) · [发布流程](RELEASE.md) · [MIT License](LICENSE)
