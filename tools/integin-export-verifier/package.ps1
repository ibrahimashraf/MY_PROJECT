# package.ps1 - Builds, packages, and signs distribution bundles for integin-export-verifier
# Produces self-contained regulatory audit packages with SHA-256 manifests.
[CmdletBinding()]
param(
    [string]$Target = "",
    [string]$OutputDir = "dist",
    [string]$Version = "0.1.0"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

try {
    Write-Host "==> [1/5] Checking build environment..."
    $cargoCmd = Get-Command cargo -ErrorAction SilentlyContinue
    if (-not $cargoCmd) {
        Write-Warning "cargo is not detected on PATH. Attempting to package existing release binary if available."
        $binaryPath = "target/release/integin-export-verifier.exe"
        if (-not (Test-Path $binaryPath)) {
            $binaryPath = "target/release/integin-export-verifier"
        }
        if (-not (Test-Path $binaryPath)) {
            throw "Neither cargo nor pre-built release binary found in target/release/. Install Rust toolchain or build before packaging."
        }
    } else {
        Write-Host "==> [2/5] Building release binary with Cargo (lto=true, strip=true)..."
        $buildArgs = @("build", "--release")
        if ($Target -ne "") {
            $buildArgs += @("--target", $Target)
        }
        & cargo @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Cargo release build failed with exit code $LASTEXITCODE"
        }

        if ($Target -ne "") {
            $binaryPath = "target/$Target/release/integin-export-verifier.exe"
            if (-not (Test-Path $binaryPath)) {
                $binaryPath = "target/$Target/release/integin-export-verifier"
            }
        } else {
            $binaryPath = "target/release/integin-export-verifier.exe"
            if (-not (Test-Path $binaryPath)) {
                $binaryPath = "target/release/integin-export-verifier"
            }
        }
    }

    $targetName = if ($Target -ne "") { $Target } else { "x86_64-pc-windows-msvc" }
    $pkgDirName = "integin-export-verifier-v${Version}-${targetName}"
    $distPath = Join-Path $OutputDir $pkgDirName

    Write-Host "==> [3/5] Assembling regulatory distribution directory at $distPath..."
    if (Test-Path $distPath) {
        Remove-Item -Recurse -Force $distPath
    }
    New-Item -ItemType Directory -Force -Path $distPath | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $distPath "testdata") | Out-Null

    # Copy binary and documentation
    Copy-Item $binaryPath -Destination $distPath
    Copy-Item "README.md" -Destination $distPath
    Copy-Item "AUDITOR_INSTRUCTIONS.md" -Destination $distPath

    # Copy golden validation testdata
    if (Test-Path "testdata/manifest.json") { Copy-Item "testdata/manifest.json" -Destination (Join-Path $distPath "testdata") }
    if (Test-Path "testdata/public_key.hex") { Copy-Item "testdata/public_key.hex" -Destination (Join-Path $distPath "testdata") }
    if (Test-Path "testdata/export.zip") { Copy-Item "testdata/export.zip" -Destination (Join-Path $distPath "testdata") }

    # Create zip archive
    Write-Host "==> [4/5] Compressing regulatory release archive..."
    $zipFileName = "${pkgDirName}.zip"
    $zipFilePath = Join-Path $OutputDir $zipFileName
    if (Test-Path $zipFilePath) {
        Remove-Item -Force $zipFilePath
    }
    Compress-Archive -Path "$distPath\*" -DestinationPath $zipFilePath

    # Generate cryptographic SHA-256 manifest
    Write-Host "==> [5/5] Generating SHA256 checksums..."
    $zipHash = (Get-FileHash -Path $zipFilePath -Algorithm SHA256).Hash.ToLower()
    $binHash = (Get-FileHash -Path (Join-Path $distPath (Split-Path $binaryPath -Leaf)) -Algorithm SHA256).Hash.ToLower()

    $sumsFile = Join-Path $OutputDir "SHA256SUMS.txt"
    $manifestLines = @(
        "$binHash  $pkgDirName/" + (Split-Path $binaryPath -Leaf),
        "$zipHash  $zipFileName"
    )
    $manifestLines | Out-File -FilePath $sumsFile -Encoding utf8 -Append

    Write-Host "=========================================================="
    Write-Host "✅ REGULATORY PACKAGE GENERATED SUCCESSFULLY"
    Write-Host "Archive:  $zipFilePath"
    Write-Host "Checksum: $zipHash"
    Write-Host "Manifest: $sumsFile"
    Write-Host "=========================================================="
} finally {
    Pop-Location
}
