const DB_NAME = 'integin-field'
const DB_VERSION = 1
const STORE_INSPECTIONS = 'inspections'
const STORE_SYNC_QUEUE = 'sync_queue'

export function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_INSPECTIONS)) {
        const store = db.createObjectStore(STORE_INSPECTIONS, { keyPath: 'id' })
        store.createIndex('status', 'status', { unique: false })
        store.createIndex('synced', 'synced', { unique: false })
      }
      if (!db.objectStoreNames.contains(STORE_SYNC_QUEUE)) {
        db.createObjectStore(STORE_SYNC_QUEUE, { keyPath: 'id', autoIncrement: true })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

export async function saveInspection(inspection: InspectionRecord): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_INSPECTIONS, 'readwrite')
    tx.objectStore(STORE_INSPECTIONS).put(inspection)
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error)
  })
}

export async function getInspection(id: string): Promise<InspectionRecord | undefined> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_INSPECTIONS, 'readonly')
    const req = tx.objectStore(STORE_INSPECTIONS).get(id)
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function listPendingSync(): Promise<InspectionRecord[]> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_INSPECTIONS, 'readonly')
    const index = tx.objectStore(STORE_INSPECTIONS).index('synced')
    const req = index.getAll(IDBKeyRange.only(0))
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error)
  })
}

export async function markSynced(id: string): Promise<void> {
  const inspection = await getInspection(id)
  if (inspection) {
    inspection.synced = 1
    inspection.synced_at = new Date().toISOString()
    await saveInspection(inspection)
  }
}

export type InspectionRecord = {
  id: string
  tenant_id: string
  organization_id: string
  work_order_id: string
  asset_id: string
  equipment_type: string
  status: 'DRAFT' | 'IN_PROGRESS' | 'COMPLETED' | 'SYNCED'
  synced: 0 | 1
  synced_at?: string
  answers: Record<string, unknown>
  photos: string[]
  created_at: string
  updated_at: string
}

export function generateInspectionID(): string {
  return `INS-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}
