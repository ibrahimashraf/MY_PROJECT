# verify.ps1 - regeneration + cross-language verification for
# tools/integin-export-verifier. Regenerates the golden fixture with the Go
# module, then builds and runs the Rust verifier against it. Requires Go and
# cargo/rustc on PATH.
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

try {
    Write-Host "[1/3] regenerating golden fixture with Go ..."
    go run tools/integin-export-verifier/testdata/generate_fixture.go

    Write-Host "[2/3] building Rust verifier ..."
    Push-Location tools/integin-export-verifier
    try {
        cargo build --release
    } finally {
        Pop-Location
    }

    Write-Host "[3/4] verifying golden manifest + directory blobs ..."
    & tools/integin-export-verifier/target/release/integin-export-verifier.exe `
        --manifest tools/integin-export-verifier/testdata/manifest.json `
        --public-key tools/integin-export-verifier/testdata/public_key.hex `
        --evidence-dir tools/integin-export-verifier/testdata/blobs
    if ($LASTEXITCODE -ne 0) {
        Write-Error "verifier rejected directory evidence blobs (exit $LASTEXITCODE)"
    }

    Write-Host "[4/4] verifying golden export.zip archive ..."
    & tools/integin-export-verifier/target/release/integin-export-verifier.exe `
        --archive tools/integin-export-verifier/testdata/export.zip `
        --public-key tools/integin-export-verifier/testdata/public_key.hex
    if ($LASTEXITCODE -ne 0) {
        Write-Error "verifier rejected export.zip archive (exit $LASTEXITCODE)"
    }

    Write-Host "cross-language end-to-end verification PASSED"
} finally {
    Pop-Location
}