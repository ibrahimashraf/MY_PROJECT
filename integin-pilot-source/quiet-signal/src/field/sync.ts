import { listPendingSync, markSynced } from './db'

const API_BASE = '/api/v1'

export async function syncPendingInspections(
  tenantId: string,
  orgId: string,
  onProgress?: (synced: number, total: number) => void
): Promise<{ synced: number; failed: number }> {
  const pending = await listPendingSync()
  let synced = 0
  let failed = 0

  for (const inspection of pending) {
    try {
      const r = await fetch(`${API_BASE}/inspections`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Organization-ID': orgId,
        },
        body: JSON.stringify(inspection),
      })
      if (r.ok) {
        await markSynced(inspection.id)
        synced++
      } else {
        failed++
      }
    } catch {
      failed++
    }
    onProgress?.(synced, pending.length)
  }

  return { synced, failed }
}

export function registerSyncOnReconnect(
  tenantId: string,
  orgId: string
): () => void {
  let syncing = false

  async function trySync() {
    if (syncing) return
    syncing = true
    try {
      await syncPendingInspections(tenantId, orgId)
    } finally {
      syncing = false
    }
  }

  window.addEventListener('online', trySync)

  return () => {
    window.removeEventListener('online', trySync)
  }
}
