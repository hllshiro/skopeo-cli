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

## Prerequisites

- [Bun](https://bun.sh) runtime
- [skopeo](https://github.com/containers/skopeo) CLI installed

## Installation

```bash
bun install
```

## Development

```bash
# Run directly
bun run start

# Or
bun run skopeo.ts
```

## Build

```bash
bun run build
```

This compiles `skopeo.ts` into a standalone executable named `skopeo` (or `skopeo.exe` on Windows).

### Cross-Platform Builds

This project uses GitHub Actions to build for multiple platforms. Push a tag to trigger a release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will automatically build and publish binaries for:
- Windows (x86_64)
- Linux (x86_64)
- macOS (x86_64)

## Usage

### Download a single image

```bash
./skopeo download <image> [options]
```

**Options:**
- `--save <path>` — Save directory (default: `~/Downloads/docker`)
- `--platform <platform>` — Target platform (e.g., `linux/amd64`)
- `--overwrite` — Overwrite existing files
- `--no-upload-script` — Skip generating upload script

**Example:**
```bash
./skopeo download nginx:latest --save ./images
```

### Compose batch download

```bash
./skopeo compose <file> [options]
```

**Options:**
- `--save <path>` — Save directory (default: `~/Downloads/docker`)
- `--filter <image>` — Filter specific image to download
- `--overwrite` — Overwrite existing files
- `--no-upload-script` — Skip generating upload script

**Example:**
```bash
./skopeo compose docker-compose.yml --save ./images
```

## Generated Upload Script

After downloading images, an `upload_all.ps1` PowerShell script is generated that can push all downloaded images to a private registry (e.g., `docker.senjone.com`).

## Legacy Scripts

The `legacy/` directory contains the original PowerShell scripts that this tool replaces:

- `skopeo-download.ps1` — Original download script
- `skopeo-compose.ps1` — Original compose script
- `skopeo-common.ps1` — Shared utilities

These are kept for reference only and are no longer maintained.

## License

MIT
