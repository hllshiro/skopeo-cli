# 加载公共模块
. (Join-Path $PSScriptRoot "skopeo-common.ps1")

# 1. 设置默认保存路径 (用户下载目录下的 docker 文件夹)
$DEFAULT_SAVE_PATH = Join-Path $env:USERPROFILE "Downloads\docker"

# 2. 检查镜像名称参数
if ($args.Count -lt 1) {
    Write-ColorMessage "用法: .\download_image.ps1 <IMAGE> [PATH]" -Color Yellow
    Write-ColorMessage "示例: .\download_image.ps1 nginx:latest"
    Write-ColorMessage "      .\download_image.ps1 ubuntu:22.04 D:\myimages"
    exit 1
}

# 3. 获取镜像名称和保存路径
$IMAGE_NAME = $args[0]
$SAVE_PATH = if ($args.Count -ge 2) { $args[1] } else { $DEFAULT_SAVE_PATH }

# 4. 验证并创建目标目录
Ensure-Directory $SAVE_PATH

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
Confirm-FileOverwrite $ARCHIVE_FILE | Out-Null

# 8. 执行下载
Write-ColorMessage "正在下载镜像: $IMAGE_NAME..." -Color Green
# 注意：请确保 skopeo 已在系统环境变量 PATH 中
skopeo copy --all "docker://$IMAGE_NAME" "oci-archive:$ARCHIVE_FILE"

if ($LASTEXITCODE -ne 0) {
    Write-ColorMessage "下载镜像失败！" -Color Red
    if (Test-Path -Path $ARCHIVE_FILE) { Remove-Item -Path $ARCHIVE_FILE }
    exit 1
}

# 9. 定义生成的上传脚本文件名 (Windows 版生成 .ps1)
$UPLOAD_SCRIPT = Join-Path $SAVE_PATH "upload_all.ps1"

# 10. 如果上传脚本不存在，添加计数器初始化代码
if (-not (Test-Path -Path $UPLOAD_SCRIPT)) {
    $InitCode = @'
# 上传计数器
$script:TotalUploaded = 0
'@
    Set-Content -Path $UPLOAD_SCRIPT -Value $InitCode
}

# 11. 将上传命令追加到脚本中（使用去掉 registry 的 $REPO_PATH）
$ProgressLog = "Write-Host `"[$($script:TotalUploaded + 1)] 正在上传: $FILE_NAME.tar ...`" -ForegroundColor Cyan"
$COMMAND = "skopeo copy --all `"oci-archive:$FILE_NAME.tar`" `"docker://docker.senjone.com/$REPO_PATH`""
$IncrementCounter = '$script:TotalUploaded++'
Add-Content -Path $UPLOAD_SCRIPT -Value @($ProgressLog, $COMMAND, $IncrementCounter, "")

Write-ColorMessage "已记录: $IMAGE_NAME -> docker.senjone.com/$REPO_PATH" -Color Green
exit 0