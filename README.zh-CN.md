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

## 前置要求

- [Bun](https://bun.sh) 运行时
- [skopeo](https://github.com/containers/skopeo) CLI

## 安装

```bash
bun install
```

## 开发

```bash
# 直接运行
bun run start

# 或者
bun run skopeo.ts
```

## 构建

```bash
bun run build
```

将 `skopeo.ts` 编译为独立可执行文件 `skopeo`（Windows 上为 `skopeo.exe`）。

### 跨平台构建

项目使用 GitHub Actions 为多个平台构建。推送标签触发发布：

```bash
git tag v1.0.0
git push origin v1.0.0
```

自动构建并发布以下平台的二进制文件：
- Windows (x86_64)
- Linux (x86_64)
- macOS (x86_64)

## 使用方法

### 下载单个镜像

```bash
./skopeo download <image> [options]
```

**选项：**
- `--save <path>` — 保存目录（默认：`~/Downloads/docker`）
- `--platform <platform>` — 目标平台（如 `linux/amd64`）
- `--overwrite` — 覆盖已有文件
- `--no-upload-script` — 跳过生成上传脚本

**示例：**
```bash
./skopeo download nginx:latest --save ./images
```

### 批量下载

```bash
./skopeo compose <file> [options]
```

**选项：**
- `--save <path>` — 保存目录（默认：`~/Downloads/docker`）
- `--filter <image>` — 筛选特定镜像下载
- `--overwrite` — 覆盖已有文件
- `--no-upload-script` — 跳过生成上传脚本

**示例：**
```bash
./skopeo compose docker-compose.yml --save ./images
```

## 上传脚本

下载完成后会生成 `upload_all.ps1` PowerShell 脚本，可将所有镜像推送到私有仓库（如 `docker.senjone.com`）。

## 旧版脚本

`legacy/` 目录包含本工具替代的原始 PowerShell 脚本：

- `skopeo-download.ps1` — 原下载脚本
- `skopeo-compose.ps1` — 原合成脚本
- `skopeo-common.ps1` — 共享工具函数

仅供参考，不再维护。

## 许可证

MIT
