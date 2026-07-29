[![English](https://img.shields.io/badge/English-readme-blue)](README.md)

# skopeo-cli

一个使用 [skopeo](https://github.com/containers/skopeo) 下载和合成 Docker 容器镜像的命令行工具。

## 项目背景

本工具专为离线/内网环境设计，用于从公共仓库（Docker Hub、GitHub Container Registry 等）下载容器镜像，然后上传到内网的离线仓库。支持多架构镜像（linux/amd64、linux/arm64 等），适用于混合架构基础设施环境。

推荐使用 [zot](https://github.com/project-zot/zot) 搭建私有容器仓库 —— 一个原生 OCI 兼容的仓库，资源占用极低。

## 功能特性

- **下载** 单个 Docker 镜像
- **合成** 从 compose 文件批量下载多个镜像
- 生成 PowerShell 上传脚本，推送到私有仓库
- 多架构支持 (linux/amd64, linux/arm64 等)
- 通过 `--registry` 选项自定义目标仓库地址
- 便携式输出目录 (`docker-image-will-upload/`)，Windows 下自动包含 skopeo 二进制文件
- 跨平台 skopeo 检测与可用性检查

## 前置要求

- [Bun](https://bun.sh) 运行时（从源码构建时需要）
- [skopeo](https://github.com/containers/skopeo) CLI

## 安装

### 预编译二进制文件

从 [Releases](https://github.com/hllshiro/skopeo-cli/releases) 下载适合您平台的最新版本。

### 从源码构建

```bash
bun install
bun run build
```

## 使用方法

### 下载单个镜像

```bash
./skopeo-cli download <image> [options]
```

**选项：**
- `--save <path>` — 保存目录（默认：`~/Downloads`）
- `--platform <platform>` — 目标平台（如 `linux/amd64`）
- `--registry <host>` — 目标仓库地址（默认：`docker.senjone.com`）
- `--overwrite` — 覆盖已有文件
- `--no-upload-script` — 跳过生成上传脚本

**示例：**
```bash
./skopeo-cli download nginx:latest --save ./images --registry my-registry.example.com
```

### 批量下载

```bash
./skopeo-cli compose <file> [options]
```

**选项：**
- `--save <path>` — 保存目录（默认：`~/Downloads`）
- `--filter <image>` — 筛选特定镜像下载
- `--registry <host>` — 目标仓库地址（默认：`docker.senjone.com`）
- `--overwrite` — 覆盖已有文件
- `--no-upload-script` — 跳过生成上传脚本

**示例：**
```bash
./skopeo-cli compose docker-compose.yml --save ./images --registry gcr.io/my-project
```

## 输出目录结构

所有下载的镜像、脚本以及（Windows 下的）skopeo 二进制文件都放在保存路径下的 `docker-image-will-upload/` 目录中：

```
~/Downloads/docker-image-will-upload/
├── nginx-latest.tar
├── ubuntu-latest.tar
├── upload_all.ps1
└── skopeo.exe          # 仅 Windows
```

这样方便将整个包迁移到其他机器上使用。

## 上传脚本

下载完成后会生成 `upload_all.ps1` PowerShell 脚本，可将所有镜像推送到配置的目标仓库。脚本使用相对路径并包含 `Set-Location $PSScriptRoot`，可以从任意位置运行。

## 跨平台构建

项目使用 GitHub Actions 为多个平台构建。推送标签触发发布：

```bash
git tag v1.1.0
git push origin v1.1.0
```

自动构建并发布以下平台的二进制文件：
- Windows (x86_64)
- Linux (x86_64)
- macOS (x86_64)

## 旧版脚本

`legacy/` 目录包含本工具替代的原始 PowerShell 脚本：

- `skopeo-download.ps1` — 原下载脚本
- `skopeo-compose.ps1` — 原合成脚本
- `skopeo-common.ps1` — 共享工具函数

仅供参考，不再维护。

## 许可证

MIT
