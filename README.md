# skopeo

A CLI tool for downloading and composing Docker container images using [skopeo](https://github.com/containers/skopeo).

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

## License

MIT
