# DiggerTerm 发布流程

本手册记录可重复执行的版本检查、打包、发布和回滚规则。GitHub Release 的变更内容从对应 Tag 的 `CHANGELOG.md` 生成。下载信息和签名状态由校验过的发行资产生成，分别写入简体中文和 English 两段，不另维护一份版本说明。

## 发布范围

- `dev` 用于开发，`main` 用于经过验证的发行代码。
- Windows x64 发布为面向当前用户的 NSIS `.exe` 安装包，Linux x86_64 发布 `.AppImage` 和 Ubuntu/Debian 的 `.deb`，并附统一的 `SHA256SUMS`。Windows 安装包未签名，Linux 构建以 Ubuntu 24.04 为目标平台。
- 历史版本 `v0.0.1` 发布的是 Windows ZIP 包和单独的校验文件。已有的 Tag 和发行资产保持不变。
- CI 构建产物只用于验证，只有正式 GitHub Release 中的附件属于公开发行资产。

## 发行准备

1. 在 `dev` 整理通过审查的源码，并更新 `CHANGELOG.md` 的中英文 Unreleased 条目。条目只写已经实现的行为变化。
2. 更新 `frontend/package.json`、`frontend/package-lock.json`、`build/config.yml` 和对应平台的版本资源。
3. 将要发布的条目归入 `[X.Y.Z]` 章节，确认两种语言描述的是同一件事。版本章节的日期填写计划发布日。发布日变化时，在创建 Tag 前同步修改。
4. 使用项目测试和构建命令验证功能、Daemon 会话边界和发布元数据：

   ```sh
   npm --prefix frontend ci
   npm --prefix frontend test
   npm --prefix frontend run build
   go test ./... -count=1
   go vet ./...
   node --test scripts/check-docs.test.mjs scripts/release-notes.test.mjs scripts/release-downloads.test.mjs
   node scripts/check-docs.mjs
   node .github/engineering/scripts/skills.mjs --check .
   node .github/engineering/scripts/sync-community.mjs --check .
   node scripts/verify-release.mjs vX.Y.Z
   ```

5. GitHub Actions 的 `publish → Run workflow` 支持手工输入版本，用于 Windows/Linux 构建预检。手工运行只产出临时 CI 工件，不创建 Tag 或 GitHub Release。

可复用的验证工作流会对完整 Git 历史执行 Gitleaks Secret Scan。正式 Tag 触发的发布流程也会先完成这项检查。

## Tag 与发布

1. 将验证通过的提交合入 `main`，核对正式版本对应的 CI 结果。
2. 基于发布用的提交创建并推送新的注释 Tag `vX.Y.Z`。
3. `.github/workflows/publish.yml` 运行 Windows/Linux 构建和发布资产检查，核对三个安装包、Windows 安装与卸载行为、版本标记、文件格式及 SHA-256。
4. 工作流通过 `scripts/release-notes.mjs` 提取目标版本的中英双语变更，再按校验过的附件名称生成两种语言的下载表和签名说明。随后上传三个安装包和校验文件，创建 GitHub Release。标题只写产品和版本，不加“更新说明 / Release highlights”：

   ```sh
   node scripts/release-notes.mjs --changelog CHANGELOG.md --tag vX.Y.Z --manifest frontend/package.json --repo xswt442-cmd/DiggerTerm --assets-dir dist
   ```

5. 下载发行附件并校验 SHA-256，然后在目标平台安装、启动，验证基本终端功能。

## 安全和恢复

- 不强制推送，不移动旧 Tag，也不覆盖已发布的二进制附件。已发布 Release 的文字修订，只能在核实对应 Tag、实际附件和双语内容之后进行，修订不算重新发版。
- Windows 当前安装包不提供代码签名，发布资产没有可验证的发布者数字签名。`SHA256SUMS` 只用于完整性检查，不证明发布者身份。
- 测试和打包不得终止用户正在使用的 Daemon 会话，也不得接管外部观察到的进程。
- 不将 `testplace/`、`bin/`、`dist/`、`frontend/dist/`、本机日志、个人配置或密钥提交到 Git。
- 发布失败时，按提交、Tag 和发行附件的实际状态处理。本机或 CI 的一次成功不代表正式发行已经完成。
