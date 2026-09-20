# config/edge-appliance — alias, not a copy

`GLOBAL_ARCHITECTURE_PLAN.md` §4 addresses the sovereign air-gapped
appliance as `config/edge-appliance/`. It lives in:

- `../deploy/compose/integin-infrastructure.compose.yaml` — single-node
  offline stack (HAProxy, IdP, pooler, PostgreSQL, RustFS, server).
- `../deploy/edge-appliance/backup/` — pgBackRest config, `restore-appliance.sh`
  drill script, DR verification test.

Kept as a redirect on purpose: compose files cannot kustomize-reference,
and duplicating them would let the copies drift.
