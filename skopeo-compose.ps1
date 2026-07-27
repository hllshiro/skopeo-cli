Param(
    # 必填参数：Compose 文件路径
    [Parameter(Mandatory=$true, Position=0)]
    [string]$ComposeFile,

    # 可选参数：保存路径（默认为 $null）
    [Parameter(Mandatory=$false, Position=1)]
    [string]$SavePath = $null,

    # 可选参数：过滤器
    [Parameter(Mandatory=$false, Position=2)]
    [Alias("f")]
    [string]$Filter = $null
)

# 加载公共模块
. (Join-Path $PSScriptRoot "skopeo-common.ps1")

# 1. 检查 Compose 文件是否存在
Test-FileExists $ComposeFile "错误: 找不到文件 $ComposeFile"

# 2. 确定下载脚本位置
$DownloadScript = Join-Path $PSScriptRoot "skopeo-download.ps1"
Test-FileExists $DownloadScript "错误: 找不到下载脚本 $DownloadScript"

# 3. 解析 Compose 文件
Write-ColorMessage "正在解析 $ComposeFile ..." -Color Cyan
$content = Get-Content -Path $ComposeFile
$allImages = @()

foreach ($line in $content)
{
    if ($line -match '^\s*image:\s*(?<image>[^\s]+)')
    {
        $img = $matches['image'] -replace "['`"]", ""
        $allImages += $img
    }
}

$allImages = $allImages | Select-Object -Unique

if ($allImages.Count -eq 0)
{
    Write-ColorMessage "未在 $ComposeFile 中找到任何镜像配置。" -Color Yellow
    exit 0
}

# 4. 应用过滤器 (Filter)
$targetImages = $allImages
if (-not [string]::IsNullOrWhiteSpace($Filter))
{
    Write-ColorMessage "应用过滤器: *$Filter*" -Color Magenta
    $targetImages = $allImages | Where-Object { $_ -like "*$Filter*" }
}

if ($targetImages.Count -eq 0)
{
    Write-ColorMessage "经过过滤后，没有匹配的镜像需要下载。" -Color Yellow
    exit 0
}

Write-ColorMessage "找到 $($targetImages.Count) 个匹配镜像，开始下载..." -Color Cyan
Write-Host "=================================================="

# 5. 循环调用下载脚本
foreach ($imageName in $targetImages)
{
    Write-ColorMessage "[解析器] 准备处理: $imageName" -Color Cyan

    # 核心修复：根据 SavePath 是否为空，动态构建参数数组
    $scriptArgs = @($imageName)
    if (-not [string]::IsNullOrWhiteSpace($SavePath))
    {
        $scriptArgs += $SavePath
    }

    # 使用 @scriptArgs 这种“飞溅 (Splatting)”方式传递参数，确保位置正确
    & $DownloadScript @scriptArgs

    if ($LASTEXITCODE -ne 0)
    {
        Write-ColorMessage "[解析器] 警告: $imageName 下载失败。" -Color Yellow
    }
    Write-Host "--------------------------------------------------"
}

Write-ColorMessage "所有镜像调用任务结束！" -Color Green
exit 0
