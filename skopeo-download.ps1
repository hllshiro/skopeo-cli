# 1. 设置默认保存路径 (用户下载目录下的 docker 文件夹)
$DEFAULT_SAVE_PATH = Join-Path $env:USERPROFILE "Downloads\docker"

# 2. 检查镜像名称参数
if ($args.Count -lt 1) {
    Write-Host "用法: .\download_image.ps1 <IMAGE> [PATH]" -ForegroundColor Yellow
    Write-Host "示例: .\download_image.ps1 nginx:latest"
    Write-Host "      .\download_image.ps1 ubuntu:22.04 D:\myimages"
    exit 1
}

# 3. 获取镜像名称和保存路径
$IMAGE_NAME = $args[0]
$SAVE_PATH = if ($args.Count -ge 2) { $args[1] } else { $DEFAULT_SAVE_PATH }

# 4. 验证并创建目标目录
if (-not (Test-Path -Path $SAVE_PATH)) {
    New-Item -ItemType Directory -Path $SAVE_PATH -Force | Out-Null
}

# 5. 如果镜像带 registry（例如 ghcr.io、registry:5000、localhost 等），去掉 registry 层
#    规则：如果第一个部分包含 '.' 或 ':' 或 等于 'localhost'，则认为是 registry
$parts = $IMAGE_NAME -split '/'
if ($parts.Length -gt 1 -and ($parts[0] -match '\.' -or $parts[0] -match ':' -or $parts[0] -eq 'localhost')) {
    # 重新组合第二部分及之后的 path（包含 tag）
    $REPO_PATH = ($parts[1..($parts.Length - 1)] -join '/')
} else {
    $REPO_PATH = $IMAGE_NAME
}

# 6. 生成文件名（基于去掉 registry 的 $REPO_PATH：替换 : 为 -、/ 为 _）
$FILE_NAME = $REPO_PATH -replace ':', '-' -replace '/', '_'
$ARCHIVE_FILE = Join-Path $SAVE_PATH "$FILE_NAME.tar"

# 7. 检查目标文件是否已存在
if (Test-Path -Path $ARCHIVE_FILE) {
    $Confirmation = Read-Host "文件 $ARCHIVE_FILE 已存在，是否覆盖? (y/N)"
    if ($Confirmation -notmatch '^y$') {
        Write-Host "操作已取消" -ForegroundColor Cyan
        exit 0
    }
    Remove-Item -Path $ARCHIVE_FILE -Force
}

# 8. 执行下载
Write-Host "正在下载镜像: $IMAGE_NAME..." -ForegroundColor Green
# 注意：请确保 skopeo 已在系统环境变量 PATH 中
skopeo copy --all "docker://$IMAGE_NAME" "oci-archive:$ARCHIVE_FILE"

if ($LASTEXITCODE -ne 0) {
    Write-Host "下载镜像失败！" -ForegroundColor Red
    if (Test-Path -Path $ARCHIVE_FILE) { Remove-Item -Path $ARCHIVE_FILE }
    exit 1
}

# 9. 定义生成的上传脚本文件名 (Windows 版生成 .ps1)
$UPLOAD_SCRIPT = Join-Path $SAVE_PATH "upload_all.ps1"

# 10. 将上传命令追加到脚本中（使用去掉 registry 的 $REPO_PATH）
$COMMAND = "skopeo copy --all `"oci-archive:$FILE_NAME.tar`" `"docker://docker.senjone.com/$REPO_PATH`""
Add-Content -Path $UPLOAD_SCRIPT -Value $COMMAND

Write-Host "已记录: $IMAGE_NAME -> docker.senjone.com/$REPO_PATH" -ForegroundColor Green
exit 0