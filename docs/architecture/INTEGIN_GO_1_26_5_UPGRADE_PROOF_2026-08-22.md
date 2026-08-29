# Go 1.26.5 Upgrade Proof — 2026-08-22

## Completed upgrade

The active Windows Go toolchain is `go1.26.5 windows/amd64`. The project module directive was updated from `go 1.22` to `go 1.26.5`, then module metadata was reconciled with `go mod tidy`.

## Compatibility evidence

| Check | Result |
|---|---|
| Active executable | `C:\Program Files\Go\bin\go.exe` reports Go 1.26.5 |
| Focused certificate, public verifier, and renderer tests | Passed |
| Full `go test -count=1 ./...` | Passed |
| Full `go vet ./...` | Passed |

## Dependency decision still frozen

The owner-supplied official Codeberg release page shows that `codeberg.org/go-pdf/fpdf` has a current stable v0.12.0 release. The currently installed prototype still imports the older `github.com/go-pdf/fpdf v0.9.0` module path. The official Codeberg main module uses the separate `codeberg.org/go-pdf/fpdf` module path and requires Go 1.25, which the upgraded toolchain now satisfies.

No switch from the GitHub-era dependency to the Codeberg module was made in this upgrade. The QR dependency also remains frozen. Retaining, replacing, or removing those dependencies requires the owner’s explicit follow-up approval.

## Reference

Official Codeberg releases: <https://codeberg.org/go-pdf/fpdf/releases>
