$ErrorActionPreference = "Stop"

$repo = "dosaki/claude-sessions"
$installDir = "$env:USERPROFILE\.claude-sessions\bin"

Write-Host "Fetching latest release..."
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$tag = $release.tag_name
$asset = "claude-sessions-windows-amd64.exe"
$url = "https://github.com/$repo/releases/download/$tag/$asset"

Write-Host "Installing claude-sessions $tag..."

if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

$dest = Join-Path $installDir "claude-sessions.exe"
Invoke-WebRequest -Uri $url -OutFile $dest

# Add to user PATH if not already present
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your user PATH. Restart your terminal for it to take effect."
}

Write-Host "Done! Run 'claude-sessions' to get started."
