## v2.0.0 — Complete rewrite in Go

skopeo-cli has been rewritten from Bun/TypeScript to Go. The CLI interface remains compatible with v1.x, with several improvements and fixes.

### Highlights

- **Go rewrite**: no third-party dependencies, statically linked binaries, no runtime required
- **Simpler releases**: all platforms are cross-compiled from a single job (`linux`, `windows`, `macos` on `amd64`)
- **Taskfile**: `task build`, `task build:all`, `task test`, `task cli -- <args>`, and more
- **Tests**: full unit-test suite ported, plus integration tests behind `go test -tags integration`

### CLI changes

- Added `--help` (per-command help), `--version`, and `--key=value` flag syntax
- Reworked help output: shows both `download` and `compose` commands with their full option list
- `--platform` now applies to both `download` and `compose` (default: `all` architectures)

### Fixes

- Fixed argument parsing when flags appear before the command
- Fixed a race when copying `skopeo.exe` on Windows (now a synchronous copy)
- Replaced manual `which`/`where` detection with `exec.LookPath`
- Unified error output to stderr with consistent exit codes

### Breaking changes

- Building from source now requires Go 1.25+ (previously Bun)
- Legacy PowerShell scripts removed

### Upgrading

```bash
go build -o skopeo-cli .
# or with Task installed:
task build
```
