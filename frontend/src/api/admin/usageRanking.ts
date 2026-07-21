/**
 * Admin usage ranking API.
 *
 * The backend endpoint is reserved as:
 * GET /api/admin/usage/ranking?period=today&page=1&pageSize=20
 */

import { apiClient } from '../client'

export type UsageRankingPeriod = 'today' | 'week' | 'month' | 'custom'

export interface UsageRankingUser {
  rank: number
  email: string
  tokens: number
}

export interface UsageRankingQuery {
  period: UsageRankingPeriod
  page?: number
  pageSize?: number
  startDate?: string
  endDate?: string
}

export interface UsageRankingResponse {
  totalTokens: number
  period: UsageRankingPeriod | string
  updatedAt: string
  topUsers: UsageRankingUser[]
  items: UsageRankingUser[]
  total: number
  page: number
  pageSize: number
  mock?: boolean
}

interface RawUsageRankingResponse {
  totalTokens?: number
  total_tokens?: number
  period?: UsageRankingPeriod | string
  updatedAt?: string
  updated_at?: string
  topUsers?: UsageRankingUser[]
  top_users?: UsageRankingUser[]
  items?: UsageRankingUser[]
  users?: UsageRankingUser[]
  total?: number
  page?: number
  pageSize?: number
  page_size?: number
}

const MOCK_USERS: Omit<UsageRankingUser, 'rank'>[] = [
  { email: '791012705@qq.com', tokens: 1_421_580_000 },
  { email: '309778892@qq.com', tokens: 809_110_000 },
  { email: 'codex-admin0@gmail.com', tokens: 706_690_000 },
  { email: 'devops-1@opspark.com', tokens: 434_620_000 },
  { email: 'delta-d@qmail.com', tokens: 288_980_000 },
  { email: '185996250@qq.com', tokens: 244_390_000 },
  { email: '267788860@qq.com', tokens: 175_510_000 },
  { email: 'service-api@acme.dev', tokens: 154_250_000 },
  { email: 'prod.gateway@sub2api.io', tokens: 131_700_000 },
  { email: 'observer@aihub.example', tokens: 118_460_000 },
  { email: 'ops-bot@quant.team', tokens: 101_180_000 },
  { email: 'workflow@terra.dev', tokens: 95_320_000 },
  { email: 'ai-lab@northstar.ai', tokens: 88_900_000 },
  { email: 'keymaster@river.dev', tokens: 80_240_000 },
  { email: 'finance-api@sample.com', tokens: 72_110_000 },
  { email: 'pilot@blueprint.ai', tokens: 64_430_000 },
  { email: 'support-cx@vendor.io', tokens: 58_720_000 },
  { email: 'agent-router@edge.dev', tokens: 50_480_000 },
  { email: 'image-batch@studio.ai', tokens: 44_960_000 },
  { email: 'qa-runner@sub2api.test', tokens: 39_510_000 },
  { email: 'team-alpha@example.com', tokens: 35_880_000 },
  { email: 'team-beta@example.com', tokens: 32_420_000 },
  { email: 'modelhub@company.dev', tokens: 29_750_000 },
  { email: 'gateway-audit@company.dev', tokens: 25_060_000 },
  { email: 'billing@company.dev', tokens: 22_430_000 },
  { email: 'research@lab.example', tokens: 20_190_000 },
  { email: 'batch-runner@lab.example', tokens: 18_570_000 },
  { email: 'preview@lab.example', tokens: 16_020_000 },
  { email: 'analytics@lab.example', tokens: 14_880_000 },
  { email: 'worker@lab.example', tokens: 12_730_000 },
  { email: 'stage@lab.example', tokens: 10_260_000 },
  { email: 'sandbox@lab.example', tokens: 8_910_000 },
  { email: 'demo@lab.example', tokens: 7_640_000 },
  { email: 'operator@lab.example', tokens: 6_820_000 },
  { email: 'node-1@lab.example', tokens: 5_770_000 },
  { email: 'node-2@lab.example', tokens: 5_210_000 },
  { email: 'node-3@lab.example', tokens: 4_690_000 },
  { email: 'qa-1@lab.example', tokens: 4_120_000 },
  { email: 'qa-2@lab.example', tokens: 3_580_000 },
  { email: 'qa-3@lab.example', tokens: 3_030_000 },
]

const periodMultiplier: Record<UsageRankingPeriod, number> = {
  today: 1,
  week: 3.2,
  month: 7.6,
  custom: 2.4,
}

function shouldUseMockFallback(error: unknown): boolean {
  const status = (error as { status?: number })?.status
  return status === 0 || status === 404
}

function scaleMockUsers(period: UsageRankingPeriod): UsageRankingUser[] {
  const multiplier = periodMultiplier[period] ?? 1
  return MOCK_USERS
    .map((user) => ({
      ...user,
      tokens: Math.round(user.tokens * multiplier),
    }))
    .sort((a, b) => b.tokens - a.tokens)
    .map((user, index) => ({
      ...user,
      rank: index + 1,
    }))
}

function buildMockUsageRanking(query: UsageRankingQuery): UsageRankingResponse {
  const page = query.page || 1
  const pageSize = query.pageSize || 20
  const period = query.period || 'today'
  const users = scaleMockUsers(period)
  const start = (page - 1) * pageSize
  const items = users.slice(start, start + pageSize)

  return {
    totalTokens: Math.round(6_252_930_000 * (periodMultiplier[period] ?? 1)),
    period,
    updatedAt: new Date().toISOString(),
    topUsers: users.slice(0, 3),
    items,
    total: users.length,
    page,
    pageSize,
    mock: true,
  }
}

function normalizeUsageRankingResponse(
  raw: RawUsageRankingResponse,
  query: UsageRankingQuery
): UsageRankingResponse {
  const page = raw.page || query.page || 1
  const pageSize = raw.pageSize || raw.page_size || query.pageSize || 20
  const topUsers = raw.topUsers || raw.top_users || []
  const items = raw.items || raw.users || topUsers

  return {
    totalTokens: raw.totalTokens ?? raw.total_tokens ?? items.reduce((sum, item) => sum + item.tokens, 0),
    period: raw.period || query.period,
    updatedAt: raw.updatedAt || raw.updated_at || new Date().toISOString(),
    topUsers: topUsers.length > 0 ? topUsers : items.slice(0, 3),
    items,
    total: raw.total ?? items.length,
    page,
    pageSize,
  }
}

export async function getUsageRanking(query: UsageRankingQuery): Promise<UsageRankingResponse> {
  const params = {
    period: query.period,
    page: query.page || 1,
    pageSize: query.pageSize || 20,
    start_date: query.startDate,
    end_date: query.endDate,
  }

  try {
    const { data } = await apiClient.get<RawUsageRankingResponse>('/admin/usage/ranking', { params })
    return normalizeUsageRankingResponse(data, query)
  } catch (error) {
    if (shouldUseMockFallback(error)) {
      return buildMockUsageRanking(query)
    }
    throw error
  }
}

export const usageRankingAPI = {
  getUsageRanking,
}

export default usageRankingAPI
