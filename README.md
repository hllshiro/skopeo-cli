[![简体中文](https://img.shields.io/badge/简体中文-readme-blue)](README.zh-CN.md)

# skopeo-cli

A CLI tool for downloading and composing Docker container images using [skopeo](https://github.com/containers/skopeo).

## Background

This tool is designed for air-gapped environments where you need to download container images from public registries (Docker Hub, GitHub Container Registry, etc.) and upload them to an offline repository on your intranet. It supports multi-architecture images (linux/amd64, linux/arm64, etc.), making it ideal for hybrid infrastructure environments.

For setting up a private container registry, we recommend [zot](https://github.com/project-zot/zot) — a native, OCI-compatible registry with minimal resource requirements.

## Features

- **Download** single Docker images from registries
- **Compose** batch download multiple images from a compose file
- Generates PowerShell upload scripts for pushing to private registries
- Multi-architecture support (linux/amd64, linux/arm64, etc.)
- Configurable target registry via `--registry` option
- Portable output directory (`docker-image-will-upload/`) with skopeo binary included on Windows
- Cross-platform skopeo detection and availability check

## Prerequisites

- [Go](https://go.dev) 1.25+ (for building from source)
- [skopeo](https://github.com/containers/skopeo) CLI installed

### Skopeo Version Requirements

**Minimum version: 1.6.0+**

This tool requires skopeo with support for `--all` flag (multi-architecture image copying). Versions below 1.6.0 may fail with:

```
unsupported MIME type for compression: application/vnd.in-toto+json
```

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
- `--platform <platform>` — Target platform (e.g., `linux/amd64`)
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
- `--filter <image>` — Filter specific image to download
- `--registry <host>` — Target registry (default: `docker.senjone.com`)
- `--overwrite` — Overwrite existing files
- `--no-upload-script` — Skip generating upload script

**Example:**
```bash
./skopeo-cli compose docker-compose.yml --save ./images --registry gcr.io/my-project
```

## Output Structure

All downloaded images, scripts, and (on Windows) the skopeo binary are placed in a `docker-image-will-upload/` directory under the save path:

```
~/Downloads/docker-image-will-upload/
├── nginx-latest.tar
├── ubuntu-latest.tar
├── upload_all.ps1
└── skopeo.exe          # Windows only
```

This makes it easy to migrate the entire package to another machine.

## Generated Upload Script

After downloading images, an `upload_all.ps1` PowerShell script is generated that can push all downloaded images to the configured registry. The script uses relative paths and includes `Set-Location $PSScriptRoot` so it can be run from any location.

## Cross-Platform Builds

This project uses GitHub Actions to build for multiple platforms. Push a tag to trigger a release:

```bash
git tag v1.1.0
git push origin v1.1.0
```

This will automatically build and publish binaries for:
- Windows (x86_64)
- Linux (x86_64)
- macOS (x86_64)

## License

MIT
