## v2.1.0 — Robust compose parsing and reliability fixes / 更健壮的 compose 解析与可靠性修复

### Highlights / 亮点

- **Real YAML compose parsing / 真正的 YAML compose 解析**: compose files are now parsed with `gopkg.in/yaml.v3` instead of regex scanning — comments, anchors, merge keys and multi-line values are handled correctly / compose 文件改用 `gopkg.in/yaml.v3` 完整解析，替代正则扫描——正确处理注释、锚点、合并键与多行值
- **Variable interpolation / 变量插值**: `${VAR}`, `${VAR:-default}`, `${VAR-default}` and `$VAR` are resolved from the process environment, then a `.env` file next to the compose file / `${VAR}`、`${VAR:-default}`、`${VAR-default}`、`$VAR` 从环境变量及 compose 同目录 `.env` 文件解析
- **More platforms / 更多平台**: CI now publishes `linux`, `windows`, `macos` for both `amd64` and `arm64` / CI 现发布 linux、windows、macos 的 amd64 与 arm64 双架构
- **skopeo version gate / skopeo 版本门槛**: the tool now checks `skopeo --version` and refuses versions below 1.6.0 / 工具现在会校验 `skopeo --version`，拒绝低于 1.6.0 的版本

### Changes / 变更

- `compose` accepts a directory or standard file names (`compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`) / `compose` 支持传入目录或标准文件名
- Errors are consistently written to stderr with non-zero exit codes on any download failure / 错误统一输出到 stderr，任一镜像下载失败即返回非零退出码
- The unused `NoUploadScript` field was removed from the download options / 移除下载选项中未使用的 `NoUploadScript` 字段

### Fixes / 修复

- Compose files with `${VAR}` image references no longer fail (previously downloaded the literal template) / 含 `${VAR}` 变量引用的 compose 文件不再下载失败（此前会按字面模板下载）
- `image:` inside YAML comments is no longer extracted / 不再误提取 YAML 注释中的 `image:`
- Download failures now surface in the exit code, so scripts and CI can detect partial failures / 下载失败现在会反映到退出码，脚本与 CI 可感知部分失败

### Upgrading / 升级指引

```bash
go build -o skopeo-cli .
# or with Task installed / 或使用 Task:
task build
```

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
