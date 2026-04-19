[CmdletBinding()]
param(
    [string]$Profile = "qa-dev",
    [string]$DisplayName = "QA Dev",
    [int]$DialogsLimit = 10,
    [switch]$SkipPull,
    [switch]$SkipBuild,
    [switch]$SkipDialogs,
    [switch]$ForceLogin
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "_mi_telegram_common.ps1")

Initialize-MiTelegramCli -SkipPull:$SkipPull -SkipBuild:$SkipBuild | Out-Null
Ensure-MiTelegramProfile -Profile $Profile -DisplayName $DisplayName

$authorizationStatus = Get-MiTelegramAuthStatus -Profile $Profile
if ($ForceLogin -or $authorizationStatus -ne "Authorized") {
    $binary = Get-MiTelegramBinaryPath
    $loginSucceeded = $false
    for ($attempt = 1; $attempt -le 5; $attempt++) {
        Write-Step "QR login for profile $Profile (attempt $attempt/5)"
        $loginOutput = @()
        & $binary auth login --profile $Profile --method qr 2>&1 | Tee-Object -Variable loginOutput | Out-Host
        if ($LASTEXITCODE -eq 0) {
            $loginSucceeded = $true
            break
        }

        $rawLoginOutput = ($loginOutput | Out-String).Trim()
        if ($rawLoginOutput -match 'ProfileLocked' -and $attempt -lt 5) {
            Write-Host "Profile lock is busy; waiting before retry..." -ForegroundColor Yellow
            Start-Sleep -Seconds 2
            continue
        }

        throw "QR login failed with exit code $LASTEXITCODE"
    }

    if (-not $loginSucceeded) {
        throw "QR login could not acquire the profile lock after multiple attempts."
    }
} else {
    Write-Step "Profile $Profile is already authorized"
}

Write-Step "auth status"
$status = Invoke-MiTelegramCliJson -CommandArgs @("auth", "status", "--profile", $Profile, "--json") -ProfileLockRetries 3
$status.Json | ConvertTo-Json -Depth 20

Write-Step "me"
$me = Invoke-MiTelegramCliJson -CommandArgs @("me", "--profile", $Profile, "--json") -ProfileLockRetries 3
$me.Json | ConvertTo-Json -Depth 20

if (-not $SkipDialogs) {
    Write-Step "dialogs list"
    $dialogs = Invoke-MiTelegramCliJson -CommandArgs @("dialogs", "list", "--profile", $Profile, "--limit", $DialogsLimit.ToString(), "--json") -ProfileLockRetries 3
    $dialogs.Json | ConvertTo-Json -Depth 20
}
