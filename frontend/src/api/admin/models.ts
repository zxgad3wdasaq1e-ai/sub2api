/**
 * Shared model pricing API.
 *
 * Calls the real backend endpoint:
 * GET /api/v1/models/pricing?keyword=&category=
 */

import { apiClient } from '../client'

export type ModelMarketCategory = 'recommended' | 'all' | 'platform'

export interface ModelPricing {
  input: number | null
  output: number | null
  cachedInput: number | null
  cachedOutput: number | null
}

export interface ModelMarketModel {
  id: string
  name: string
  type: string
  typeLabel?: string
  category: string
  platform?: string
  status?: 'available' | 'disabled'
  recommended?: boolean
  platformAdapted?: boolean
  pricing: ModelPricing
  channelCount?: number
  channelIds?: number[]
}

export interface ModelListParams {
  keyword?: string
  category?: ModelMarketCategory | string
}

export interface ModelListResponse {
  models: ModelMarketModel[]
  total: number
  availableChannels: number
  updatedAt: string
}

export type UpdateModelRequest = Partial<Omit<ModelMarketModel, 'id'>>

/** Raw response from GET /api/v1/models/pricing */
interface RawModelMarketEntry {
  id: string
  name: string
  type: string
  type_label?: string
  category?: string
  platform: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  channel_count: number
  channel_ids?: number[]
  recommended?: boolean
  platform_adapted?: boolean
}

interface RawModelListResponse {
  models?: RawModelMarketEntry[]
  total?: number
  available_channels?: number
}

function transformModelList(raw: RawModelListResponse): ModelListResponse {
  const models: ModelMarketModel[] = (raw.models || []).map((entry) => {
    return {
      id: entry.id,
      name: entry.name,
      type: entry.type || (entry.platform_adapted ? 'ADAPTED' : 'OFFICIAL'),
      typeLabel: entry.type_label || entry.platform,
      category: entry.category || 'all',
      platform: entry.platform,
      status: 'available' as const,
      recommended: entry.recommended === true,
      platformAdapted: entry.platform_adapted === true,
      pricing: {
        input: entry.input_price,
        output: entry.output_price,
        cachedInput: entry.cache_write_price,
        cachedOutput: entry.cache_read_price,
      },
      channelCount: entry.channel_count,
      channelIds: entry.channel_ids || [],
    }
  })

  return {
    models,
    total: raw.total ?? models.length,
    availableChannels: raw.available_channels ?? 0,
    updatedAt: new Date().toISOString(),
  }
}

export async function getModels(params: ModelListParams = {}): Promise<ModelListResponse> {
  const { data } = await apiClient.get<RawModelListResponse>('/models/pricing', {
    params: {
      keyword: params.keyword || undefined,
      category: params.category || 'all',
    },
  })
  return transformModelList(data)
}

export async function updateModel(id: string, payload: UpdateModelRequest): Promise<ModelMarketModel> {
  const { data } = await apiClient.put<ModelMarketModel>(`/admin/models/${id}`, payload)
  return data
}

export const modelMarketAPI = {
  getModels,
  updateModel,
}

export default modelMarketAPI
