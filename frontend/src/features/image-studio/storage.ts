import type { ImageJob, StudioImage } from './types'

const DB_NAME = 'sub2api-image-studio'
const DB_VERSION = 3
const IMAGE_STORE_NAME = 'images'
const JOB_STORE_NAME = 'jobs'
const OWNER_INDEX_NAME = 'ownerUserId'
export const STUDIO_RETENTION_DAYS = 7
export const STUDIO_RETENTION_MS = STUDIO_RETENTION_DAYS * 24 * 60 * 60 * 1000

function openDatabase(): Promise<IDBDatabase | null> {
  if (typeof indexedDB === 'undefined') return Promise.resolve(null)
  return new Promise((resolve) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    let settled = false
    const finish = (database: IDBDatabase | null) => {
      if (settled) {
        database?.close()
        return
      }
      settled = true
      resolve(database)
    }
    request.onerror = () => finish(null)
    request.onblocked = () => finish(null)
    request.onupgradeneeded = () => {
      const database = request.result
      const transaction = request.transaction
      if (!transaction) return
      for (const storeName of [IMAGE_STORE_NAME, JOB_STORE_NAME]) {
        const store = database.objectStoreNames.contains(storeName)
          ? transaction.objectStore(storeName)
          : database.createObjectStore(storeName, { keyPath: 'id' })
        if (!store.indexNames.contains(OWNER_INDEX_NAME)) {
          store.createIndex(OWNER_INDEX_NAME, OWNER_INDEX_NAME, { unique: false })
        }

        // Records created before ownership was introduced cannot be attributed
        // safely. Remove them instead of exposing them to the next login.
        const cursorRequest = store.openCursor()
        cursorRequest.onsuccess = () => {
          const cursor = cursorRequest.result
          if (!cursor) return
          if (!isValidOwnerUserId((cursor.value as { ownerUserId?: unknown })?.ownerUserId)) {
            cursor.delete()
          }
          cursor.continue()
        }
      }
    }
    request.onsuccess = () => finish(request.result)
  })
}

function isValidOwnerUserId(ownerUserId: unknown): ownerUserId is number {
  return typeof ownerUserId === 'number' && Number.isInteger(ownerUserId) && ownerUserId > 0
}

function transactionRequest<T>(
  database: IDBDatabase,
  storeName: string,
  mode: IDBTransactionMode,
  action: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T | null> {
  return new Promise((resolve) => {
    const request = action(database.transaction(storeName, mode).objectStore(storeName))
    request.onerror = () => resolve(null)
    request.onsuccess = () => resolve(request.result)
  })
}

async function loadOwnedRecords<T>(storeName: string, ownerUserId: number): Promise<T[]> {
  if (!isValidOwnerUserId(ownerUserId)) return []
  const database = await openDatabase()
  if (!database) return []
  const items = await transactionRequest<T[]>(database, storeName, 'readonly', (store) => (
    store.index(OWNER_INDEX_NAME).getAll(IDBKeyRange.only(ownerUserId))
  ))
  database.close()
  return items || []
}

async function putOwnedRecord(storeName: string, ownerUserId: number, value: { id: string, ownerUserId: number }): Promise<boolean> {
  if (!isValidOwnerUserId(ownerUserId) || value.ownerUserId !== ownerUserId || !value.id) return false
  const database = await openDatabase()
  if (!database) return false
  return new Promise((resolve) => {
    const transaction = database.transaction(storeName, 'readwrite')
    const store = transaction.objectStore(storeName)
    let saved = false
    const getRequest = store.get(value.id)
    getRequest.onerror = () => transaction.abort()
    getRequest.onsuccess = () => {
      const existing = getRequest.result as { ownerUserId?: unknown } | undefined
      if (existing && existing.ownerUserId !== ownerUserId) {
        transaction.abort()
        return
      }
      const putRequest = store.put(value)
      putRequest.onsuccess = () => { saved = true }
    }
    transaction.oncomplete = () => {
      database.close()
      resolve(saved)
    }
    transaction.onerror = () => {
      database.close()
      resolve(false)
    }
    transaction.onabort = () => {
      database.close()
      resolve(false)
    }
  })
}

async function deleteOwnedRecord(storeName: string, ownerUserId: number, id: string): Promise<boolean> {
  if (!isValidOwnerUserId(ownerUserId) || !id) return false
  const database = await openDatabase()
  if (!database) return false
  return new Promise((resolve) => {
    const transaction = database.transaction(storeName, 'readwrite')
    const store = transaction.objectStore(storeName)
    let deleted = false
    const getRequest = store.get(id)
    getRequest.onerror = () => transaction.abort()
    getRequest.onsuccess = () => {
      const existing = getRequest.result as { ownerUserId?: unknown } | undefined
      if (!existing || existing.ownerUserId !== ownerUserId) return
      const deleteRequest = store.delete(id)
      deleteRequest.onsuccess = () => { deleted = true }
    }
    transaction.oncomplete = () => {
      database.close()
      resolve(deleted)
    }
    transaction.onerror = () => {
      database.close()
      resolve(false)
    }
    transaction.onabort = () => {
      database.close()
      resolve(false)
    }
  })
}

export async function loadStudioImages(ownerUserId: number): Promise<StudioImage[]> {
  const items = await loadOwnedRecords<StudioImage>(IMAGE_STORE_NAME, ownerUserId)
  return (items || []).sort((a, b) => b.createdAt - a.createdAt)
}

export async function saveStudioImage(ownerUserId: number, image: StudioImage): Promise<void> {
  if (!isValidOwnerUserId(ownerUserId) || image.ownerUserId !== ownerUserId) return
  await putOwnedRecord(IMAGE_STORE_NAME, ownerUserId, image)
}

export async function deleteStudioImage(ownerUserId: number, id: string): Promise<boolean> {
  return deleteOwnedRecord(IMAGE_STORE_NAME, ownerUserId, id)
}

export async function cleanupExpiredStudioImages(ownerUserId: number, now = Date.now()): Promise<number> {
  const images = await loadStudioImages(ownerUserId)
  const expired = images.filter((image) => image.expiresAt <= now)
  await Promise.all(expired.map((image) => deleteStudioImage(ownerUserId, image.id)))
  return expired.length
}

export async function loadStudioJobs(ownerUserId: number): Promise<ImageJob[]> {
  const items = await loadOwnedRecords<ImageJob>(JOB_STORE_NAME, ownerUserId)
  return (items || []).sort((a, b) => b.createdAt - a.createdAt)
}

export async function saveStudioJob(ownerUserId: number, job: ImageJob): Promise<void> {
  if (!isValidOwnerUserId(ownerUserId) || job.ownerUserId !== ownerUserId) return
  const persisted = {
    ...job,
    references: job.references ? [...job.references] : undefined,
  }
  delete persisted.abortController
  await putOwnedRecord(JOB_STORE_NAME, ownerUserId, persisted)
}

export async function deleteStudioJob(ownerUserId: number, id: string): Promise<boolean> {
  return deleteOwnedRecord(JOB_STORE_NAME, ownerUserId, id)
}

export async function cleanupExpiredStudioJobs(ownerUserId: number, now = Date.now()): Promise<number> {
  const jobs = await loadStudioJobs(ownerUserId)
  const expired = jobs.filter((job) => job.createdAt + STUDIO_RETENTION_MS <= now)
  await Promise.all(expired.map((job) => deleteStudioJob(ownerUserId, job.id)))
  return expired.length
}
