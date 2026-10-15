# 开发阶段验证

本轮开发保留 `dev` 分支，不发布版本。GUI 仅通过 IPC 使用 Daemon 会话；关闭视图和 GUI 不结束会话。外部进程观察始终只读。

## P0：基线与诊断

2026-10-15，Windows 原生环境：`go test ./...`、`go vet ./...`、`npm run test:preferences`、`npm run test:shortcuts`、`npm run build` 全部通过。扩展 WSL 测试按项目现有环境开关执行。

SSH 使用系统 OpenSSH：`ssh -tt`，端口使用独立参数。认证、主机指纹和 SSH 配置由 OpenSSH 处理。PTY 在子进程退出后关闭，GUI 客户端关闭不调用会话终止。尚未获得用户报告的闲置断联现场环境，不能认定原因或宣称修复。

终端复制代码核查：smart 模式原有 alternate-screen 判断会影响已有 xterm 选区的 Ctrl+C；复制无选区缺少反馈。nano 自身标记区域不等于 xterm 选区。真实鼠标选择与系统剪贴板验收单独记录。

## 验收边界

- SSH 现场：对比系统 SSH 与 DiggerTerm，记录闲置 15/30/60 分钟、隐藏窗口、切换 Pane、退出重开、休眠和网络中断；只使用可控测试主机。
- 复制：使用无敏感内容的中文、多行和 Unicode 测试字符串，在 nano/vim/tmux、WSL/SSH 中验证普通拖动、Shift 拖动、显式复制入口，以及粘贴到另一应用。
- 画面恢复：Daemon 当前保存最近 256 KiB 原始输出。输出回放不等于无损 VT 屏幕快照，长期运行全屏程序仍需真实画面验证。
- 不在已有用户 Daemon 上执行测试；所有新测试使用独立 IPC 命名空间，只清理自己创建的会话。
