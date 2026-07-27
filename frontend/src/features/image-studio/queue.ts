import { reactive } from 'vue'
import type { ImageJob } from './types'

export const MAX_BATCH_SIZE = 4
export const MAX_CONCURRENCY = 4
export const MAX_QUEUE_SIZE = 16

export function remainingImageQueueCapacity(pendingCount: number): number {
  return Math.max(0, MAX_QUEUE_SIZE - pendingCount)
}

export function createReactiveImageJob(job: ImageJob): ImageJob {
  return reactive(job) as ImageJob
}
