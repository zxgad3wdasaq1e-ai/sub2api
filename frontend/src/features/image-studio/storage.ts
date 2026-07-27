import type { ImageJob, StudioImage } from './types'

const DB_NAME = 'sub2api-image-studio'
const DB_VERSION = 2
const IMAGE_STORE_NAME = 'images'
const JOB_STORE_NAME = 'jobs'
export const STUDIO_RETENTION_DAYS = 7
export const STUDIO_RETENTION_MS = STUDIO_RETENTION_DAYS * 24 * 60 * 60 * 1000

function openDatabase(): Promise<IDBDatabase | null> {
  if (typeof indexedDB === 'undefined') return Promise.resolve(null)
  return new Promise((resolve) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onerror = () => resolve(null)
    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(IMAGE_STORE_NAME)) {
        database.createObjectStore(IMAGE_STORE_NAME, { keyPath: 'id' })
      }
      if (!database.objectStoreNames.contains(JOB_STORE_NAME)) {
        database.createObjectStore(JOB_STORE_NAME, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
  })
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

export async function loadStudioImages(): Promise<StudioImage[]> {
  const database = await openDatabase()
  if (!database) return []
  const items = await transactionRequest<StudioImage[]>(database, IMAGE_STORE_NAME, 'readonly', (store) => store.getAll())
  database.close()
  return (items || []).sort((a, b) => b.createdAt - a.createdAt)
}

export async function saveStudioImage(image: StudioImage): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  await transactionRequest<IDBValidKey>(database, IMAGE_STORE_NAME, 'readwrite', (store) => store.put(image))
  database.close()
}

export async function deleteStudioImage(id: string): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  await transactionRequest<undefined>(database, IMAGE_STORE_NAME, 'readwrite', (store) => store.delete(id))
  database.close()
}

export async function cleanupExpiredStudioImages(now = Date.now()): Promise<number> {
  const images = await loadStudioImages()
  const expired = images.filter((image) => image.expiresAt <= now)
  await Promise.all(expired.map((image) => deleteStudioImage(image.id)))
  return expired.length
}

export async function loadStudioJobs(): Promise<ImageJob[]> {
  const database = await openDatabase()
  if (!database) return []
  const items = await transactionRequest<ImageJob[]>(database, JOB_STORE_NAME, 'readonly', (store) => store.getAll())
  database.close()
  return (items || []).sort((a, b) => b.createdAt - a.createdAt)
}

export async function saveStudioJob(job: ImageJob): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  const persisted = { ...job }
  delete persisted.abortController
  await transactionRequest<IDBValidKey>(database, JOB_STORE_NAME, 'readwrite', (store) => store.put(persisted))
  database.close()
}

export async function deleteStudioJob(id: string): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  await transactionRequest<undefined>(database, JOB_STORE_NAME, 'readwrite', (store) => store.delete(id))
  database.close()
}

export async function cleanupExpiredStudioJobs(now = Date.now()): Promise<number> {
  const jobs = await loadStudioJobs()
  const expired = jobs.filter((job) => job.createdAt + STUDIO_RETENTION_MS <= now)
  await Promise.all(expired.map((job) => deleteStudioJob(job.id)))
  return expired.length
}
