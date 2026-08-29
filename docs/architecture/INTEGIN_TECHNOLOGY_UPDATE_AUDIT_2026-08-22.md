# INTEGIN Technology Update Audit — 2026-08-22

## Scope

This audit reviews the active INTEGIN Go service, PostgreSQL, RustFS, Keycloak, OpenBao, and non-frozen Go dependencies. It intentionally excludes the owner-frozen `github.com/go-pdf/fpdf` and `github.com/skip2/go-qrcode` decision.

## Observed versions and posture

| Technology | Observed version | Evidence-based posture | Recommendation |
|---|---:|---|---|
| Go | `1.26.5` | Active toolchain and project directive align; focused and full regression/vet passed | Current; retain |
| PostgreSQL | `18.6` | PostgreSQL lists 18.6 as the current supported minor for major 18 | Current; pin the exact tested minor/digest for repeatability |
| Keycloak | `26.7.1` | Official 26.7.1 notice identifies this as a current security-fix release | Current; retain and apply future security patch releases through a controlled identity migration test |
| RustFS | `1.0.0-rc.1` | The active image is a release candidate; official GitHub releases now list newer `1.0.0-rc.3 (rc)` | Not current and not final-GA; do not update in place. Plan backup/restore and S3 compatibility rehearsal before any controlled upgrade |
| OpenBao | `2.6.0` | Official 2.6.x notes list 2.6.2 security fixes after 2.6.0 | Patch update recommended, but only through sealed-state/config/export and recovery testing |
| `github.com/golang-jwt/jwt/v5` | `5.3.1` | Go module update query reports no newer available module update; package is production-ready | Current; retain |
| `github.com/lib/pq` | `1.10.9` | Go module update query reports `1.12.3`; package supports maintained PostgreSQL versions but is a conservative database/sql driver line | Patch update recommended with a PostgreSQL integration proof; no driver migration implied |
| `golang.org/x/image` | `0.12.0` transitive | Go module update query reports `0.45.0`; it is currently pulled through the frozen PDF dependency | Frozen dependency boundary; do not force independently |

## Important constraints

The PostgreSQL image tag is broad (`postgres:18`) even though the running process reports 18.6. For repeatable pilot and future production workflows, a planned change should pin an exact minor image and immutable digest after a backup/recovery drill.

The local RustFS and OpenBao containers are not production release claims. Several are deliberately stopped/sealed pilot resources. Their upgrade recommendations require their own data/configuration backup and recovery evidence; this audit did not change any image, container, data volume, configuration, or dependency.

## Sources

- PostgreSQL version policy: <https://www.postgresql.org/support/versioning/>
- Keycloak 26.7.1 official release: <https://www.keycloak.org/2026/08/keycloak-2671-released>
- RustFS official releases: <https://github.com/rustfs/rustfs/releases>
- RustFS official download guidance: <https://rustfs.com/download/>
- OpenBao 2.6.x official release notes: <https://openbao.org/community/release-notes/2-6-0/>
- Go package metadata: <https://pkg.go.dev/github.com/golang-jwt/jwt/v5> and <https://pkg.go.dev/github.com/lib/pq>
