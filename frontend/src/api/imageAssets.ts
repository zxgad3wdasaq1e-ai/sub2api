import { apiClient } from './client'

export interface ImageAssetDTO {
  id: string
  task_id: string
  image_index: number
  content_type: string
  byte_size: number
  prompt: string
  model: string
  mode: 'text' | 'edit'
  size: string
  quality: string
  output_format: string
  url: string
  created_at: string
  expires_at: string
}

export async function listImageAssets(options?: { signal?: AbortSignal }): Promise<ImageAssetDTO[]> {
  const { data } = await apiClient.get<{ items: ImageAssetDTO[] }>('/user/image-assets', {
    signal: options?.signal,
  })
  return data.items || []
}

export async function deleteImageAsset(assetId: string): Promise<void> {
  await apiClient.delete(`/user/image-assets/${encodeURIComponent(assetId)}`)
}

export const imageAssetsAPI = { list: listImageAssets, delete: deleteImageAsset }

export default imageAssetsAPI
