import { computed, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { createReactiveImageJob, MAX_QUEUE_SIZE, remainingImageQueueCapacity } from '../queue'
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
})
