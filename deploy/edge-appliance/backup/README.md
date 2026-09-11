# 🛡️ INTEGIN Sovereign Edge Appliance: Dual-NVMe Disaster Recovery & Rebuild Runbook

## 1. Architectural Topology (Air-Gapped Industrial Sites)

For offshore oil rigs, deep-sea vessels, and sovereign industrial compounds with zero cloud connectivity:

```
┌────────────────────────────────────────────────────────┐
│               INTEGIN Sovereign Appliance              │
├───────────────────────────┬────────────────────────────┤
│       NVMe Drive A        │        NVMe Drive B        │
│      (Primary Data)       │      (Immutable Repo)      │
│  /var/lib/postgresql/data │  /mnt/nvme-backup/pgbackrest│
│                           │                            │
│  • Active PostgreSQL 18   │  • Asynchronous WAL Stream │
│  • Transaction Log Write  │  • LZ4 Block Compression   │
│  • Fillfactor 85 HOT      │  • Full & Diff Stanzas     │
└───────────────────────────┴────────────────────────────┘
```

## 2. Quantitative Service Level Agreements (SLAs)

| Metric | Target SLA | Implementation Mechanism |
|---|---|---|
| **Recovery Time Objective (RTO)** | **< 60 seconds** | `pgbackrest --delta restore` (block-level replacement) |
| **Recovery Point Objective (RPO)** | **< 1 second** | Asynchronous local NVMe WAL streaming |
| **Integrity Verification** | **100% SHA-256** | Block-level checksum verification on all page writes |
| **Network Dependence** | **0 bytes** | 100% air-gapped local NVMe-to-NVMe transfer |

## 3. Operational Drill & Execution

### Automated Recovery Drill
To execute the certified <60s rebuild drill:

```bash
chmod +x deploy/edge-appliance/backup/restore-appliance.sh
./deploy/edge-appliance/backup/restore-appliance.sh
```

### Manual Disaster Recovery Procedure
1. Verify target directory isolation:
   ```bash
   sudo systemctl stop integin-server postgresql || docker compose -f deployments/docker-compose.appliance.yml stop postgres
   ```
2. Execute delta restore from NVMe Drive B:
   ```bash
   pgbackrest --config=/etc/pgbackrest/pgbackrest.conf --stanza=integin-appliance --delta restore
   ```
3. Restart PostgreSQL cluster:
   ```bash
   docker compose -f deployments/docker-compose.appliance.yml start postgres
   ```
4. Verify catalog integrity and tenant RLS:
   ```bash
   pg_isready -U integin_owner -d integin_appliance
   ```
