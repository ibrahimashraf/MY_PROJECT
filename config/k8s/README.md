# config/k8s — alias, not a copy

`GLOBAL_ARCHITECTURE_PLAN.md` §4 addresses the K8s production manifests as
`config/k8s/`. They live in `../deploy/k8s/cells/` (cell `sa-central-01`,
`eu-west-01`, base deployment, network policy, verifier ingress).

Build from either root with identical output:

```bash
kubectl kustomize config/k8s
kubectl kustomize deploy/k8s/cells
```
