import { IDBFactory, IDBKeyRange } from 'fake-indexeddb'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { reactive } from 'vue'
import {
  deleteStudioImage,
  deleteStudioJob,
  loadStudioImages,
  loadStudioJobs,
  saveStudioImage,
  saveStudioJob,
} from '../storage'
import type { ImageJob, StudioImage } from '../types'

const DB_NAME = 'sub2api-image-studio'

function image(id: string, ownerUserId: number): StudioImage {
  return {
    id,
    ownerUserId,
    prompt: `image-${id}`,
    model: 'gpt-image-2',
    mode: 'text',
    size: '1024x1024',
    quality: 'auto',
    outputFormat: 'png',
    createdAt: 1,
    expiresAt: Date.now() + 60_000,
  }
}

function job(id: string, ownerUserId: number): ImageJob {
  return {
    id,
    ownerUserId,
    prompt: `job-${id}`,
    model: 'gpt-image-2',
    mode: 'text',
    size: '1024x1024',
    aspectRatio: '1:1',
    apiKeyId: 1,
    quality: 'auto',
    outputFormat: 'png',
    status: 'completed',
    createdAt: 1,
  }
}

function deleteDatabase(): Promise<void> {
  return new Promise((resolve) => {
    const request = indexedDB.deleteDatabase(DB_NAME)
    request.onsuccess = request.onerror = request.onblocked = () => resolve()
  })
}

function seedLegacyDatabase(): Promise<void> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 2)
    request.onerror = () => reject(request.error)
    request.onupgradeneeded = () => {
      const database = request.result
      const imageStore = database.createObjectStore('images', { keyPath: 'id' })
      const jobStore = database.createObjectStore('jobs', { keyPath: 'id' })
      imageStore.put({ ...image('legacy-image', 101), ownerUserId: undefined })
      jobStore.put({ ...job('legacy-job', 101), ownerUserId: undefined })
    }
    request.onsuccess = () => {
      request.result.close()
      resolve()
    }
  })
}

function countRecords(storeName: string): Promise<number> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 3)
    request.onerror = () => reject(request.error)
    request.onsuccess = () => {
      const database = request.result
      const countRequest = database.transaction(storeName).objectStore(storeName).count()
      countRequest.onerror = () => reject(countRequest.error)
      countRequest.onsuccess = () => {
        database.close()
        resolve(countRequest.result)
      }
    }
  })
}

beforeEach(() => {
  Object.defineProperty(globalThis, 'indexedDB', { configurable: true, value: new IDBFactory() })
  Object.defineProperty(globalThis, 'IDBKeyRange', { configurable: true, value: IDBKeyRange })
})

afterEach(async () => {
  await deleteDatabase()
})

describe('image studio ownership storage', () => {
  it('only loads records owned by the requested user', async () => {
    await saveStudioImage(101, image('image-a', 101))
    await saveStudioImage(202, image('image-b', 202))
    await saveStudioJob(101, job('job-a', 101))
    await saveStudioJob(202, job('job-b', 202))

    expect((await loadStudioImages(101)).map((item) => item.id)).toEqual(['image-a'])
    expect((await loadStudioImages(202)).map((item) => item.id)).toEqual(['image-b'])
    expect((await loadStudioJobs(101)).map((item) => item.id)).toEqual(['job-a'])
    expect((await loadStudioJobs(202)).map((item) => item.id)).toEqual(['job-b'])
  })

  it('cannot overwrite or delete another user\'s record by id', async () => {
    await saveStudioImage(101, image('shared-id', 101))
    await saveStudioJob(101, job('shared-job', 101))

    await saveStudioImage(202, image('shared-id', 202))
    await saveStudioJob(202, job('shared-job', 202))
    expect(await deleteStudioImage(202, 'shared-id')).toBe(false)
    expect(await deleteStudioJob(202, 'shared-job')).toBe(false)

    expect((await loadStudioImages(101))[0]?.ownerUserId).toBe(101)
    expect((await loadStudioJobs(101))[0]?.ownerUserId).toBe(101)
  })

  it('persists an edit job after Vue makes its reference array reactive', async () => {
    const editJob = reactive({
      ...job('edit-job', 101),
      mode: 'edit' as const,
      references: [new File(['source'], 'source.png', { type: 'image/png' })],
    }) as ImageJob

    await saveStudioJob(101, editJob)

    const stored = await loadStudioJobs(101)
    expect(stored).toHaveLength(1)
    expect(stored[0]?.id).toBe('edit-job')
    expect(stored[0]?.references).toHaveLength(1)
  })

  it('deletes legacy records that have no trustworthy owner', async () => {
    await seedLegacyDatabase()

    expect(await loadStudioImages(101)).toEqual([])
    expect(await loadStudioJobs(101)).toEqual([])
    expect(await countRecords('images')).toBe(0)
    expect(await countRecords('jobs')).toBe(0)
  })
})
