/**
 * Admin model market API.
 *
 * Reserved endpoints:
 * GET /api/admin/models?keyword=&category=
 * PUT /api/admin/models/:id
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
  mock?: boolean
}

export type UpdateModelRequest = Partial<Omit<ModelMarketModel, 'id'>>

interface RawModelListResponse {
  models?: ModelMarketModel[]
  items?: ModelMarketModel[]
  total?: number
  availableChannels?: number
  available_channels?: number
  updatedAt?: string
  updated_at?: string
}

const MOCK_MODELS: ModelMarketModel[] = [
  {
    id: 'gpt-5-6-sol',
    name: 'gpt-5.6-sol',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'recommended',
    platform: 'openai',
    status: 'available',
    recommended: true,
    platformAdapted: true,
    pricing: { input: 5, output: 30, cachedInput: 6.25, cachedOutput: 0.5 },
  },
  {
    id: 'gpt-5-6-terra',
    name: 'gpt-5.6-terra',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'recommended',
    platform: 'openai',
    status: 'available',
    recommended: true,
    platformAdapted: true,
    pricing: { input: 2.5, output: 15, cachedInput: 3.125, cachedOutput: 0.25 },
  },
  {
    id: 'gpt-5-6-luna',
    name: 'gpt-5.6-luna',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'recommended',
    platform: 'openai',
    status: 'available',
    recommended: true,
    platformAdapted: true,
    pricing: { input: 1, output: 6, cachedInput: 1.25, cachedOutput: 0.1 },
  },
  {
    id: 'gpt-5-5',
    name: 'gpt-5.5',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'recommended',
    platform: 'openai',
    status: 'available',
    recommended: true,
    platformAdapted: true,
    pricing: { input: 5, output: 30, cachedInput: 0, cachedOutput: 0.5 },
  },
  {
    id: 'gpt-5-4',
    name: 'gpt-5.4',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'all',
    platform: 'openai',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 2.5, output: 15, cachedInput: 0, cachedOutput: 0.25 },
  },
  {
    id: 'gpt-5-4-mini',
    name: 'gpt-5.4-mini',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'all',
    platform: 'openai',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 0.75, output: 4.5, cachedInput: 0, cachedOutput: 0.075 },
  },
  {
    id: 'gpt-5-3-codex-spark',
    name: 'gpt-5.3-codex-spark',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'platform',
    platform: 'openai',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 1.25, output: 7.5, cachedInput: 0.2, cachedOutput: 0.15 },
  },
  {
    id: 'gpt-image-2',
    name: 'gpt-image-2',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'platform',
    platform: 'openai',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 3, output: 12, cachedInput: 0, cachedOutput: 0 },
  },
  {
    id: 'gpt-5-4-nano',
    name: 'gpt-5.4-nano',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'all',
    platform: 'openai',
    status: 'available',
    pricing: { input: 0.3, output: 1.2, cachedInput: 0, cachedOutput: 0.03 },
  },
  {
    id: 'o4-reasoning',
    name: 'o4-reasoning',
    type: 'OFFICIAL',
    typeLabel: 'OpenAI',
    category: 'all',
    platform: 'openai',
    status: 'available',
    pricing: { input: 8, output: 40, cachedInput: 1, cachedOutput: 0.8 },
  },
  {
    id: 'claude-sonnet-4-6',
    name: 'claude-sonnet-4.6',
    type: 'ADAPTED',
    typeLabel: 'Anthropic',
    category: 'platform',
    platform: 'anthropic',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 3, output: 15, cachedInput: 3.75, cachedOutput: 0.3 },
  },
  {
    id: 'gemini-3-1-pro',
    name: 'gemini-3.1-pro',
    type: 'ADAPTED',
    typeLabel: 'Gemini',
    category: 'platform',
    platform: 'gemini',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 1.25, output: 10, cachedInput: 0.31, cachedOutput: 0.25 },
  },
  {
    id: 'grok-5-fast',
    name: 'grok-5-fast',
    type: 'ADAPTED',
    typeLabel: 'Grok',
    category: 'all',
    platform: 'grok',
    status: 'available',
    pricing: { input: 2, output: 10, cachedInput: 0.5, cachedOutput: 0.2 },
  },
  {
    id: 'codex-router-auto',
    name: 'codex-router-auto',
    type: 'ADAPTED',
    typeLabel: 'Sub2API',
    category: 'platform',
    platform: 'openai',
    status: 'available',
    platformAdapted: true,
    pricing: { input: 0.6, output: 3.2, cachedInput: 0.12, cachedOutput: 0.06 },
  },
]

function shouldUseMockFallback(error: unknown): boolean {
  const status = (error as { status?: number })?.status
  return status === 0 || status === 404
}

function matchesCategory(model: ModelMarketModel, category?: string): boolean {
  if (!category || category === 'all') return true
  if (category === 'recommended') return model.recommended === true || model.category === 'recommended'
  if (category === 'platform') return model.platformAdapted === true || model.category === 'platform'
  return model.category === category
}

function buildMockModelList(params: ModelListParams = {}): ModelListResponse {
  const keyword = params.keyword?.trim().toLowerCase()
  const models = MOCK_MODELS.filter((model) => {
    const matchesKeyword = !keyword || model.name.toLowerCase().includes(keyword)
    return matchesKeyword && matchesCategory(model, params.category)
  })

  return {
    models,
    total: models.length,
    availableChannels: 2,
    updatedAt: new Date().toISOString(),
    mock: true,
  }
}

function normalizeModelListResponse(raw: RawModelListResponse): ModelListResponse {
  const models = raw.models || raw.items || []
  return {
    models,
    total: raw.total ?? models.length,
    availableChannels: raw.availableChannels ?? raw.available_channels ?? 0,
    updatedAt: raw.updatedAt || raw.updated_at || new Date().toISOString(),
  }
}

export async function getModels(params: ModelListParams = {}): Promise<ModelListResponse> {
  try {
    const { data } = await apiClient.get<RawModelListResponse>('/admin/models', { params })
    return normalizeModelListResponse(data)
  } catch (error) {
    if (shouldUseMockFallback(error)) {
      return buildMockModelList(params)
    }
    throw error
  }
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
