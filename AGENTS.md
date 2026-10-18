# DiggerTerm · Agent 工程约束

- DiggerTerm 使用 Go/Wails 3、Vue 3/TypeScript 和 xterm.js。GUI 是 IPC 客户端，只有独立的 Session Daemon 持有 PTY/ConPTY/SSH 会话。
- 关闭、隐藏或重启 GUI 不应终止仍由 Daemon 托管的会话；Daemon 或系统终止后不保证会话存活。不得为了测试结束真实用户会话。
- 严格区分 Managed（Daemon 管理）与 Observed（只读观察）的进程。观察外部 Windows/WSL 进程不能授予终止、注入输入或接管 PTY 的权限；涉及控制必须有明确授权。
- Windows ConPTY、WSL、Unix PTY 和 OpenSSH 平台逻辑保持隔离，不夸大尚未支持的平台行为。
- 修改前重新读取当前文件；保持更改范围可追溯，只描述最终存在的设计与用户行为。不要记录失败尝试、已删除方案或本机调试流水。
- README 只介绍用户实际可用功能、下载与安装、操作和支持限制。正式文档不得包含个人绝对路径、私有 SSH 别名、单机配置、运行日志、待办任务和短期验收状态。代码注释只解释当前有效的约束；复杂图示优先 Mermaid。
- `CHANGELOG.md` 是版本内容的唯一来源；GitHub Release 从 Tag 对应章节生成。PR 正文中英双语，只写最终更改和真实通过的验证；未来任务交由 Issue 追踪。
- 用户可见文档中英双语语义一致；发布操作见 [RELEASE](docs/development/RELEASE.md)。`CLAUDE.md` 只引用本文件。
- 根据实际改动运行 `go test ./... -count=1`、`go vet ./...`、`npm --prefix frontend test`、`npm --prefix frontend run build`；涉及发行时检查 `node scripts/verify-release.mjs vX.Y.Z`。
- `testplace/` 为 Git 忽略的本地草稿与 Agent 交接目录，保留 `icons/` 和已有用户文件。
- Git 提交、推送、切换或重置分支、创建 Tag、发布 GitHub Release、修改用户资产或改动外部服务均需当前任务的明确授权。
