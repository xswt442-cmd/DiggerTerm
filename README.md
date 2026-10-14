# DiggerTerm

一个轻量的桌面终端工作台：左侧统一会话栏，右侧交互式终端。计划支持本地 Shell、Windows/WSL 和 SSH，并以只读方式发现外部进程。

**当前状态：仅完成桌面工程骨架。** 尚未实现 PTY、SSH、托盘、后台会话保活或进程观察；启动后展示占位界面，不会控制任何终端。

## 技术栈

Go + Wails 3（Beta）+ Vue 3 / TypeScript。暂不拆分后台 Daemon，等会话生命周期需求验证后再决定。

## 本地开发

需要 Go 1.25+、Node.js 22+，以及 [Wails 3 CLI](https://v3.wails.io/quick-start/installation/)。确保 `$(go env GOPATH)/bin` 在 `PATH` 中；Windows PowerShell 可临时执行 `$env:PATH = "$(go env GOPATH)\bin;$env:PATH"`。

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
cd frontend
npm ci
cd ..
wails3 dev
```

检查和构建：

```sh
cd frontend && npm run build && cd ..
go test ./...
wails3 build
```

生成的依赖、前端产物与二进制文件不提交。图标源文件是 `icons/diggerterm-icon.svg`，桌面构建使用已转换的 `build/appicon.png`。

## 设计底线

- 自有会话和外部发现的进程是不同对象；外部进程默认只读。
- 外部进程控制需要针对具体操作显式确认，且不能越过系统权限。
- 隐藏窗口与终止会话是两个独立动作；会话持久化未实现前不作此保证。
- 优先保持结构简单、跨平台差异隔离，不提前引入多进程架构。

MIT License.
