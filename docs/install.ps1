# Install script for Hector (Windows)
# Usage:
#   irm https://gohector.dev/install.ps1 | iex
#
# Options (set before piping):
#   $env:HECTOR_VERSION = "v1.2.3"        # specific version (default: latest)
#   $env:HECTOR_INSTALL_DIR = "C:\tools"   # install directory (default: ~\.hector\bin)

$ErrorActionPreference = "Stop"

$repo = "verikod/hector"
$binaryName = "hector.exe"
$version = $env:HECTOR_VERSION
$installDir = if ($env:HECTOR_INSTALL_DIR) { $env:HECTOR_INSTALL_DIR } else { Join-Path $HOME ".hector\bin" }

# Detect architecture
$arch = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    "X64"   { "amd64" }
    "Arm64" { "arm64" }
    default {
        Write-Error "Unsupported architecture: $_"
        exit 1
    }
}

# Resolve latest version if not specified
if (-not $version) {
    Write-Host "Fetching latest release..."
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest"
    $version = $release.tag_name
    if (-not $version) {
        Write-Error "Could not determine latest version. Set `$env:HECTOR_VERSION and retry."
        exit 1
    }
}

$versionNumber = $version -replace "^v", ""
$archive = "hector_${versionNumber}_windows_${arch}.zip"
$url = "https://github.com/$repo/releases/download/$version/$archive"

Write-Host "Installing hector $version (windows/$arch)..."
Write-Host "  from: $url"

# Download and extract
$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("hector-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

try {
    $zipPath = Join-Path $tmpDir $archive
    Invoke-WebRequest -Uri $url -OutFile $zipPath -UseBasicParsing
    Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

    $binaryPath = Join-Path $tmpDir $binaryName
    if (-not (Test-Path $binaryPath)) {
        Write-Error "Binary not found in archive."
        exit 1
    }

    # Create install directory
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    Copy-Item -Path $binaryPath -Destination (Join-Path $installDir $binaryName) -Force

    # Add to PATH for current session
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$installDir*") {
        Write-Host ""
        Write-Host "Adding $installDir to your user PATH..."
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$installDir", "User")
        $env:Path = "$env:Path;$installDir"
    }

    Write-Host ""
    Write-Host "Installed: $installDir\$binaryName"
    & (Join-Path $installDir $binaryName) version 2>$null
    Write-Host ""
    Write-Host "Run 'hector --help' to get started."
    Write-Host "Note: restart your terminal for PATH changes to take effect."
}
finally {
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
}
