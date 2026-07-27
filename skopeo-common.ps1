# skopeo-common.ps1 - 公共函数模块
# 提供两个脚本共享的工具函数

function Write-ColorMessage {
    <#
    .SYNOPSIS
        统一的彩色输出函数
    #>
    param(
        [string]$Message,
        [ValidateSet("Red", "Yellow", "Green", "Cyan", "Magenta", "White")]
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-FileExists {
    <#
    .SYNOPSIS
        检查文件是否存在，不存在则输出错误并退出
    #>
    param(
        [string]$FilePath,
        [string]$ErrorMessage
    )
    
    if (-not (Test-Path -Path $FilePath)) {
        Write-ColorMessage $ErrorMessage -Color Red
        exit 1
    }
    return $true
}

function Ensure-Directory {
    <#
    .SYNOPSIS
        确保目录存在，不存在则创建
    #>
    param(
        [string]$Path
    )
    
    if (-not (Test-Path -Path $Path)) {
        New-Item -ItemType Directory -Path $Path -Force | Out-Null
    }
}

function Exit-WithError {
    <#
    .SYNOPSIS
        输出错误消息并退出
    #>
    param(
        [string]$Message,
        [int]$ExitCode = 1
    )
    
    Write-ColorMessage $Message -Color Red
    exit $ExitCode
}

function Exit-WithSuccess {
    <#
    .SYNOPSIS
        输出成功消息并退出
    #>
    param(
        [string]$Message
    )
    
    Write-ColorMessage $Message -Color Green
    exit 0
}

function Confirm-FileOverwrite {
    <#
    .SYNOPSIS
        确认是否覆盖已存在的文件
    #>
    param(
        [string]$FilePath
    )
    
    if (Test-Path -Path $FilePath) {
        $Confirmation = Read-Host "文件 $FilePath 已存在，是否覆盖? (y/N)"
        if ($Confirmation -notmatch '^y$') {
            Write-ColorMessage "操作已取消" -Color Cyan
            exit 0
        }
        Remove-Item -Path $FilePath -Force
        return $true
    }
    return $false
}
