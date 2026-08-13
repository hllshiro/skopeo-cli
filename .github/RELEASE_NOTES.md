## v2.0.0 — Complete rewrite in Go / 完全用 Go 重写

skopeo-cli has been rewritten from Bun/TypeScript to Go. The CLI interface remains compatible with v1.x, with several improvements and fixes.

skopeo-cli 已从 Bun/TypeScript 重写为 Go。CLI 接口保持与 v1.x 兼容，并带来多项改进与修复。

### Highlights / 亮点

- **Go rewrite / Go 重写**: no third-party dependencies, statically linked binaries, no runtime required / 零第三方依赖、静态链接二进制、无需运行时
- **Simpler releases / 更简单的发布**: all platforms are cross-compiled from a single job (`linux`, `windows`, `macos` on `amd64`) / 单一任务交叉编译所有平台
- **Taskfile**: `task build`, `task build:all`, `task test`, `task cli -- <args>`, and more / 提供常用开发任务
- **Tests / 测试**: full unit-test suite ported, plus integration tests behind `go test -tags integration` / 完整移植单元测试，另含集成测试

### CLI changes / CLI 变更

- Added `--help` (per-command help), `--version`, and `--key=value` flag syntax / 新增 `--help`（子命令帮助）、`--version`、`--key=value` 参数形式
- Reworked help output: shows both `download` and `compose` commands with their full option list / 重构帮助输出，完整展示两个子命令及选项
- `--platform` now applies to both `download` and `compose` (default: `all` architectures) / `--platform` 对 `download` 和 `compose` 均可用（默认所有架构）

### Fixes / 修复

- Fixed argument parsing when flags appear before the command / 修复 flag 出现在命令前时的参数解析
- Fixed a race when copying `skopeo.exe` on Windows (now a synchronous copy) / 修复 Windows 下复制 `skopeo.exe` 的竞态（改为同步复制）
- Replaced manual `which`/`where` detection with `exec.LookPath` / 用 `exec.LookPath` 替代手写 `which`/`where` 探测
- Unified error output to stderr with consistent exit codes / 统一错误输出到 stderr 及一致的退出码

### Breaking changes / 破坏性变更

- Building from source now requires Go 1.25+ (previously Bun) / 从源码构建现在需要 Go 1.25+（原为 Bun）
- Legacy PowerShell scripts removed / 移除旧版 PowerShell 脚本

### Upgrading / 升级指引

```bash
go build -o skopeo-cli .
# or with Task installed / 或使用 Task:
task build
```
