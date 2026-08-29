# ==============================================================================
#  ⚡ ARIS Windows PowerShell Installer
# ==============================================================================

$Repo = "SalvucciFacundo/ARIS"
$GithubUrl = "https://github.com/$Repo"

Write-Host ""
Write-Host "  ⚡ ===================================================== ⚡" -ForegroundColor Cyan
Write-Host "      ARIS — Autonomous Reasoner for Image System (Windows) " -ForegroundColor Cyan
Write-Host "  ⚡ ===================================================== ⚡" -ForegroundColor Cyan
Write-Host ""

# 1. Detect Architecture
$Arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    $Arch = "arm64"
}
Write-Host "🖥️  Detected Architecture: windows/$Arch" -ForegroundColor Green

# 2. Determine Install Path
$InstallDir = "$env:LOCALAPPDATA\Programs\aris"
if (!(Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

# 3. Query Latest Release Tag from GitHub
Write-Host "🔍 Checking latest version from GitHub..." -ForegroundColor Yellow
$LatestTag = "v1.0.0"
try {
    $ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{"User-Agent"="ARIS-Installer"}
    if ($ReleaseInfo.tag_name) {
        $LatestTag = $ReleaseInfo.tag_name
    }
} catch {
    Write-Host "⚠️  Could not fetch release tag from GitHub API, falling back to $LatestTag" -ForegroundColor Yellow
}

Write-Host "📦 Latest Version: $LatestTag" -ForegroundColor Green

# 4. Download Zip Package
$ZipName = "aris_${LatestTag}_windows_${Arch}.zip"
$DownloadUrl = "$GithubUrl/releases/download/$LatestTag/$ZipName"
$TempZip = "$env:TEMP\$ZipName"

Write-Host "⬇️  Downloading $ZipName..." -ForegroundColor Cyan
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing
} catch {
    Write-Host "❌ Failed to download pre-built binary. Attempting 'go install'..." -ForegroundColor Yellow
    if (Get-Command go -ErrorAction SilentlyContinue) {
        go install "github.com/$Repo/cmd/aris@latest"
        Write-Host "✅ Installed successfully via 'go install'!" -ForegroundColor Green
        Exit 0
    } else {
        Write-Host "❌ Download failed and Go is not installed on this machine." -ForegroundColor Red
        Exit 1
    }
}

# 5. Extract and Install Binary
Write-Host "📦 Extracting aris.exe..." -ForegroundColor Cyan
Expand-Archive -Path $TempZip -DestinationPath $InstallDir -Force
Remove-Item -Path $TempZip -Force

# 6. Check and Add to User PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "⚙️  Adding $InstallDir to user PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
}

Write-Host ""
Write-Host "✨ ARIS successfully installed to $InstallDir\aris.exe!" -ForegroundColor Green
Write-Host "Run 'aris --help' or 'aris chat' in your terminal to start!" -ForegroundColor Cyan
Write-Host ""
