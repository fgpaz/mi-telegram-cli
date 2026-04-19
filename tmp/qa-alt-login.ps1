$ErrorActionPreference = "Stop"
Set-Location "C:\repos\mios\mi-telegram-cli"
Get-Content "C:\repos\mios\mi-telegram-cli\infra\.env" | ForEach-Object {
  if ($_ -match '^\s*#' -or $_ -match '^\s*$') { return }
  $parts = $_ -split '=', 2
  if ($parts.Count -eq 2) {
    [System.Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), 'Process')
  }
}
.\bin\mi-telegram-cli.exe auth login --profile qa-alt --method code --phone +5492612536449
