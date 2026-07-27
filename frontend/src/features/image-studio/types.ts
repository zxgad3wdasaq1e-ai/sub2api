export type ImageMode = 'text' | 'edit'
export type ImageJobStatus = 'queued' | 'running' | 'completed' | 'failed' | 'canceled'

export interface StudioImage {
  id: string
  ownerUserId: number
  prompt: string
  model: string
  mode: ImageMode
  size: string
  aspectRatio?: string
  quality: string
  outputFormat: string
  createdAt: number
  expiresAt: number
  blob?: Blob
  remoteUrl?: string
}

export interface ImageJob {
  id: string
  ownerUserId: number
  prompt: string
  model: string
  mode: ImageMode
  size: string
  aspectRatio: string
  apiKeyId: number
  quality: string
  outputFormat: string
  references?: File[]
  status: ImageJobStatus
  createdAt: number
  error?: string
  abortController?: AbortController
}
