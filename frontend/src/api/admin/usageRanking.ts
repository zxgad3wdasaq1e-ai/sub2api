/**
 * Admin usage ranking API.
 *
 * Calls the real backend endpoint:
 * GET /api/v1/admin/dashboard/users-ranking?start_date=YYYY-MM-DD&end_date=YYYY-MM-DD&limit=N
 */

import { apiClient } from '../client'

export type UsageRankingPeriod = 'today' | 'week' | 'month' | 'custom'

export interface UsageRankingUser {
  rank: number
  email: string
  tokens: number
  actualCost: number
  requests: number
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
  totalActualCost: number
  totalRequests: number
  period: UsageRankingPeriod | string
  startDate: string
  endDate: string
  updatedAt: string
  topUsers: UsageRankingUser[]
  items: UsageRankingUser[]
  total: number
  page: number
  pageSize: number
}

/** Raw response from GET /api/v1/admin/dashboard/users-ranking */
interface RawRankingItem {
  user_id: number
  email: string
  actual_cost: number
  requests: number
  tokens: number
}

interface RawRankingResponse {
  ranking?: RawRankingItem[]
  total_actual_cost?: number
  total_requests?: number
  total_tokens?: number
  start_date?: string
  end_date?: string
}

/**
 * Compute date range from period.
 */
function periodToDateRange(period: UsageRankingPeriod, customStart?: string, customEnd?: string): { start: string; end: string } {
  const fmt = (d: Date): string => {
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
  }

  const now = new Date()
  const today = fmt(now)

  if (period === 'custom' && customStart && customEnd) {
    return { start: customStart, end: customEnd }
  }

  switch (period) {
    case 'today':
      return { start: today, end: today }
    case 'week': {
      const start = new Date(now)
      const day = start.getDay()
      const diff = day === 0 ? -6 : 1 - day
      start.setDate(start.getDate() + diff)
      return { start: fmt(start), end: today }
    }
    case 'month': {
      const start = new Date(now.getFullYear(), now.getMonth(), 1)
      return { start: fmt(start), end: today }
    }
    default:
      return { start: today, end: today }
  }
}

function transformRankingResponse(raw: RawRankingResponse, query: UsageRankingQuery): UsageRankingResponse {
  const rankingItems = raw.ranking || []
  const allUsers: UsageRankingUser[] = rankingItems.map((item, index) => ({
    rank: index + 1,
    email: item.email,
    tokens: item.tokens,
    actualCost: item.actual_cost,
    requests: item.requests,
  }))

  const page = query.page || 1
  const pageSize = query.pageSize || 20
  const total = allUsers.length
  const startIdx = (page - 1) * pageSize
  const pagedItems = allUsers.slice(startIdx, startIdx + pageSize)

  const dateRange = periodToDateRange(query.period, query.startDate, query.endDate)

  return {
    totalTokens: raw.total_tokens ?? 0,
    totalActualCost: raw.total_actual_cost ?? 0,
    totalRequests: raw.total_requests ?? 0,
    period: query.period,
    startDate: raw.start_date || dateRange.start,
    endDate: raw.end_date || dateRange.end,
    updatedAt: new Date().toISOString(),
    topUsers: allUsers.slice(0, 3),
    items: pagedItems,
    total,
    page,
    pageSize,
  }
}

export async function getUsageRanking(query: UsageRankingQuery): Promise<UsageRankingResponse> {
  const { start, end } = periodToDateRange(query.period, query.startDate, query.endDate)
  const limit = 50

  const { data } = await apiClient.get<RawRankingResponse>('/admin/dashboard/users-ranking', {
    params: {
      start_date: start,
      end_date: end,
      limit,
    },
  })

  return transformRankingResponse(data, query)
}

export const usageRankingAPI = {
  getUsageRanking,
}

export default usageRankingAPI
