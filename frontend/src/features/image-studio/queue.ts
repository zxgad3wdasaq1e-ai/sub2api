import { reactive } from 'vue'
import type { ImageJob, ImageJobStatus } from './types'

export const MAX_BATCH_SIZE = 4
export const MAX_CONCURRENCY = 4
export const MAX_QUEUE_SIZE = 16

export function remainingImageQueueCapacity(pendingCount: number): number {
  return Math.max(0, MAX_QUEUE_SIZE - pendingCount)
}

export function createReactiveImageJob(job: ImageJob): ImageJob {
  return reactive(job) as ImageJob
}

export function recoverImageJob(job: ImageJob): ImageJob {
  if (job.status !== 'running' || job.remoteTaskId) return job
  return { ...job, status: 'queued', error: undefined }
}

export function shouldPersistImageJobFailure(error: unknown, status: ImageJobStatus): boolean {
  const errorName = error && typeof error === 'object' ? (error as { name?: unknown }).name : undefined
  return errorName !== 'AbortError' && status !== 'canceled'
}
