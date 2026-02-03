# AGENTS 指南

面向一切在 `/home/kkk/Documents/fx` 仓库内运行的智能体。此文件提供构建/测试命令、代码风格、发布流程与常见陷阱，帮助任何自动化代理在最短时间内完成可靠修改。

## 1. 使命关键概览
1. `fx` 是一个 Go 1.23 CLI（Bubble Tea UI + goja JS 引擎）并附带一个 ESM Node CLI（`npm/`）。
2. 没有集中式 lint/format 命令；Go 代码必须 `gofmt`/`goimports`，Node 代码遵循现有风格。
3. 发布二进制靠 `npx zx scripts/build.mjs`，需要 Go toolchain + `gh` 登录。
4. 测试矩阵：`go test ./...` + `npm test`（在 `npm/` 目录执行）。
5. 所有新增 flag 记得注册到 `internal/complete` 以保持自动补全一致。

## 2. 仓库地图
- `main.go`, `view.go`: CLI/TUI 入口与 Bubble Tea model。
- `internal/engine`: 解析 JSON、执行 goja JS、加载 `.fxrc.js`。
- `internal/jsonx`, `internal/jsonpath`, `internal/theme`, `internal/complete`, `internal/utils`：引擎、渲染与辅助模块。
- `npm/`: 纯 Node CLI，包含 `index.js`, `test.js`, `package.json`。
- `scripts/`: `build.mjs`（zx）发布脚本。
- `snap/`: `snapcraft.yaml` 描述 snap 包构建。
- `.github/workflows/`: `test.yml`, `docker.yml`, `snap.yml`, `brew.yml`。
- `testdata/`: golden 输出（通过 `.gitattributes` 标记为 binary）。

## 3. 构建与打包命令
### 3.1 本地二进制
- `CGO_ENABLED=0 go build -o fx .` —— 与 Docker builder 一致（见 `Dockerfile`）。
- `go install ./...` 亦可，但要确保 `$GOBIN` 在 PATH。

### 3.2 发布脚本
- `npx zx scripts/build.mjs` —— 在仓库根执行：
  - `go mod download`
  - 交叉编译 `GOOS ∈ {linux,darwin,windows}`, `GOARCH ∈ {amd64,arm64}`。
  - `gh release upload <tag> fx_<os>_<arch>[.exe]`。
  - 清理临时产物。
- 先配置 `gh auth login` + Node（zx 使用 `npx` 自动加载）。

### 3.3 容器与 snap
- `docker build -t fx .` —— Dockerfile 通过多阶段构建调用 `CGO_ENABLED=0 go build -o fx .`。
- `snapcraft` —— 读取 `snap/snapcraft.yaml`，使用 `plugin: go` 构建并打包 Node runtime。

## 4. 测试命令
### 4.1 Go 测试
- 运行全部：`go test ./...`。
- 单例测试：`go test ./internal/engine -run TestName -count=1`（可替换包路径与测试名）。
- Golden 文件保留 ANSI，需要 `git config --global core.autocrlf false` 避免换行破坏。

### 4.2 Node 测试
- 在 `npm/` 目录：`npm test`（即 `node test.js`）。
- harness 会串行执行全部 `test()` 定义；**无** single-test 过滤逻辑。

### 4.3 端到端注意事项
- JS 测试会 `spawn('node', ['index.js', ...])`，需可执行 `node`。
- Go 测试依赖 `teatest` 伪终端，请保持 `$TERM` 合法；CI 在 Ubuntu/ARM 上验证。

## 5. 工具与格式化
- Go：提交前运行 `gofmt -w` 与 `goimports`（或 `gofumpt`），确保 import 分组：标准库 / 第三方 / 内部。
- Node：采用 ESM、无分号、`const` 优先、`async/await`，保持顶层 `#!/usr/bin/env node` shebang。
- 没有 ESLint/Prettier/Makefile；如需临时脚本放入 `scripts/`，确保文档化。

## 6. Go 代码风格
### 6.1 Imports 与包结构
- 入口示例：`main.go` 将 Bubble Tea 依赖放第二组，内部包第三组；`view.go` 对 jsonx 使用 dot import（仅在 UI 层允许）。
- 需要别名时采用 `tea "github.com/charmbracelet/bubbletea"` 风格。

### 6.2 命名与类型
- 包级布尔 flag 采用 `flagCamelCase`（`flagYaml`, `flagRaw`）。
- 导出接口放在 `internal/engine`（`type Parser interface`），实现结构体保持未导出以控制可见性。
- JSON AST 用强类型枚举 `type Kind byte` + 常量，节点结构在 `internal/jsonx/node.go`。

### 6.3 结构化状态
- Bubble Tea `model` 保存 UI 状态；新增字段需同时更新 `Init/Update/View`。
- 搜索（`searchID`, `searchCancel`）采用 channel + monotonic id，扩展逻辑时保持竞态安全。

### 6.4 错误处理
- CLI 参数错误使用 `fmt.Println`+`os.Exit(1)`，不可 panic。
- 解析 YAML/TOML 时打印 `err.Error()` 并 `os.Exit(1)`（见 `main.go` 183-205）。
- `engine.Start` 返回 exit code，调用方需 `os.Exit(exitCode)`。
- 在 `FX_PPROF` 模式下直接 `panic`，这是唯一允许的 panic 分支。

### 6.5 Theme / Env
- 在读取 env 时先 `os.LookupEnv`，布尔值用 `strings.EqualFold` 方式判定多种真值（`FX_SHOW_SIZE` 示例）。
- 主题通过 `theme.ThemeTester`/`theme.ExportThemes` 暴露；新增主题在 `internal/theme/theme.go` 注册。

### 6.6 `.fxrc.js` 寻址
- 搜索顺序：当前目录 → `$HOME` → `~/.config/fx` → `XDG_CONFIG_DIRS`（`internal/engine/fxrc.go`）。
- 加载顺序决定覆盖；在 agent 修改相关逻辑时保持兼容。

## 7. Node CLI 风格
- ESM + 顶层 `await import('node:fs')`；`const { stdout } = await import('node:process')` 模式常见。
- 数据流水线使用 `mapFilter`, `groupBy`, `skip Symbol`，请优先复用 helpers。
- 错误输出：使用 `underline` 指示符号并 `process.exit(1)`。
- Flags mirror Go CLI（`--yaml`, `--toml`, `--strict`, etc.）；若新增 flag，保持两个实现同步。
- `npm/test.js` 自行封装 `test('name', fn)`；断言基于 `assert.equal` + `status` 检查。

## 8. 配置与环境变量
- `FX_PPROF`：启用 CPU/mem profiling，产出 `cpu.prof`, `mem.prof`。
- `FX_COLLAPSED`, `FX_LINE_NUMBERS`, `FX_SHOW_SIZE`, `FX_NO_MOUSE`, `FX_THEME`, `FX_EXPORT_THEMES` 等控制 UI；新增变量时在 `usage()` 与 `help.go` 同步文档。
- `.gitattributes` 将 `*.golden` 标记为 binary，提交前勿手动转换编码。

## 9. 发布与版本管理
1. 更新版本（Go `version.go`, npm `package.json`, snap `snapcraft.yaml`）。
2. `git tag` / `gh release create` 由 `RELEASE.md` 指导；release script会读取最新 tag 上传资产。
3. Homebrew/snap/docker workflows 位于 `.github/workflows/*.yml`，如需修改 CI 需同步 release 文档。

## 10. CI 期望
- `test.yml`：
  - Go job：`go test ./...` 在 ubuntu-latest + macOS + ARM。
  - Node job：`npm install` + `npm test` inside `npm/`。
- `docker.yml` / `snap.yml` / `brew.yml`：用于发行渠道验证；若触发失败，需检查对应脚本。
- 没有自动 lint；CI 失败多半是测试或构建脚本错误。

## 11. Cursor / Copilot 规则
- 仓库无 `.cursor/rules`、`.cursorrules`、`.github/copilot-instructions.md` 等文件；如未来添加须在本文件补充。

## 12. 贡献者速查表
- ✅ 先读 `RELEASE.md` 了解多生态版本同步。
- ✅ 新增 flag 要：解析逻辑、`complete.Flags`, JS CLI, usage 文案、测试覆盖全部同步。
- ✅ 修改 UI 颜色需更新 `internal/theme` 并手测 `FX_THEME`。
- ✅ 任何生成输出（golden、release 压缩包）请在 PR 描述注明生成步骤。
- ✅ `r`/`R` 键位：`r` 将 JSON 字符串解码为结构体节点，`R` 再次转义回字符串；核心逻辑位于 `main.handleKey` 与 `internal/jsonx/transform.go`。
- ⚠️ 不要在 `main.go` 中引用新的全局状态而未更新 `model`；Bubble Tea 需要显式字段。
- ⚠️ 不要在 Node CLI 引入 CommonJS/TypeScript；保持 ESM + 原生 API。
- ⚠️ 不要假定存在 lint/format 脚本；提交前自行运行 gofmt 与保持 JS 风格。

## 13. 常见排错 Tips
- `fx` 卡死：检查输入是否无限流；`engine.Start` 在 `--slurp` 模式会积累所有节点。
- `node test.js` 失败：确保已在 `npm/` 目录执行并使用当前 Node LTS。
- `npx zx scripts/build.mjs` 报 `gh` 错：运行 `gh auth status`，并保证拥有发布权限。
- Snap 构建失败：确认 `snapcraft` 本地安装且 Multipass/ LXD 正常。
- Golden 失配：运行 `go test ./...`，失败日志会指明差异；可直接覆盖 `testdata/*.golden` 但需人工确认。

## 14. 文件定位速查
- CLI flag 解析：`main.go` 90-170；新增 flag 需同步 `usage()` 与 `internal/complete/complete.go`。
- 文本渲染：`view.go` 15-220 负责 viewport、键绑定、搜索跳转。
- JSON 解析：`internal/jsonx`（AST）+ `internal/jsonpath`（查询）；调试 parse 逻辑请查看 `NewJsonParser`。
- JS 执行：`internal/engine/engine.go`，含 goja VM 初始化与 `.fxrc.js` 注入流程。
- 主题：`internal/theme/theme.go` 维护所有 `lipgloss` 样式与 `FX_THEME` 变体。
- Shell 自动补全：`internal/complete/complete.go`，`complete.Bash/Zsh/Fish` 文本需一并更新。
- Node CLI：`npm/index.js`（主逻辑）、`npm/test.js`（harness）、`npm/package.json`（脚本）。
- 发布脚本：`scripts/build.mjs`；外部发布说明：`RELEASE.md`。
- CI 定义：`.github/workflows/*.yml`；容器逻辑可从 `Dockerfile` 读取。

## 15. 任务前 Checklist
1. 确认是否需要并行修改 Go 与 Node 两端；若是，先列出同步点。
2. 运行 `git status -sb`，避免在脏工作区投入修改；必要时备份未跟踪文件。
3. 评估是否要更新 golden 文件/截图，并在 PR 描述写明生成步骤。
4. 若任务涉及发布/打包，确保本地 `gh auth status`、`go env`、`npm -v` 均可用。
5. 修改前阅读相关子模块目录下最近一次 commit，理解当前风格（`git log -p -- <path>`）。
6. 修改完成后执行对应测试矩阵（至少受影响语言），并附带关键命令输出。

---
本文件约 150 行，供所有后续代理复用。如需扩写，请保持章节编号与细粒度指引风格。
