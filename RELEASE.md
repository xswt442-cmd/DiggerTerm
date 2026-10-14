# DiggerTerm 发布流程

> 维护者笔记。用户可见的 Release Notes 放在 `.github/release-notes/vX.Y.Z.md`，**不要用本文件作为 GitHub Release 说明**。

## 范围

目前只发布 **Windows x64 便携 ZIP**，不承诺 Linux/macOS GUI 可用；暂不做安装器、代码签名或自动更新。GUI 退出后的会话保活仅依赖仍运行的 Daemon。

## 每次发布

1. 从 `dev` 准备版本。同步 `frontend/package.json`、锁文件、`build/config.yml` 及实际构建所需的版本元信息；更新 `CHANGELOG.md`。
2. 新建简短的双语 `.github/release-notes/vX.Y.Z.md`，只写本次主要变化、下载与 SHA-256 校验方式、必要的限制。避免复制 README 或整份 CHANGELOG。
3. 本机验证（先构建前端，因为 Go 使用 `//go:embed frontend/dist`）：

   ```powershell
   npm --prefix frontend ci
   npm --prefix frontend run build
   go test ./...
   go vet ./...
   wails3 build
   ./scripts/package-windows.ps1 -Version vX.Y.Z
   ```

4. 推送 `dev`，确认 GitHub `Check` 成功；按实际分支策略将 `main` **非强制快进**到经过验证的提交。
5. 从该提交创建并推送注释 Tag `vX.Y.Z`，触发 `.github/workflows/release.yml`。
6. 核验 GitHub Release 正式发布、ZIP 与校验文件存在，回下载比对 SHA-256，并检查归档内容。失败时保留现状，绝不强推或重写已有 Tag。

## 约束

- 只有 Daemon 持有 PTY；发布操作不可终止现有运行中会话。
- 不要把 `testplace/`、依赖目录、`frontend/dist/`、`bin/`、`dist/` 或凭据提交进仓库。
- 不修改已发布版本的 Tag 和附件；修复后发布新版本。
- 本仓库 v0.0.1 的历史发布包使用单独的 `.zip.sha256` 文件校验。
