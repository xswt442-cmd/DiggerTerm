# 开发阶段验证

本轮开发保留 `dev` 分支，不发布版本。GUI 仅通过 IPC 使用 Daemon 会话；关闭视图和 GUI 不结束会话。外部进程观察始终只读。

## P0：基线与诊断

2026-10-15，Windows 原生环境：`go test ./...`、`go vet ./...`、`npm run test:preferences`、`npm run test:shortcuts`、`npm run build` 全部通过。扩展 WSL 测试按项目现有环境开关执行。

SSH 使用系统 OpenSSH：`ssh -tt`，端口使用独立参数。认证、主机指纹和 SSH 配置由 OpenSSH 处理。PTY 在子进程退出后关闭，GUI 客户端关闭不调用会话终止。尚未获得用户报告的闲置断联现场环境，不能认定原因或宣称修复。

终端复制诊断区分 nano 自身标记区域、xterm 选区和系统剪贴板。真实鼠标选择与系统剪贴板验收单独记录。

## P1：用户主题

用户主题服务位于 GUI 进程，读取用户主目录下的 `user/DiggerTerm/`。只扫描一级 JSON/CSS，保留文件 `keybindings.json` 不作为主题。每文件上限 128 KiB，拒绝符号链接和越界读取；CSS 使用词法与语法解析，拒绝外部资源、at-rules、转义与不受支持的函数。

前端使用 `builtin:` / `user:` 稳定 ID，兼容旧的内置主题偏好。JSON 部分字段从基础主题合并，名称来自 `name`；同名主题显示来源 ID。CSS 切换替换同一个样式节点，依次应用 JSON、所选 CSS、全局 `custom.css`；禁用自定义 CSS 保留 JSON 配色。主题目录不存在时使用内置主题，删除所选主题后刷新恢复默认并显示提示。

自动测试覆盖多个主题、中文名称、重名、成对文件、CSS-only、全局 CSS、坏 JSON/CSS、大小限制、链接、刷新后文件删除、偏好迁移及深合并。Windows 大小写文件名碰撞由文件系统限制，独立文件测试在大小写敏感文件系统执行。

## P1.5：复制与快捷键

smart 模式已有真实终端选区时，Ctrl+C 在 alternate screen 中也复制；无选区时保留 ETX。native 模式的 Ctrl+C 继续原生行为。显式复制无选区给出提示，失败可通过同一按钮重试；提示不包含选区内容。

终端工具栏与右键入口提供复制、粘贴和复制模式。复制模式通过 xterm 公开选区 API 提取当前可见文字，保留自动换行文本的连续性，提供只读文本选择，退出即清空副本；不读取完整历史、不向 PTY 发送选择控制序列。xterm 保留 Shift 选择机制，macOS 启用 Option 选择；真实 nano 验收中 Shift 拖动未产生选区，应使用复制模式。

GUI 侧 `KeybindingService` 读取/原子保存固定配置文件，独立校验动作白名单、scope、大小、组合键、保留按键和有效绑定冲突；损坏配置显示错误并保留当前有效绑定。设置页支持多键位、清空、录制、重新加载、打开已保存配置和恢复默认。输入框、IME/AltGr 与设置对话框优先处理自身按键。

自动检查：Go 全套测试、vet，前端偏好、快捷键、绑定解析测试和构建通过；绑定交换、跨 scope 冲突、符号键、未知动作/字段和链接路径拒绝均有测试。真实 WebView 系统剪贴板和 nano/vim 鼠标行为的验收结果仍需单独记录。

## P2：分屏与控制权

`workspaceLayout.ts` 使用纯函数维护版本化布局，支持左右、上下与嵌套切分、聚焦、会话指派、关闭和比例调整。最多 16 个面板、12 层嵌套；恢复时清除不存在、退出和重复会话。面板以稳定节点渲染，调整树结构不会重新挂载保留的终端。

GUI 偏好保存树和焦点。会话 ID 包含随机 Daemon 实例前缀，旧数字 ID 不用于恢复；实例更换时旧绑定保持空白，等待用户选择。一个会话最多一个可交互面板，重复选择聚焦已有面板。关闭面板只调用 Detach，不调用 Terminate；外部进程页面保留只读边界。

Daemon 限制每个会话由一个 IPC 客户端输入/调整尺寸，Attach/Detach 和客户端断开可交接控制权。第二个 GUI 无法输入或争抢尺寸，需等待控制释放后重试。GUI 按会话串行发送 Attach、输入、有效尺寸和 Detach，关闭面板后丢弃尚未发送的输入。各面板独立处理输出和快照。

每个 IPC 客户端按会话排队输入、Resize 和 Attach/Detach，最多 256 个待处理请求。不同会话和只读刷新独立处理；显式终止无需排在阻塞写入后面。客户端断开丢弃未执行队列，不结束会话。实际 IPC/PTY 测试验证同一客户端中一个会话等待控制锁时，其他会话仍能 Resize，List 仍能返回。

自动检查覆盖布局畸形输入、深度/数量/比例限制、重复绑定、旧会话 ID 拒绝、真实 PTY 的控制交接和客户端断开存活。Windows 与 WSL Ubuntu 的 `go test ./...`、`go vet ./...` 均通过；WSL 包含真实 GNU nano 原生控制键测试。前端全部单测与构建通过。

真实浏览器 → Wails server → 私有 IPC → ConPTY 验收通过：两会话输出隔离，三面板嵌套，重复选择聚焦已有面板，关闭面板保留两会话运行，鼠标比例 50%→57%、键盘 57%→60%。授予测试浏览器 Clipboard 权限后，选区复制与向焦点面板粘贴测试文字通过，另一面板没有该输入。Windows WebView2 原生剪贴板与另一应用之间的粘贴仍需独立验收。

主题界面验收通过 JSON 中文名称、CSS 全局优先级/禁用、删除主题刷新回退、保留 keybindings.json 排除。快捷键界面录制 F9、重复冲突拒绝、有效保存和重新加载通过。Wails 原生 Clipboard 布尔返回值由 GUI 服务检查；Windows 失败时使用 GUI 侧原生剪贴板库，仍失败才尝试浏览器接口。真实 server runtime 的不支持错误也有自动测试。

## P3：重连、状态与画面恢复

GUI 的手动重连只拨号已有 Daemon，关闭旧 IPC 客户端后重新 List、Attach 和 Snapshot，不启动或停止 Daemon，也不重新发送历史命令。连接失效显示状态未知，禁止创建会话和输入；PTY 输出流关闭、SSH 子进程退出与 IPC 失联分别显示。会话诊断包含 ID、PID、开始/退出时间、退出码、退出类型和流状态。版本号合并防止延迟的 List/Create 回复覆盖更新的退出或重命名事件。

Daemon 记录结构化的会话 ID、PID、profile、时间、退出码和错误类别，不记录主机地址、用户名、命令参数或原始终端输出。Unix 正常 PTY 关闭的 EIO 与 Windows 的正常流结束按各平台处理。SSH 连接被隔离测试服务拒绝后，真实 IPC 可读到退出码 255 和 SSH 退出类型；其他会话保持运行。

Snapshot 明确标记最近 256 KiB 回放是否截断。前端按字节 Offset 去重和拼接，检测缺口后暂停输入并提供重新回放；回放期间限制事件积压，历史解析不向 PTY 回送终端协议响应，随后允许实时响应。使用与前端相同版本的真实 xterm 5.5.0 headless 引擎验证 UTF-8/ANSI 逐字节解析和协议响应；超过容量的 alternate-screen 用例明确证明截断回放会丢失早期屏幕状态。当前没有无损 VT 屏幕快照。

真实浏览器 → Wails server → IPC → ConPTY 验收：两会话手动重连后 ID/PID 保持，测试命令未重跑；一个 CMD 以代码 7 退出时诊断更新，另一个会话继续运行；超过 256 KiB 输出显示截断提示。另一 GUI 进程不能控制已占用会话，原 GUI Detach 后可重试并操作同一个 PID。只读外部观察页没有终端输入和终止入口。

Windows 中的真实 WSL/nano 7.2 验收：nano 打开时手动重连保留 PID、备用屏幕文字与光标状态；无选区 Ctrl+C 保留 nano 操作；复制模式复制中文和可见标记。Shell 中 240 字符连续文本跨三行自动换行后，复制结果与完整原文一致。nano 自身横向裁剪的屏幕外文字不在可见副本中，Shift 拖动未得到 xterm 选区。真实 Vim 使用 `-Nu NONE -n -c 'set mouse=a'` 打开独立三行夹具，复制模式精确复制中文/多行文字，手动重连保留 WSL 会话 PID 与 `3,18` 光标位置，Ctrl+C 保持 Vim 运行。这些复制检查使用已授权的真实浏览器 Clipboard API，不代表原生 WebView2 跨应用剪贴板已验收。

复制模式按 textarea 实际文字和选区提取内容，兼容 Windows CRLF 到 textarea LF 的转换。真实右键菜单验收通过精确子串复制、保留右键前选区、退出复制模式和焦点面板粘贴；粘贴后由 CMD 变量展开验证实际执行。菜单显示在主题根容器，分屏边缘的完整显示与点击均通过验收。Esc 先关闭菜单并保留复制模式，再按 Esc 退出复制模式。

## SSH 闲置对照

隔离的本机 loopback SSH 服务使用临时密钥、严格主机指纹检查和真实 cmd.exe。并行比较直接系统 OpenSSH 与 Manager/ConPTY 托管 OpenSSH，以及配置 ServerAliveInterval=15 / 未配置 ServerAlive 两组。15、30、60 分钟闲置后均执行成功：预期标记只通过变量展开产生，不出现在输入命令中。完整运行约 3604 秒。该长程结果覆盖 Manager/ConPTY，不覆盖 GUI/Daemon IPC 生命周期，也不模拟 NAT、VPN、休眠或网络切换。

测试日志不包含原始终端文字，清理等待自己创建的进程退出；同一测试也通过了 1 分钟检查。生产 SSH 参数保持继承系统配置，未强行加入保活覆盖。受影响服务器的偶发闲置问题尚未复现，不能宣称修复；新增诊断可用于区分实际退出与 GUI/IPC 连接问题。

可在 Windows 显式运行 `DIGGERTERM_SSH_IDLE_MINUTES=60` 对应的 `TestWindowsSSHIdleLoopback`，并将 Go 测试超时设置为至少 70 分钟；默认测试跳过长程等待。短检查使用值 `1`。

## P4：Bash 补充阶段

Windows 的 `bash` 配置仅识别 Git for Windows：从标准安装目录、用户安装目录、只读安装注册信息和经过 Git 安装标记确认的 git.exe 所在目录寻找 `bin/bash.exe` 或 `usr/bin/bash.exe`。始终使用确认后的绝对路径，绝不查找裸 `bash.exe` 或转到 WSL。没有 Git Bash 时不列出该配置，也拒绝创建。Git Bash 使用交互登录模式与 `CHERE_INVOKING=1`，保留用户指定工作目录。

Linux/macOS 的显式 Bash 配置使用本机已安装 Bash；保留原默认 Shell 配置。前端支持将 Bash 设为默认 Shell，并将新建会话工作目录交给 Daemon；控制权、分屏、退出与重连继续使用现有会话机制。真实 Windows Git Bash ConPTY 和 WSL 中的 Linux Bash PTY 均通过变量展开与含空格工作目录检查；测试 HOME 独立，失败信息不输出原始终端文字。

真实浏览器验收通过 Git Bash 的 Bash 版本和 MINGW64 系统类型、与 CMD 分屏输出隔离、手动重连后保持两会话 PID，以及将 Bash 设为默认 Shell 后重新加载的持久化。未将 Windows 的 WSL Bash 当作 Git Bash。

Shell 列表由正在运行的 Daemon 提供；新增 Bash 与 Attach/Detach 控制接口需要配套的新 Daemon。手动重连不替换 Daemon。旧 Daemon 的活跃会话可继续使用原 GUI；更新时先完成现有会话，再由用户显式停止旧 Daemon 并重新打开新版应用，GUI 不自动终止会话或升级后台进程。

Windows 原生剪贴板验证使用真实 Wails 事件循环和 ClipboardService，复制中文、多行测试文字后由独立 PowerShell 进程精确比对，再由 PowerShell 写入另一标记并由服务读取比对。Wails 返回失败时使用 [golang.design/x/clipboard](https://github.com/golang-design/clipboard) v0.11.0 的原生 API。最终测试通过；这证明原生 API 的跨进程读写，原生 WebView2 UI 到另一应用的完整操作仍未验收。此测试仅在显式设置 `DIGGERTERM_TEST_NATIVE_CLIPBOARD=1` 时执行，会覆盖当前剪贴板为无敏感测试文字；默认检查不会改动系统剪贴板。

## 集成验证与协作

| 环境 | 已通过的检查 |
| --- | --- |
| Windows 原生 | `go test ./...`、`go vet ./...`、Daemon/Session race 检查、真实 server runtime 剪贴板失败测试、原生 ClipboardService 跨进程读写、原生与 server 开发构建 |
| WSL Ubuntu / Go 1.25 | `go test ./...`、`go vet ./...`、Linux production-tag 开发构建、真实 nano 控制键与保存/搜索等交互测试 |
| 前端 | 39 项测试全部通过、TypeScript 检查与 Vite production 构建；测试使用真实纯函数和真实 xterm 引擎 |
| 界面与 IPC | 主题、快捷键编辑、嵌套分屏、比例保存、焦点复制粘贴、控制权交接、手动重连、退出诊断与截断提示的真实浏览器验收 |

实际使用三个 gpt-6-luna 子代理，分别负责主题后端/布局与安全交叉检查、SSH 配置及真实闲置实验、复制/恢复界面验收。主代理检查实际差异、完成集成，并运行 Windows、WSL 与前端全套验证。未计算或声称精确费用节省。

所有测试使用独立 IPC 命名空间和自建会话，只清理自身创建的进程；未操作现有用户 Daemon。阶段提交均保留在 `dev`，没有发布版本。

## 待现场验收

- SSH 现场：对比系统 SSH 与 DiggerTerm，记录闲置 15/30/60 分钟、隐藏窗口、切换 Pane、退出重开、休眠和网络中断；只使用可控测试主机。
- 复制：原生 Windows WebView2 UI 与另一应用之间的完整系统剪贴板流程；vim Shift 拖选、tmux 和远端 SSH 鼠标模式；macOS/Linux 原生 GUI、IME/AltGr 与默认键位。nano Shift 拖选尚不能记为通过，界面提供已验收的复制模式。
- 画面恢复：Daemon 当前保存最近 256 KiB 原始输出。输出回放不等于无损 VT 屏幕快照，长期运行全屏程序仍需真实画面验证。
- 无损恢复的后续方案需要由 Daemon 维护主/备用屏幕、光标和字符属性，在同一个 Offset 边界导出屏幕状态，再接续实时增量；以现有超容量和逐字节 UTF-8/ANSI 用例验证屏幕等价，保留现有 PTY 所有权与 IPC 边界。这项屏幕状态能力当前未实现。
- 不在已有用户 Daemon 上执行测试；所有新测试使用独立 IPC 命名空间，只清理自己创建的会话。
