import type { StudioImage } from './types'

const DB_NAME = 'sub2api-image-studio'
const DB_VERSION = 1
const STORE_NAME = 'images'

function openDatabase(): Promise<IDBDatabase | null> {
  if (typeof indexedDB === 'undefined') return Promise.resolve(null)
  return new Promise((resolve) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onerror = () => resolve(null)
    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(STORE_NAME)) {
        database.createObjectStore(STORE_NAME, { keyPath: 'id' })
      }
    }
    request.onsuccess = () => resolve(request.result)
  })
}

function transactionRequest<T>(
  database: IDBDatabase,
  mode: IDBTransactionMode,
  action: (store: IDBObjectStore) => IDBRequest<T>,
): Promise<T | null> {
  return new Promise((resolve) => {
    const request = action(database.transaction(STORE_NAME, mode).objectStore(STORE_NAME))
    request.onerror = () => resolve(null)
    request.onsuccess = () => resolve(request.result)
  })
}

export async function loadStudioImages(): Promise<StudioImage[]> {
  const database = await openDatabase()
  if (!database) return []
  const items = await transactionRequest<StudioImage[]>(database, 'readonly', (store) => store.getAll())
  database.close()
  return (items || []).sort((a, b) => b.createdAt - a.createdAt)
}

export async function saveStudioImage(image: StudioImage): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  await transactionRequest<IDBValidKey>(database, 'readwrite', (store) => store.put(image))
  database.close()
}

export async function deleteStudioImage(id: string): Promise<void> {
  const database = await openDatabase()
  if (!database) return
  await transactionRequest<undefined>(database, 'readwrite', (store) => store.delete(id))
  database.close()
}

export async function cleanupExpiredStudioImages(now = Date.now()): Promise<number> {
  const images = await loadStudioImages()
  const expired = images.filter((image) => image.expiresAt <= now)
  await Promise.all(expired.map((image) => deleteStudioImage(image.id)))
  return expired.length
}
