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

一个托管会话同时只有一个 GUI 控制端；关闭某个终端面板只解除该视图绑定。明确要求终止托管会话的操作需要重新核对目标并遵守确认流程。

## 外部观察与平台边界

`Managed` 表示由 Daemon 创建并管理的会话；`Observed` 表示外部终端或 Agent 进程的只读观察结果。外部观察只提供被允许的元信息，不提供读取 PTY 画面、注入输入、结束进程或接管能力。

Windows PID 与 WSL 各发行版的 PID 属于不同命名空间。外部父子进程关系可能不完整，界面仅把有依据的关系显示为推断。Windows ConPTY、WSL、OpenSSH 与 Linux PTY 使用平台相关实现。

## 数据与配置

终端布局、主题、快捷键及相关偏好在本机用户的配置目录维护。主题支持用户 JSON/CSS，配置文件路径由应用提供的帮助页面或对应代码确定。公开文档只描述可用配置结构，不固化某台开发机的私人路径。

## 代码与发布

源码、测试和构建命令依照相应目录与工作流维护。GitHub Release 的中英文正文从目标 Tag 的 CHANGELOG 生成；具体步骤见 [发布手册](RELEASE.md)。公开文档应描述当前行为与用户限制，不存放单次测试流水或私人环境信息。
