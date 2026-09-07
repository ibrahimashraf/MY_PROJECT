# INTEGIN Consolidation Copy Record

- Legacy roots were copied, not moved; they remain rollback references until controlled cutover validation passes.
- `pilot/` contains non-secret `C:\INTEGIN-PILOT` operational assets excluding `source/` and secret-bearing file extensions.
- `acceptance/` contains non-secret `C:\INTEGIN-RUNTIME` operating assets.
- `private/integin-secrets/` is a private copy of legacy secret files protected with non-inherited access control; no values were displayed.
- Docker-managed volumes and container storage were not copied or moved.