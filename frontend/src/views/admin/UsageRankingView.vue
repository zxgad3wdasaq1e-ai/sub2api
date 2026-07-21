<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-2xl border border-teal-100 bg-gradient-to-br from-slate-950 via-teal-950 to-sky-950 p-5 text-white shadow-card dark:border-teal-500/20">
        <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <div class="inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/10 px-3 py-1 text-xs font-bold uppercase tracking-[0.18em] text-teal-100">
              <Icon name="chartBar" size="sm" />
              {{ t('admin.usageRanking.heroBadge') }}
            </div>
            <h1 class="mt-4 text-3xl font-black tracking-normal text-white">
              {{ t('admin.usageRanking.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-teal-50/85">
              {{ t('admin.usageRanking.heroDescription') }}
            </p>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <div class="w-36">
              <Select v-model="period" :options="periodOptions" />
            </div>
            <div v-if="period === 'custom'" class="min-w-[280px]">
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="handleDateRangeChange"
              />
            </div>
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadRanking">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </div>
      </section>

      <section class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-xs font-bold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
              {{ t('admin.usageRanking.totalTokenUsage') }}
            </p>
            <p class="mt-2 text-4xl font-black tabular-nums text-gray-950 dark:text-white">
              {{ formatTokens(ranking?.totalTokens || 0) }}
            </p>
            <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.usageRanking.rangeSummary', { range: periodRangeText, updated: formattedUpdatedAt }) }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-sm">
            <span class="rounded-full bg-primary-50 px-3 py-1 font-medium text-primary-700 dark:bg-primary-500/10 dark:text-primary-300">
              {{ selectedPeriodLabel }}
            </span>
            <span v-if="ranking?.mock" class="rounded-full bg-amber-50 px-3 py-1 font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">
              {{ t('admin.usageRanking.mockData') }}
            </span>
          </div>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <UsageRankingPodiumCard
          v-for="(user, index) in topUsers"
          :key="user.rank"
          :rank="user.rank"
          :email="user.email"
          :masked-email="maskEmail(user.email)"
          :display-tokens="formatTokens(user.tokens)"
          :tone="podiumTones[index] || 'bronze'"
        />
      </div>

      <section class="card overflow-hidden">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700/50 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-bold text-gray-950 dark:text-white">
              {{ t('admin.usageRanking.topUsersTitle') }}
            </h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.usageRanking.topUsersSubtitle') }}
            </p>
          </div>
          <span class="self-start rounded-full bg-teal-50 px-3 py-1 text-xs font-semibold text-teal-700 dark:bg-teal-500/10 dark:text-teal-300 sm:self-auto">
            {{ selectedPeriodLabel }}
          </span>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="w-28 px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.usageRanking.columns.rank') }}
                </th>
                <th class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.usageRanking.columns.user') }}
                </th>
                <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                  {{ t('admin.usageRanking.columns.tokens') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td colspan="3" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="tableUsers.length === 0">
                <td colspan="3" class="px-5 py-12">
                  <EmptyState :title="t('admin.usageRanking.emptyTitle')" :description="t('admin.usageRanking.emptyDescription')" />
                </td>
              </tr>
              <tr v-for="user in tableUsers" v-else :key="`${user.rank}-${user.email}`" class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/70">
                <td class="px-5 py-4">
                  <span class="inline-flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold tabular-nums" :class="rankBadgeClass(user.rank)">
                    {{ user.rank }}
                  </span>
                </td>
                <td class="px-5 py-4">
                  <div class="max-w-[360px] truncate text-sm font-semibold text-gray-900 dark:text-white" :title="user.email">
                    {{ maskEmail(user.email) }}
                  </div>
                  <div class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {{ t('admin.usageRanking.emailMasked') }}
                  </div>
                </td>
                <td class="px-5 py-4 text-right text-base font-black tabular-nums text-gray-950 dark:text-white">
                  {{ formatTokens(user.tokens) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="(ranking?.total || 0) > 0"
          :page="page"
          :total="ranking?.total || 0"
          :page-size="pageSize"
          :page-size-options="[20, 50, 100]"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getUsageRanking, type UsageRankingPeriod, type UsageRankingResponse } from '@/api/admin/usageRanking'
import { useAppStore } from '@/stores/app'
import { formatCompactNumber } from '@/utils/format'
import AppLayout from '@/components/layout/AppLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import UsageRankingPodiumCard from '@/components/admin/usage/UsageRankingPodiumCard.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()

const period = ref<UsageRankingPeriod>('today')
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const ranking = ref<UsageRankingResponse | null>(null)
const startDate = ref(formatDateString(new Date()))
const endDate = ref(formatDateString(new Date()))
const podiumTones = ['gold', 'silver', 'bronze'] as const
let requestSeq = 0
let refreshTimer: number | null = null

const periodOptions = computed(() => [
  { value: 'today', label: t('admin.usageRanking.period.today') },
  { value: 'week', label: t('admin.usageRanking.period.week') },
  { value: 'month', label: t('admin.usageRanking.period.month') },
  { value: 'custom', label: t('admin.usageRanking.period.custom') },
])

const selectedPeriodLabel = computed(() => {
  return periodOptions.value.find((item) => item.value === period.value)?.label || period.value
})

const topUsers = computed(() => ranking.value?.topUsers || [])
const tableUsers = computed(() => ranking.value?.items || [])

const formattedUpdatedAt = computed(() => {
  if (!ranking.value?.updatedAt) return '-'
  const date = new Date(ranking.value.updatedAt)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
})

const periodRangeText = computed(() => {
  if (startDate.value === endDate.value) {
    return `${selectedPeriodLabel.value} ${startDate.value}`
  }
  return `${selectedPeriodLabel.value} ${startDate.value} - ${endDate.value}`
})

function formatDateString(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function syncPeriodRange(nextPeriod: UsageRankingPeriod): void {
  const now = new Date()
  endDate.value = formatDateString(now)

  if (nextPeriod === 'today') {
    startDate.value = formatDateString(now)
    return
  }

  if (nextPeriod === 'week') {
    const start = new Date(now)
    const day = start.getDay()
    const diff = day === 0 ? -6 : 1 - day
    start.setDate(start.getDate() + diff)
    startDate.value = formatDateString(start)
    return
  }

  if (nextPeriod === 'month') {
    startDate.value = formatDateString(new Date(now.getFullYear(), now.getMonth(), 1))
  }
}

function formatTokens(value: number): string {
  if (!Number.isFinite(value)) return '0'
  if (locale.value.startsWith('zh') && Math.abs(value) >= 10_000) {
    return `${Math.round(value / 10_000).toLocaleString()}万`
  }
  return formatCompactNumber(value)
}

function maskEmail(email: string): string {
  const [localPart, domainPart = ''] = email.split('@')
  if (!localPart || !domainPart) return email

  const firstLocal = localPart.charAt(0)
  const lastLocal = localPart.length > 1 ? localPart.charAt(localPart.length - 1) : ''
  const localStars = '*'.repeat(Math.max(3, Math.min(8, localPart.length - 2)))
  const domainSegments = domainPart.split('.')
  const domainName = domainSegments.shift() || ''
  const suffix = domainSegments.length > 0 ? `.${domainSegments.join('.')}` : ''
  const maskedDomain = domainName.length <= 1
    ? '*'
    : `${domainName.charAt(0)}${'*'.repeat(Math.max(1, Math.min(3, domainName.length - 1)))}`

  return `${firstLocal}${localStars}${lastLocal}@${maskedDomain}${suffix}`
}

function rankBadgeClass(rank: number): string {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-300'
  if (rank === 2) return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-500/20 dark:text-cyan-300'
  if (rank === 3) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300'
}

async function loadRanking(): Promise<void> {
  const seq = ++requestSeq
  loading.value = true
  try {
    const result = await getUsageRanking({
      period: period.value,
      page: page.value,
      pageSize: pageSize.value,
      startDate: period.value === 'custom' ? startDate.value : undefined,
      endDate: period.value === 'custom' ? endDate.value : undefined,
    })
    if (seq !== requestSeq) return
    ranking.value = result
  } catch (error) {
    if (seq !== requestSeq) return
    const message = (error as { message?: string })?.message || t('admin.usageRanking.loadFailed')
    appStore.showError(message)
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

function handleDateRangeChange(): void {
  if (period.value !== 'custom') return
  page.value = 1
  loadRanking()
}

function handlePageChange(nextPage: number): void {
  page.value = nextPage
  loadRanking()
}

function handlePageSizeChange(nextPageSize: number): void {
  pageSize.value = nextPageSize
  page.value = 1
  loadRanking()
}

watch(period, (nextPeriod) => {
  page.value = 1
  if (nextPeriod !== 'custom') {
    syncPeriodRange(nextPeriod)
  }
  loadRanking()
})

onMounted(() => {
  syncPeriodRange(period.value)
  loadRanking()
  refreshTimer = window.setInterval(() => {
    if (!loading.value) loadRanking()
  }, 30_000)
})

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>
