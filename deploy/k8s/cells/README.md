# 🌐 INTEGIN Sovereign Cell-Based Multi-Region Sharding Architecture

## 1. Executive Topology & Sovereignty Guarantee

To satisfy statutory national data protection and data residency mandates (e.g. **Saudi SDAIA / PDPL** and **EU GDPR Chapter V**), INTEGIN deploys isolated regional cells:

```
                                  ┌────────────────────────┐
                                  │   Anycast Geo-DNS      │
                                  │ (Strict Region Routing)│
                                  └───────────┬────────────┘
                                              │
                    ┌─────────────────────────┴─────────────────────────┐
                    ▼                                                   ▼
       ┌────────────────────────┐                          ┌────────────────────────┐
       │   cell-sa-central-01   │                          │    cell-eu-west-01     │
       │ (Riyadh / Dammam Data) │                          │  (Frankfurt / Dublin)  │
       ├────────────────────────┤                          ├────────────────────────┤
       │ • Jurisdiction: SA     │                          │ • Jurisdiction: EU     │
       │ • Currency: SAR        │                          │ • Currency: EUR        │
       │ • Tax: ZATCA           │                          │ • Tax: VIES            │
       │ • Regulator: SASP      │                          │ • Regulator: EU-OSHA   │
       │ • NetworkPolicy: Zero  │                          │ • NetworkPolicy: Zero  │
       │   Cross-Border Egress  │                          │   Cross-Border Egress  │
       └────────────────────────┘                          └────────────────────────┘
```

## 2. Hard Isolation & Physical Data Plane Pinning

1. **Kubernetes Node Affinity**:
   Workloads are strictly pinned via `integin.com/sovereign-cell: <cell-id>` and `topology.kubernetes.io/region` labels. Pods will never be scheduled on physical hardware outside the legal boundary.

2. **NetworkPolicy Egress Containment**:
   All cells enforce a strict **default-deny egress policy**. Intra-cluster communication is restricted to local cell microservices (INTEGIN Server, PgCat, PostgreSQL, RustFS S3). Direct internet egress across national borders is prohibited at the kernel packet filter layer.

3. **Multi-Tenant RLS & GUC Synchronization**:
   Every database node in the cell runs PostgreSQL 18 with table-level `ROW LEVEL SECURITY (ENABLE + FORCE)`. The local cell configuration injects jurisdiction and residency policies into runtime transaction contexts.

## 3. Directory Layout

```
deploy/k8s/cells/
├── kustomization.yaml            # Root cell aggregator
├── base/
│   ├── cell-namespace.yaml       # Restricted Pod Security Standard namespace
│   ├── network-policy.yaml       # Default-deny egress & cell boundary enforcement
│   ├── integin-deployment.yaml     # INTEGIN Server monolith deployment & service
│   └── kustomization.yaml        # Base kustomization
└── regions/
    ├── sa-central-01/            # Kingdom of Saudi Arabia sovereign cell (SDAIA / PDPL)
    │   └── kustomization.yaml
    └── eu-west-01/               # European Union sovereign cell (GDPR Chapter V)
        └── kustomization.yaml
```

## 4. Deployment Commands

```bash
# Preview Saudi sovereign cell manifests:
kubectl kustomize deploy/k8s/cells/regions/sa-central-01

# Preview EU sovereign cell manifests:
kubectl kustomize deploy/k8s/cells/regions/eu-west-01

# Apply all sovereign cells to cluster:
kubectl apply -k deploy/k8s/cells/
```
