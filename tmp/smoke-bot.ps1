[CmdletBinding()]
param(
    [string]$Profile = "qa-dev",
    [Parameter(Mandatory = $true)]
    [string]$Peer,
    [int]$TimeoutSec = 60,
    [int]$ReadLimit = 5,
    [string]$Text,
    [switch]$SkipPull,
    [switch]$SkipBuild,
    [switch]$SkipAuthCheck,
    [switch]$MarkRead,
    [switch]$IncludePhoto,
    [string]$PhotoFile,
    [string]$PhotoCaption
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

. (Join-Path $PSScriptRoot "_mi_telegram_common.ps1")

if ($TimeoutSec -lt 1 -or $TimeoutSec -gt 300) {
    throw "TimeoutSec must be between 1 and 300."
}

Initialize-MiTelegramCli -SkipPull:$SkipPull -SkipBuild:$SkipBuild | Out-Null

if (-not $SkipAuthCheck) {
    Assert-MiTelegramAuthorized -Profile $Profile
}

$messageText = if ([string]::IsNullOrWhiteSpace($Text)) {
    "smoke-" + [Guid]::NewGuid().ToString("N").Substring(0, 10)
} else {
    $Text
}

Write-Step "messages send"
$send = Invoke-MiTelegramCliJson -CommandArgs @("messages", "send", "--profile", $Profile, "--peer", $Peer, "--text", $messageText, "--json") -ProfileLockRetries 3
$send.Json | ConvertTo-Json -Depth 20

$messageId = [int64]$send.Json.data.messageId

Write-Step "messages wait"
$wait = Invoke-MiTelegramCliJson -CommandArgs @("messages", "wait", "--profile", $Profile, "--peer", $Peer, "--after-id", $messageId.ToString(), "--timeout", $TimeoutSec.ToString(), "--json") -ProfileLockRetries 3
$wait.Json | ConvertTo-Json -Depth 20

Write-Step "messages read"
$read = Invoke-MiTelegramCliJson -CommandArgs @("messages", "read", "--profile", $Profile, "--peer", $Peer, "--limit", $ReadLimit.ToString(), "--after-id", $messageId.ToString(), "--json") -ProfileLockRetries 3
$read.Json | ConvertTo-Json -Depth 20

if ($MarkRead) {
    Write-Step "dialogs mark-read"
    $markRead = Invoke-MiTelegramCliJson -CommandArgs @("dialogs", "mark-read", "--profile", $Profile, "--peer", $Peer, "--json") -ProfileLockRetries 3
    $markRead.Json | ConvertTo-Json -Depth 20
}

$photoMessageId = $null
$photoSha256 = $null
if ($IncludePhoto) {
    if ([string]::IsNullOrWhiteSpace($PhotoFile)) {
        throw "-PhotoFile is required when -IncludePhoto is set."
    }
    $resolvedPhoto = (Resolve-Path -LiteralPath $PhotoFile).Path

    Write-Step "messages send-photo"
    $photoArgs = @("messages", "send-photo", "--profile", $Profile, "--peer", $Peer, "--file", $resolvedPhoto, "--json")
    if (-not [string]::IsNullOrWhiteSpace($PhotoCaption)) {
        $photoArgs += @("--caption", $PhotoCaption)
    }
    $photo = Invoke-MiTelegramCliJson -CommandArgs $photoArgs -ProfileLockRetries 3
    $photo.Json | ConvertTo-Json -Depth 20
    $photoMessageId = [int64]$photo.Json.data.messageId
    $photoSha256 = [string]$photo.Json.data.media.sha256
}

Write-Step "summary"
[pscustomobject]@{
    profile        = $Profile
    peer           = $Peer
    sentText       = $messageText
    messageId      = $messageId
    photoMessageId = $photoMessageId
    photoSha256    = $photoSha256
} | ConvertTo-Json -Depth 10
