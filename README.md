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
    <a href="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/xswt442-cmd/DiggerTerm/actions/workflows/ci.yml/badge.svg?branch=main"></a>
    <a href="https://github.com/xswt442-cmd/DiggerTerm/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/xswt442-cmd/DiggerTerm?color=657483"></a>
    <a href="LICENSE"><img alt="MIT" src="https://img.shields.io/badge/license-MIT-657483"></a>
  </p>
</div>

## 下载

从 [GitHub Releases](https://github.com/xswt442-cmd/DiggerTerm/releases/latest) 下载适用于 Windows x64、Linux x64 或 Ubuntu / Debian x64 的安装包和 SHA-256 校验文件。

## 功能

- **多终端管理**：侧栏切换、搜索、重命名；支持左右/上下嵌套分屏、拖动比例与布局恢复。关闭面板保留后台会话。
- **本地与远程 Shell**：PowerShell、CMD、Windows Git Bash（需安装 Git for Windows）、WSL、Linux Bash 和 SSH。
- **快速切换目录**：选中托管终端后点击“快速 cd”或按 `Ctrl+Shift+G`，本地会话支持原生文件夹选择器，SSH 会话支持独立的只读远程目录浏览器，也可手动输入路径。按 Shell 类型安全引用 `cd` 命令，点击“快速 CD”后发送并执行；本地路径可设为新会话默认启动目录。
- **工作区信息**：快速 CD 可收藏目录并记录最近发送的目录；会话信息可添加备注和颜色。会话预设保存本地 Shell、启动目录或 SSH 主机与端口，方便再次创建。
- **后台会话**：退出 GUI 后，Daemon 继续托管会话；重新打开可连接。
- **Dig 进程树**：按 Windows/WSL 及发行版查看已发现的 Shell、Agent 与后代，支持展开、搜索和筛选；详情只读，不接管或控制外部进程。
- **个性化**：暗色配色、字号、字体、语言、默认 Shell 和工作目录等设置。

智能模式的 **Ctrl+C** 有选区复制、无选区中断；nano 等全屏应用保留原生控制键。终端工具栏和右键入口提供复制、粘贴和“复制模式”；鼠标报告模式下可尝试 **Shift 拖选**（macOS 用 Option），无效时使用复制模式。在“设置 → 快捷键”录制、校验和保存到当前用户主目录下的 `DiggerTerm/keybindings.json`（[示例](examples/keybindings.json)）。

## 说明

支持用户主题：将 [示例 JSON/CSS](examples/themes/) 复制到当前用户主目录下的 `DiggerTerm/`，在“设置 → 外观”刷新并选择。外观页显示实际目录，并可直接打开该目录。主题名称读取 JSON 的 `name`；`custom.css` 提供可关闭的全局覆盖。快捷键配置也保存在该目录的 `keybindings.json` 中。

Windows 安装程序为当前用户创建开始菜单快捷方式及资源管理器文件夹右键菜单。选择“Open in DiggerTerm”会打开或唤起 DiggerTerm，并在所选文件夹中创建 PowerShell 会话；Windows 11 的菜单项可能位于“显示更多选项”。卸载会移除这些入口并保留用户数据。更新或卸载前请先结束已有会话并正常退出 Daemon。

Windows 可从开始菜单的 “DiggerTerm (Administrator)” 或文件夹右键的 “Open in DiggerTerm as administrator” 按需启动管理员窗口，并在 Windows UAC 确认后创建管理员权限的本地终端。管理员窗口和普通窗口使用各自独立的 Daemon 与会话；管理员权限不会自动提升 SSH 远端用户或 WSL 内的 Linux 用户。

Windows 需要 WebView2，安装包尚未签名。Linux 安装包基于 Ubuntu 24.04 构建，其他发行版可能需要相应的 GTK4、WebKitGTK 6 等系统依赖。会话不保证在 Daemon 崩溃或系统重启后存活。

快速 CD 会向托管终端发送经过引用的命令和回车。请确保终端处于空白 Shell 提示符；在 `vim`、Agent 或其他前台程序运行时不要使用，提示符中已有文字时也不要使用。Dig 的只读外部进程不支持该操作。

目录收藏和历史、会话备注与颜色、预设保存在本机 GUI 中。最近目录表示已发送切换命令，并不证明 Shell 切换成功；SSH 目录按托管会话隔离。SSH 预设不保存密码，也不自动切换远端目录；WSL 预设不设置 Windows 启动目录。

本机文件夹选择器会把 Windows 盘符目录转换为 WSL 默认的 `/mnt/c/...` 或 Git Bash 的 `/c/...` 路径；自定义 WSL 挂载点与 Linux 原生目录仍需手动输入。SSH 目录浏览器通过独立 OpenSSH 连接读取远端目录，需要 POSIX 兼容 Shell 和免交互密钥/SSH Agent 认证，并沿用现有 `ssh_config` 与主机密钥校验。仅使用交互密码登录或非 POSIX Shell 的 SSH 会话仍可手动输入路径（可选择相应 Shell 语法）。

会话设置提供手动重连与退出诊断。运行中会话仅允许一个 GUI 控制；其他 GUI 可以查看，但不能终止已由另一个 GUI 控制的会话。已被终端接收的输入会先于视图分离排队发送。IPC 请求有超时限制；若创建或终止请求超时，操作结果可能仍在后台生效，请先刷新会话列表再重试。

新功能需要配套版本的 Daemon；更新时，先完成已有会话，再停止旧 Daemon 并打开新版应用。重新连接时会恢复最近的终端输出，长时间运行的全屏界面可能不完整。

## 许可证

MIT
