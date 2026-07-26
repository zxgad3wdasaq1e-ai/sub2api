export type ImageMode = 'text' | 'edit'
export type ImageJobStatus = 'queued' | 'running' | 'completed' | 'failed' | 'canceled'

export interface StudioImage {
  id: string
  prompt: string
  revisedPrompt?: string
  model: string
  mode: ImageMode
  size: string
  quality: string
  outputFormat: string
  createdAt: number
  expiresAt: number
  blob?: Blob
  remoteUrl?: string
}

export interface ImageJob {
  id: string
  prompt: string
  model: string
  mode: ImageMode
  size: string
  status: ImageJobStatus
  createdAt: number
  error?: string
  abortController?: AbortController
}
