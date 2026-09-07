# INTEGIN Pilot Workspace Consolidation

This is a non-secret workspace copy. The active runtime continues to use `C:\INTEGIN-PILOT`; no live process, Docker resource, private environment, credential, database, object-store volume, Keycloak ZIP process location, OpenBao recovery file, or runtime log was moved.

Included: reviewed pilot source, contract files, approved administration scripts, and the non-secret OpenBao configuration.
Excluded: `C:\INTEGIN-SECRETS`, `C:\INTEGIN-PILOT\runtime`, `C:\INTEGIN-PILOT\keycloak-zip`, `C:\INTEGIN-PILOT\tools`, Keycloak staging, OpenBao diagnostic payloads, backups, and derived dependency/build directories.

A later controlled cutover may update active paths only after the copied source passes hash, Go, Flutter, and health validation.