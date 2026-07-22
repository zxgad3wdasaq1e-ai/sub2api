/**
 * Admin model market API.
 *
 * Calls the real backend endpoint:
 * GET /api/v1/admin/models?keyword=
 * PUT /api/v1/admin/models/:id
 */

import { apiClient } from '../client'

export type ModelMarketCategory = 'recommended' | 'all' | 'platform'

export interface ModelPricing {
  input: number
  output: number
  cachedInput: number
  cachedOutput: number
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

/** Raw response from GET /api/v1/admin/models */
interface RawModelMarketEntry {
  id: string
  name: string
  type: string
  type_label?: string
  platform: string
  input_price: number
  output_price: number
  cache_write_price: number
  cache_read_price: number
  channel_count: number
}

interface RawModelListResponse {
  models?: RawModelMarketEntry[]
  total?: number
  available_channels?: number
}

/** Platform → display type/label mapping */
const platformTypeLabelMap: Record<string, { type: string; label: string }> = {
  anthropic: { type: 'ANTHROPIC', label: 'Anthropic' },
  openai: { type: 'OPENAI', label: 'OpenAI' },
  google: { type: 'GEMINI', label: 'Gemini' },
  gemini: { type: 'GEMINI', label: 'Gemini' },
  grok: { type: 'GROK', label: 'Grok' },
  xai: { type: 'GROK', label: 'Grok' },
}

function getTypeInfo(platform: string): { type: string; label: string } {
  return platformTypeLabelMap[platform] || { type: 'ADAPTED', label: platform }
}

function transformModelList(raw: RawModelListResponse): ModelListResponse {
  const models: ModelMarketModel[] = (raw.models || []).map((entry) => {
    const { type, label } = getTypeInfo(entry.platform)
    return {
      id: entry.id,
      name: entry.name,
      type,
      typeLabel: label,
      category: 'all',
      platform: entry.platform,
      status: 'available' as const,
      platformAdapted: true,
      pricing: {
        input: entry.input_price,
        output: entry.output_price,
        cachedInput: entry.cache_write_price,
        cachedOutput: entry.cache_read_price,
      },
      channelCount: entry.channel_count,
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
  const { data } = await apiClient.get<RawModelListResponse>('/admin/models', {
    params: { keyword: params.keyword || undefined },
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
