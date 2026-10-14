# DiggerTerm 发布流程

> 维护者内部说明。用户可见的 Release Notes 放在 `.github/release-notes/vX.Y.Z.md`，不要把本文件发布为 Release Notes。

## 范围

- **已发布的 v0.0.1** 是 Windows x64 ZIP 预览版，保留其历史 Tag、附件及 `.zip.sha256` 校验文件，不修改或重发。
- **下一版起计划只提供三种程序包**：Windows x64 `.exe`、Linux x86_64 `.AppImage`、Ubuntu/Debian amd64 `.deb`。附带统一的 `SHA256SUMS` 校验文件；不再将 ZIP、tar.gz 或 macOS 程序包作为正式资产。
- Windows 程序目前未代码签名。Linux 在 Ubuntu 24.04 环境构建，其他发行版尤其旧版 glibc 的兼容性以实机验收为准。
- **当前 `.github/workflows/release.yml` 仍按 v0.0.1 的 Windows ZIP 流程发布。创建下一版 Tag 之前，必须先修改并验证三平台资产的构建和上传流程。** 本地测试包不等于已发布的 Release。

## 下一版发布步骤

1. 先完成 `dev → main` PR 审核与合并，确认 Check 通过。不要强推或重写历史。
2. 同步版本信息（`frontend/package.json`、锁文件、`build/config.yml`），更新 `CHANGELOG.md`；准备简短的双语 `.github/release-notes/vX.Y.Z.md`。只写主要变化、下载与校验、必要限制。
3. 验证 Windows 与 Linux GUI：终端键盘/剪贴板、nano 等全屏程序、托盘、后台会话恢复，以及 `.exe`、`.AppImage`、`.deb` 实际运行。至少运行前端测试、`go test ./...`、`go vet ./...` 和 Wails 构建。
4. 将 Release Workflow 改为只上传上述三个安装包及 `SHA256SUMS`。在发布前验证三包存在、目标架构正确、可启动、SHA-256 一致；确认 Linux 动态库依赖。
5. 从已合并且已验收的 `main` 提交创建新的注释 Tag，触发 Release Workflow；发布后检查附件并回下载校验。失败时保留现状，修复后使用新版本发布，**不得重写已发布的 Tag**。

## 边界

- 打包和测试不得终止用户正在运行的 Daemon 托管会话。
- 不提交 `testplace/`、`bin/`、`dist/`、依赖目录、凭据或本地安装包。
- 只发布已验证的功能；Daemon 崩溃或系统重启后的会话存活不作保证。
