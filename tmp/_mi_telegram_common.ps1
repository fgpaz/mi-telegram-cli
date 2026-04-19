[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-MiTelegramRepoRoot {
    return (Split-Path -Parent $PSScriptRoot)
}

function Get-MiTelegramMkeyPath {
    return (Join-Path $HOME ".agents\skills\mi-key-cli\scripts\mkey.ps1")
}

function Get-MiTelegramEnvPath {
    return (Join-Path (Get-MiTelegramRepoRoot) "infra\.env")
}

function Get-MiTelegramBinaryPath {
    return (Join-Path (Get-MiTelegramRepoRoot) "bin\mi-telegram-cli.exe")
}

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Invoke-MiTelegramPull {
    param([switch]$SkipPull)

    if ($SkipPull) {
        return
    }

    $mkey = Get-MiTelegramMkeyPath
    if (-not (Test-Path $mkey)) {
        throw "mi-key-cli not found at $mkey"
    }

    Write-Step "Pulling secrets with mkey"
    pwsh -File $mkey pull mi-telegram-cli dev
    if ($LASTEXITCODE -ne 0) {
        throw "mkey pull mi-telegram-cli dev failed with exit code $LASTEXITCODE"
    }
}

function Import-MiTelegramEnv {
    $envPath = Get-MiTelegramEnvPath
    if (-not (Test-Path $envPath)) {
        throw "Expected env file at $envPath. Run mkey pull first."
    }

    foreach ($line in Get-Content $envPath) {
        if ($line -match '^\s*$' -or $line -match '^\s*#') {
            continue
        }

        $parts = $line -split '=', 2
        if ($parts.Count -ne 2) {
            continue
        }

        [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1], "Process")
    }
}

function Ensure-MiTelegramBinary {
    param([switch]$SkipBuild)

    $binary = Get-MiTelegramBinaryPath
    if ($SkipBuild -and (Test-Path $binary)) {
        return $binary
    }

    if ((-not $SkipBuild) -or (-not (Test-Path $binary))) {
        $repoRoot = Get-MiTelegramRepoRoot
        $binDir = Split-Path -Parent $binary
        if (-not (Test-Path $binDir)) {
            New-Item -ItemType Directory -Path $binDir -Force | Out-Null
        }

        Write-Step "Building mi-telegram-cli"
        Push-Location $repoRoot
        try {
            go build -o $binary .\cmd\mi-telegram-cli
            if ($LASTEXITCODE -ne 0) {
                throw "go build failed with exit code $LASTEXITCODE"
            }
        } finally {
            Pop-Location
        }
    }

    return $binary
}

function Initialize-MiTelegramCli {
    param(
        [switch]$SkipPull,
        [switch]$SkipBuild
    )

    Invoke-MiTelegramPull -SkipPull:$SkipPull
    Import-MiTelegramEnv
    return (Ensure-MiTelegramBinary -SkipBuild:$SkipBuild)
}

function Invoke-MiTelegramCliCapture {
    param(
        [string[]]$CommandArgs,
        [switch]$AllowFailure,
        [int]$ProfileLockRetries = 0,
        [int]$ProfileLockDelaySeconds = 2
    )

    $binary = Get-MiTelegramBinaryPath
    if (-not (Test-Path $binary)) {
        throw "Binary not found at $binary. Run Initialize-MiTelegramCli first."
    }

    for ($attempt = 0; $attempt -le $ProfileLockRetries; $attempt++) {
        $output = & $binary @CommandArgs 2>&1 | Out-String
        $exitCode = $LASTEXITCODE
        $trimmed = $output.Trim()

        $isProfileLocked = $trimmed -match 'ProfileLocked'
        if ($exitCode -ne 0 -and $isProfileLocked -and $attempt -lt $ProfileLockRetries) {
            Start-Sleep -Seconds $ProfileLockDelaySeconds
            continue
        }

        if ((-not $AllowFailure) -and $exitCode -ne 0) {
            throw "mi-telegram-cli failed ($exitCode): $trimmed"
        }

        return @{
            ExitCode = $exitCode
            Output   = $trimmed
        }
    }
}

function Invoke-MiTelegramCliJson {
    param(
        [string[]]$CommandArgs,
        [switch]$AllowFailure
    )

    $result = Invoke-MiTelegramCliCapture -CommandArgs $CommandArgs -AllowFailure:$AllowFailure
    if ([string]::IsNullOrWhiteSpace($result.Output)) {
        return @{
            ExitCode = $result.ExitCode
            Json     = $null
            Raw      = $result.Output
        }
    }

    try {
        $json = $result.Output | ConvertFrom-Json -Depth 20
    } catch {
        throw "Expected JSON output for args '$($CommandArgs -join ' ')', got: $($result.Output)"
    }

    return @{
        ExitCode = $result.ExitCode
        Json     = $json
        Raw      = $result.Output
    }
}

function Ensure-MiTelegramProfile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Profile,
        [Parameter(Mandatory = $true)]
        [string]$DisplayName
    )

    $show = Invoke-MiTelegramCliJson -CommandArgs @("profiles", "show", "--profile", $Profile, "--json") -AllowFailure -ProfileLockRetries 3
    if ($show.ExitCode -eq 0) {
        return
    }

    Write-Step "Creating profile $Profile"
    $create = Invoke-MiTelegramCliJson -CommandArgs @("profiles", "add", "--profile", $Profile, "--display-name", $DisplayName, "--json") -ProfileLockRetries 3
    $create.Json | ConvertTo-Json -Depth 20
}

function Get-MiTelegramAuthStatus {
    param([Parameter(Mandatory = $true)][string]$Profile)

    $status = Invoke-MiTelegramCliJson -CommandArgs @("auth", "status", "--profile", $Profile, "--json") -AllowFailure -ProfileLockRetries 3
    if ($status.ExitCode -ne 0 -or $null -eq $status.Json) {
        return $null
    }

    return [string]$status.Json.data.authorizationStatus
}

function Assert-MiTelegramAuthorized {
    param([Parameter(Mandatory = $true)][string]$Profile)

    $status = Get-MiTelegramAuthStatus -Profile $Profile
    if ($status -ne "Authorized") {
        throw "Profile '$Profile' is not authorized. Run tmp\\smoke-auth.ps1 first."
    }
}
