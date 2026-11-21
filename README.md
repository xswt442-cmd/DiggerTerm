<div align="center">
  <img src="icons/diggerterm-icon.svg" alt="DiggerTerm" width="88" height="88">
  <h1>DiggerTerm</h1>
  <p>桌面多终端工作区，会话由后台 Daemon 托管</p>
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

从 [GitHub Releases](https://github.com/xswt442-cmd/DiggerTerm/releases/latest) 下载 Windows x64 的 `.exe` 安装包、Linux x64 的 `.AppImage`、Ubuntu / Debian x64 的 `.deb`，以及 SHA-256 校验文件。

## 功能

- **多终端管理**：侧栏列出全部终端，可以切换、搜索和重命名，支持左右或上下嵌套分屏，可以拖动调整比例并恢复布局。关闭面板后会话留在后台。
- **专注模式**：隐藏工作区控件，只显示当前终端，右键菜单、鼠标报告、键盘输入和点击链接都交给终端处理，DiggerTerm 不再拦截。分屏、命令面板、快速 CD 和复制覆盖层此时不可用，窄侧栏只保留退出入口和终端外观设置。
- **本地与远程 Shell**：PowerShell、CMD、Windows Git Bash（需安装 Git for Windows）、WSL、Linux Bash 和 SSH。
- **快速切换目录**：选中托管终端后点击“快速 CD”或按 `Ctrl+Shift+G`，本地会话使用原生文件夹选择器，SSH 会话使用独立的只读远程目录浏览器，两者都能手动输入路径。生成 `cd` 命令时按目标 Shell 类型转义路径再交给托管终端执行，本地路径也可以设为新会话的默认启动目录。
- **工作区信息**：快速 CD 可以收藏目录，并记录最近发送过的目录。可以给会话加备注和颜色，会话预设保存本地 Shell、启动目录，或 SSH 主机与端口。
- **后台会话**：退出 GUI 后 Daemon 继续托管会话，重新打开 GUI 会连接到原来的会话。
- **Dig 进程树**：按 Windows/WSL 及其发行版查看已发现的 Shell、Agent 和它们的子进程，可以展开、搜索和筛选。详情只读，不能接管或控制外部进程。
- **个性化**：可以设置暗色配色、字号、字体、语言、默认 Shell 和默认工作目录。
- **更新检测**：启动后每 24 小时最多在后台检查一次 GitHub 最新稳定版本，可以在帮助设置里关闭。发现新版本后显示下载入口。

智能模式下，**Ctrl+C** 在有选区时复制，没有选区时中断，nano 等全屏应用保留原生控制键。终端工具栏和右键菜单提供复制、粘贴和“复制模式”。启用了鼠标报告的终端，按住 **Shift** 拖动可以选中文本（macOS 用 Option）。仍然选不中时，改用“复制模式”。自定义快捷键在“设置 → 快捷键”页面录制，保存在 `DiggerTerm/keybindings.json`（[示例](examples/keybindings.json)）。

## 说明

主题和快捷键：把[示例 JSON/CSS](examples/themes/) 复制到当前用户主目录下的 `DiggerTerm/`，在“设置 → 外观”点击刷新，然后选择主题。JSON 里的 `name` 是主题显示名，`custom.css` 用于全局覆盖样式，快捷键保存在同一目录的 `keybindings.json`。

Windows 安装包会为当前用户创建开始菜单快捷方式和资源管理器右键菜单，Windows 11 上可能收在“显示更多选项”里。在文件夹上右键选择 “Open in DiggerTerm”，会在该目录打开一个 PowerShell 会话。卸载时会移除这些入口，但保留用户数据。开始菜单的 “DiggerTerm (Administrator)” 打开管理员窗口，它和普通窗口各自使用独立的 Daemon 和会话，不会提升 SSH 远端或 WSL 内的权限。Windows 需要 WebView2，安装包尚未签名。

Linux 安装包基于 Ubuntu 24.04 构建，其他发行版需要自行安装 GTK4、WebKitGTK 6 等依赖。会话不保证在 Daemon 崩溃或系统重启后存活。

快速 CD 会在托管终端里发送一条 `cd` 命令并按下回车。使用前确认提示符为空：`vim`、Agent 等前台程序正在运行时，或提示符里已有输入时，不要使用快速 CD。Dig 观察到的外部进程是只读的，没有这个入口。

收藏、目录历史、会话备注与颜色、预设只保存在本机 GUI 中。最近目录只表示命令已经发出，不代表 Shell 已经切换成功。SSH 目录按会话隔离，SSH 预设不存密码，也不会自动切换远端目录。

Windows 盘符路径会自动转成 WSL 的 `/mnt/c/...` 或 Git Bash 的 `/c/...`，自定义挂载点和 Linux 原生目录需要手动输入。SSH 目录浏览器使用单独的 OpenSSH 连接，要求远端 Shell 兼容 POSIX，并且可以通过密钥或 SSH Agent 免交互登录。只能使用密码登录时，路径仍可手动输入。

更新检测只读取 GitHub Release 的版本号，不会静默下载或替换程序。新功能需要配套版本的 Daemon。升级或卸载前，先结束已有会话并退出旧 Daemon。重新连接会恢复最近的终端输出，长时间运行的全屏程序可能不完整。

## 许可证

[MIT](LICENSE)。
