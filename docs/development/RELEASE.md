# DiggerTerm 发布流程

本手册记录可重复执行的版本检查、打包、发布和回滚规则。GitHub Release 正文从发布 Tag 对应的 `CHANGELOG.md` 自动提取中英双语内容，不维护第二份版本说明。

## 发布范围

- `dev` 用于开发，`main` 用于经过验证的发行代码。
- Windows x64 发布 `.exe` 安装包；Linux x86_64 发布 `.AppImage`、Ubuntu/Debian `.deb`，附统一 `SHA256SUMS`。Windows 发行包未签名，Linux 构建以 Ubuntu 24.04 为目标平台。
- 历史 `v0.0.1` 发布的是 Windows ZIP 包及单独校验文件；既有 Tag 和发行资产保持不变。
- CI 构建产物用于验证，只有正式 GitHub Release 中的附件属于公开发行资产。

## 发行准备

1. 在 `dev` 整理经过审查的源码，并更新 `CHANGELOG.md` 的中英文 Unreleased 条目；条目只包含实际实现的行为变化。
2. 更新 `frontend/package.json`、`frontend/package-lock.json`、`build/config.yml` 和对应平台的版本资源。
3. 将要发布的条目归入 `[X.Y.Z]` 章节，确认两种语言描述相同事实。
4. 使用项目测试和构建命令验证功能、Daemon 会话边界和发布元数据：

   ```sh
   npm --prefix frontend ci
   npm --prefix frontend test
   npm --prefix frontend run build
   go test ./... -count=1
   go vet ./...
   node --test scripts/release-notes.test.mjs
   node scripts/verify-release.mjs vX.Y.Z
   ```

5. GitHub Actions 的 `Release → Run workflow` 支持手工输入版本进行 Windows/Linux 构建预检；手工运行只产出临时 CI 工件，不创建 Tag 或 GitHub Release。

## Tag 与发布

1. 将验证通过的提交合入 `main`，核对正式版本对应的 CI 结果。
2. 从发行提交创建并推送新的注释 Tag `vX.Y.Z`。
3. `.github/workflows/release.yml` 运行 Windows/Linux 构建和发布资产检查，确认三个安装包、版本标记、文件格式及 SHA-256。
4. 工作流通过 `scripts/release-notes.mjs` 提取目标版本的中英双语正文，再上传三个安装包和校验文件，创建 GitHub Release：

   ```sh
   node scripts/release-notes.mjs --changelog CHANGELOG.md --tag vX.Y.Z --manifest frontend/package.json
   ```

5. 下载发行附件校验 SHA-256，在目标平台安装、启动并验证基本终端功能。

## 安全和恢复

- 不强推、不移动旧 Tag，也不覆盖已发布 Release 的正文或二进制附件；修复应采用经过验证的新版本。
- 测试和打包不得终止用户真实的 Daemon 会话或控制外部观察进程。
- 不将 `testplace/`、`bin/`、`dist/`、`frontend/dist/`、本机日志、个人配置或密钥提交到 Git。
- 发布失败按提交、Tag 和发行附件的实际状态处理；本机或 CI 的一次成功不等于正式发行。
