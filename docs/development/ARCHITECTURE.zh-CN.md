# DiggerTerm 架构

[English](ARCHITECTURE.en.md) · 简体中文

## 组件与职责

DiggerTerm 是 Go/Wails 3 桌面终端应用，使用 Vue 3/TypeScript 和 xterm.js 展示会话。相同 Go 程序可以作为 GUI 或独立的 `--daemon` 启动。

- 根目录 Go 程序：应用入口、资源嵌入、GUI/Daemon 启动选择。
- `frontend/`：终端显示、会话侧栏、分屏、搜索、设置及与 Wails 服务的交互。
- `internal/desktop/`：GUI 暴露给前端的服务、用户设置和 GUI 生命周期；通过 IPC 调用后台 Daemon。
- `internal/daemon/`：管理会话、系统进程及 IPC 连接；Daemon 负责实际 PTY/ConPTY、WSL 和 SSH 客户端操作。
- `internal/session/`：托管会话的读写、调整大小、输出回放及生命周期。
- `internal/observe/`：外部 Windows/WSL 进程的只读元信息扫描与合并，保持平台来源和进程身份的区别。

## 会话与 IPC 边界

GUI 是 Daemon 的 IPC 客户端。GUI 关闭、隐藏或重新打开时，Daemon 管理的会话可以保持运行并重新连接；Daemon 本身或操作系统退出后不会保留活跃进程。Daemon 使用当前系统用户专属的本地 IPC，界面不得绕过 Daemon 直接接管进程。

快速 cd 根据 PowerShell、CMD 或 POSIX Shell 为选定路径生成经过引用的命令，通过托管终端已有的输入通道排队发送，不自动按回车。用户须在空白 Shell 提示符下自行执行；它不会直接修改其他进程的工作目录。只有本机绝对路径可以设为新建本地会话的默认目录，Dig 观察到的外部进程没有 Quick CD 控制入口。

本机文件夹通过 Wails 3 原生目录对话框选择，无需 Daemon 参与。Windows 盘符路径会转换为 WSL/Git Bash 默认挂载格式，自定义挂载点需手动输入。

SSH 浏览器使用托管会话中不可被重命名改变的目标主机和端口。独立的 OpenSSH BatchMode 进程沿用 ssh_config 与主机密钥验证，只读枚举目录；结果以 NUL 分隔，对耗时、并发和大小设限，不向会话当前 PTY 发送命令。浏览需要免交互密钥/Agent 认证以及 POSIX 兼容远端 Shell；仅支持交互密码或非 POSIX Shell 的 SSH 目标仍可手动输入路径。

一个运行中的托管会话同时只有一个 GUI 控制端。输入、窗口尺寸调整以及终止操作均在 Daemon 侧遵守该控制租约；其他 GUI 不能终止正在被控制的会话，未被控制的托管会话允许显式终止。关闭面板只分离视图；已经接受的终端输入按会话排队，并在分离请求之前发送。明确要求批量终止的操作会重新核对目标并遵守确认流程。

GUI 的 IPC 请求具有有界超时：普通请求 15 秒、会话创建 30 秒、外部进程扫描 45 秒。超时只表示客户端尚未收到最终响应，Daemon 可能仍在完成操作；创建或终止请求超时后，需先重新查询会话状态再决定是否重试，避免重复动作。

## 外部观察与平台边界

`Managed` 表示由 Daemon 创建并管理的会话；`Observed` 表示外部终端或 Agent 进程的只读观察结果。外部观察只提供被允许的元信息，不提供读取 PTY 画面、注入输入、结束进程或接管能力。

Windows PID 与 WSL 各发行版的 PID 属于不同命名空间。外部父子进程关系可能不完整，界面仅把有依据的关系显示为推断。Windows ConPTY、WSL、OpenSSH 与 Linux PTY 使用平台相关实现。

## 数据与配置

终端布局、主题、快捷键及相关偏好在本机用户的配置目录维护。主题支持用户 JSON/CSS，配置文件路径由应用提供的帮助页面或对应代码确定。公开文档只描述可用配置结构，不固化某台开发机的私人路径。

## 代码与发布

源码、测试和构建命令依照相应目录与工作流维护。PR CI 使用 `scripts/check-docs.mjs` 检查文档路径、中英文配对、CHANGELOG 双语分类和当前版本一致性，并校验 PR 模板同步。GitHub Release 的中英文正文从目标 Tag 的 CHANGELOG 生成；具体步骤见 [发布手册](RELEASE.md)。公开文档应描述当前行为与用户限制，不存放单次测试流水或私人环境信息。
