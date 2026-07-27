import { computed, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { createReactiveImageJob, MAX_QUEUE_SIZE, recoverImageJob, remainingImageQueueCapacity, shouldPersistImageJobFailure } from '../queue'
import type { ImageJob } from '../types'

function queuedJob(): ImageJob {
  return {
    id: 'job-1',
    ownerUserId: 1,
    prompt: '生成一个男孩',
    model: 'gpt-image-2',
    mode: 'text',
    size: '1024x1024',
    aspectRatio: '1:1',
    apiKeyId: 1,
    quality: 'auto',
    outputFormat: 'png',
    status: 'queued',
    createdAt: 1,
  }
}

describe('image studio queue', () => {
  it('reacts to the first queued job becoming active', () => {
    const jobs = ref<ImageJob[]>([])
    const activeCount = computed(() => jobs.value.filter((job) => job.status === 'running').length)
    const job = createReactiveImageJob(queuedJob())

    jobs.value.push(job)
    expect(activeCount.value).toBe(0)

    job.status = 'running'
    expect(activeCount.value).toBe(1)
  })

  it('limits pending jobs to sixteen', () => {
    expect(remainingImageQueueCapacity(0)).toBe(MAX_QUEUE_SIZE)
    expect(remainingImageQueueCapacity(15)).toBe(1)
    expect(remainingImageQueueCapacity(16)).toBe(0)
    expect(remainingImageQueueCapacity(20)).toBe(0)
  })

  it('keeps a running server task active when restoring after refresh', () => {
    const stored = { ...queuedJob(), status: 'running' as const, remoteTaskId: 'imgtask_123' }

    expect(recoverImageJob(stored)).toBe(stored)
    expect(recoverImageJob(stored).status).toBe('running')
  })

  it('requeues legacy browser-owned requests that have no server task id', () => {
    const stored = { ...queuedJob(), status: 'running' as const, error: 'old error' }

    expect(recoverImageJob(stored)).toMatchObject({ status: 'queued', error: undefined })
  })

  it('does not turn a lifecycle abort or explicit cancellation into a failure', () => {
    expect(shouldPersistImageJobFailure(new DOMException('Aborted', 'AbortError'), 'running')).toBe(false)
    expect(shouldPersistImageJobFailure(new Error('late provider error'), 'canceled')).toBe(false)
    expect(shouldPersistImageJobFailure(new Error('provider error'), 'running')).toBe(true)
  })
})
