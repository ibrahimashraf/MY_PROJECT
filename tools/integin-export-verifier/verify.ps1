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

    Write-Host "[3/3] verifying golden manifest ..."
    & tools/integin-export-verifier/target/release/integin-export-verifier.exe `
        --manifest tools/integin-export-verifier/testdata/manifest.json `
        --public-key tools/integin-export-verifier/testdata/public_key.hex
    if ($LASTEXITCODE -ne 0) {
        Write-Error "verifier rejected a Go-signed manifest (exit $LASTEXITCODE)"
    }

    Write-Host "cross-language verification PASSED"
} finally {
    Pop-Location
}