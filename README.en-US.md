[![简体中文](https://img.shields.io/badge/简体中文-readme-blue)](README.md)

# skopeo-cli

A CLI tool for downloading and composing Docker container images using [skopeo](https://github.com/containers/skopeo).

## Background

This tool is designed for air-gapped environments where you need to download container images from public registries (Docker Hub, GitHub Container Registry, etc.) and upload them to an offline repository on your intranet. It supports multi-architecture images (linux/amd64, linux/arm64, etc.), making it ideal for hybrid infrastructure environments.

For setting up a private container registry, we recommend [zot](https://github.com/project-zot/zot) — a native, OCI-compatible registry with minimal resource requirements.

When your private registry uses plain HTTP, skopeo will print a warning. Follow the hint and create a `policy.json` file at the indicated location (usually `/etc/containers/policy.json` on Linux) with the following content:

```json
{"default": [{"type":"insecureAcceptAnything"}]}
```

## Features

- **Download** single Docker images from registries
- **Compose** batch download multiple images from a compose file (with `${VAR}` interpolation and `.env` support)
- Generates cross-platform upload scripts (`upload_all.sh` / `upload_all.ps1`) for pushing to private registries
- Multi-architecture support (linux/amd64, linux/arm64, etc.)
- Configurable target registry via `--registry` option
- Portable output directory (`docker-image-will-upload/`) with whitelisted skopeo binary bundled on all platforms
- Cross-platform skopeo detection and availability check

## Prerequisites

- [Go](https://go.dev) 1.25+ (for building from source)
- [skopeo](https://github.com/containers/skopeo) CLI installed (prebuilt Windows executables: [winskopeo](https://github.com/passcod/winskopeo))

### Skopeo Version Requirements

**Minimum version: 1.6.0+**

This tool requires skopeo with support for `--all` flag (multi-architecture image copying). Versions below 1.6.0 may fail with:

```
unsupported MIME type for compression: application/vnd.in-toto+json
```

The version is also checked automatically at runtime; versions below 1.6.0 are rejected with a hint.

**Tested versions:**
- 1.23.0 (recommended)
- 1.14.0+
- 1.6.0+

## Installation

### Pre-built Binaries

Download the latest release for your platform from [Releases](https://github.com/hllshiro/skopeo-cli/releases).

### From Source

```bash
go build -o skopeo-cli .
```

With [Task](https://taskfile.dev) installed, common commands are available as tasks:

```bash
task cli -- download nginx:latest   # run from source
task build                          # build for current platform
task build:all                      # build for all platforms
task test                           # run tests
```

## Usage

### Download a single image

```bash
./skopeo-cli download <image> [options]
```

**Options:**
- `--save <path>` — Save directory (default: `~/Downloads`)
- `--platform <os/arch>` — Target platform (default: all, e.g. `linux/amd64`)
- `--registry <host>` — Target registry (default: `docker.senjone.com`)
- `--overwrite` — Overwrite existing files
- `--no-upload-script` — Skip generating upload script

**Example:**
```bash
./skopeo-cli download nginx:latest --save ./images --registry my-registry.example.com
```

### Compose batch download

```bash
./skopeo-cli compose <file> [options]
```

**Options:**
- `--save <path>` — Save directory (default: `~/Downloads`)
- `--platform <os/arch>` — Target platform (default: all, e.g. `linux/amd64`)
- `--filter <image>` — Filter specific image to download
- `--registry <host>` — Target registry (default: `docker.senjone.com`)
- `--overwrite` — Overwrite existing files
- `--no-upload-script` — Skip generating upload script

**Example:**
```bash
./skopeo-cli compose docker-compose.yml --save ./images --registry gcr.io/my-project
```

`<file>` may be a file path or a directory (standard names `compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml` are looked up inside). `${VAR}`, `${VAR:-default}` and `${VAR-default}` in image references are resolved from the environment first, then a `.env` file next to the compose file.

## Output Structure

All downloaded images, scripts, and the skopeo binary are placed in a `docker-image-will-upload/` directory under the save path:

```
~/Downloads/docker-image-will-upload/
├── nginx-latest.tar
├── ubuntu-latest.tar
├── upload_all.sh       # POSIX sh, for Linux / macOS
├── upload_all.ps1      # PowerShell, for Windows
├── skopeo              # bundled when whitelist matches (Unix)
└── skopeo.exe          # bundled when whitelist matches (Windows)
```

The skopeo binary is bundled via a whitelist (`skopeo`, `skopeo.exe`): files in the directory of the resolved skopeo executable whose names match the whitelist are copied into the output directory. If nothing matches, bundling is skipped and an info line is logged; the upload scripts then fall back to skopeo on PATH.

This makes it easy to migrate the entire package to another machine.

## Generated Upload Scripts

After downloading images, two upload scripts are generated to push all images to the configured registry:

- `upload_all.sh` — POSIX sh script, for Linux / macOS;
- `upload_all.ps1` — PowerShell script, for Windows.

Both scripts share the same image list and prefer a bundled skopeo binary next to the script, falling back to skopeo on PATH. They use relative paths and can be run from any location.

## Cross-Platform Builds

This project uses GitHub Actions to build for multiple platforms. Push a tag to trigger a release:

```bash
git tag v1.1.0
git push origin v1.1.0
```

This will automatically build and publish binaries for:
- Windows (x86_64, arm64)
- Linux (x86_64, arm64)
- macOS (x86_64, arm64)

## License

MIT
