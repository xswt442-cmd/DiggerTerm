# DiggerTerm 发布流程

> 维护者笔记（中文）。用户可见的简短双语 Release Notes 放在 `.github/release-notes/vX.Y.Z.md`。

## 发布范围

- **v0.0.1 是历史 Windows ZIP 包**，附独立 `.zip.sha256`；保留原有 Tag、Release 和附件。
- **从 v0.0.2 起**，正式资产仅包含 Windows x64 `.exe`、Linux x86_64 `.AppImage`、Ubuntu/Debian amd64 `.deb`，另附统一 `SHA256SUMS`。没有 macOS 包，也不再输出 ZIP/tar.gz。
- Windows 包尚未代码签名；Linux 基于 Ubuntu 24.04 构建，其他发行版不承诺完全兼容。

## 发版步骤

1. 通常从 `dev` 通过 PR 合并到 `main` 并检查 GitHub Check；**v0.0.2 是例外：远端旧 `dev` 已误删，本版直接在 `main` 完成发布准备**。同步 `frontend/package.json`、锁文件、`build/config.yml` 及平台版本资源，更新 `CHANGELOG.md`、双语 README 和简短双语 Release Notes。
2. **先验证再打 Tag**。可以从 GitHub Actions 手动执行 **Release → Run workflow**，传入 `vX.Y.Z`，此模式只构建并上传临时 CI 工件，**不会创建 Tag 或 GitHub Release**。同时进行本机 Windows/Ubuntu 测试，核对三种包的安装、托盘、终端、快捷键和依赖。
3. 确认合并后的 `main` 对应已验收的源码，再创建并推送新的注释 Tag `vX.Y.Z`。Tag push 将触发同一 Release Workflow 的 Windows/Linux 构建，并在两端构建全部成功后计算 `SHA256SUMS`、验证安装包格式，再创建正式 GitHub Release。
4. 核验 Release **只有三个可执行安装包和 `SHA256SUMS`**，回下载校验散列、打开 Release Notes、安装并启动。若失败，不要覆盖已发布的 Tag/附件；先定位问题，再决定修复版本。
5. **v0.0.2 发布验收成功后**，从此时发布版本对应的 `main` HEAD 新建并推送 `dev`，恢复后续 `dev` 开发、PR 合并到 `main` 的正常分支工作流；不要从旧本地 `dev` 建分支，也不要提前重建。

## 边界

- 绝不强推、改写旧 Tag 或替换 v0.0.1 的历史资产。
- 构建与测试不结束用户正在运行的 Daemon 托管会话；外部进程只读观察。
- 不提交 `testplace/`、`bin/`、`dist/`、`frontend/dist/`、依赖目录或凭据。
