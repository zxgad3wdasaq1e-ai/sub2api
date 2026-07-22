<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="rounded-2xl border border-cyan-100 bg-gradient-to-br from-cyan-50 via-white to-emerald-50 p-5 shadow-card dark:border-cyan-500/20 dark:from-cyan-950/20 dark:via-dark-800 dark:to-emerald-950/10">
        <div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
          <div class="max-w-3xl">
            <div class="inline-flex items-center gap-2 rounded-full bg-white/80 px-3 py-1 text-xs font-bold uppercase tracking-[0.16em] text-teal-700 ring-1 ring-teal-100 dark:bg-dark-900/60 dark:text-teal-300 dark:ring-teal-500/20">
              <Icon name="beaker" size="sm" />
              {{ t('admin.modelMarket.heroBadge') }}
            </div>
            <h1 class="mt-4 text-3xl font-black tracking-normal text-gray-950 dark:text-white">
              {{ t('admin.modelMarket.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">
              {{ t('admin.modelMarket.description') }}
            </p>
          </div>

          <div class="grid grid-cols-2 gap-3 sm:min-w-[220px]">
            <div class="rounded-2xl border border-white/80 bg-white/80 p-4 shadow-sm dark:border-white/10 dark:bg-dark-900/60">
              <p class="text-xs font-bold text-gray-500 dark:text-gray-400">{{ t('admin.modelMarket.availableModels') }}</p>
              <p class="mt-1 text-2xl font-black text-gray-950 dark:text-white">{{ responseTotal }}</p>
            </div>
            <div class="rounded-2xl border border-white/80 bg-white/80 p-4 shadow-sm dark:border-white/10 dark:bg-dark-900/60">
              <p class="text-xs font-bold text-gray-500 dark:text-gray-400">{{ t('admin.modelMarket.availableChannels') }}</p>
              <p class="mt-1 text-2xl font-black text-teal-700 dark:text-teal-300">{{ availableChannels }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="card p-4">
        <div class="flex flex-col gap-4 xl:flex-row xl:items-center">
          <div class="relative w-full xl:max-w-md">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="keyword"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.modelMarket.searchPlaceholder')"
            />
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button
              v-for="option in categoryOptions"
              :key="option.value"
              type="button"
              class="rounded-full px-3 py-1.5 text-sm font-semibold transition-colors"
              :class="category === option.value
                ? 'bg-primary-50 text-primary-700 ring-1 ring-primary-100 dark:bg-primary-500/10 dark:text-primary-300 dark:ring-primary-500/20'
                : 'bg-gray-50 text-gray-600 hover:bg-gray-100 dark:bg-dark-900 dark:text-gray-300 dark:hover:bg-dark-700'"
              @click="setCategory(option.value)"
            >
              {{ option.label }}
            </button>
          </div>

          <div class="flex flex-wrap items-center gap-3 xl:ml-auto">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">
              {{ t('admin.modelMarket.modelCount', { count: responseTotal }) }}
            </span>
            <div class="inline-flex rounded-xl border border-gray-200 bg-white p-1 dark:border-dark-700 dark:bg-dark-900">
              <button
                type="button"
                class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors"
                :class="viewMode === 'grid' ? 'bg-primary-50 text-primary-700 dark:bg-primary-500/10 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-dark-800'"
                :title="t('admin.modelMarket.gridView')"
                @click="viewMode = 'grid'"
              >
                <Icon name="grid" size="sm" />
              </button>
              <button
                type="button"
                class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors"
                :class="viewMode === 'list' ? 'bg-primary-50 text-primary-700 dark:bg-primary-500/10 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-dark-800'"
                :title="t('admin.modelMarket.listView')"
                @click="viewMode = 'list'"
              >
                <Icon name="list" size="sm" />
              </button>
            </div>
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadModels">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              {{ t('common.refresh') }}
            </button>
          </div>
        </div>
      </section>

      <div v-if="loading && models.length === 0" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <div v-if="viewMode === 'grid'" class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <ModelPricingCard
            v-for="model in models"
            :key="model.id"
            :model="model"
            @configure="handleConfigure"
          />
        </div>

        <section v-else class="card overflow-hidden">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.columns.model') }}
                  </th>
                  <th class="px-5 py-3 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.columns.type') }}
                  </th>
                  <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.pricing.input') }}
                  </th>
                  <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.pricing.output') }}
                  </th>
                  <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.pricing.cachedInput') }}
                  </th>
                  <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('admin.modelMarket.pricing.cachedOutput') }}
                  </th>
                  <th class="px-5 py-3 text-right text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
                    {{ t('common.actions') }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="models.length === 0">
                  <td colspan="7" class="px-5 py-12">
                    <EmptyState :title="t('admin.modelMarket.emptyTitle')" :description="t('admin.modelMarket.emptyDescription')" />
                  </td>
                </tr>
                <tr v-for="model in models" v-else :key="model.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/70">
                  <td class="whitespace-nowrap px-5 py-4">
                    <div class="font-bold text-gray-950 dark:text-white">{{ model.name }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">/ 1M TOKEN</div>
                  </td>
                  <td class="whitespace-nowrap px-5 py-4">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-bold uppercase text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300">{{ model.type }}</span>
                      <span v-if="model.typeLabel" class="rounded-full bg-sky-50 px-2 py-0.5 text-xs font-semibold text-sky-700 dark:bg-sky-500/10 dark:text-sky-300">{{ model.typeLabel }}</span>
                    </div>
                  </td>
                  <td class="whitespace-nowrap px-5 py-4 text-right font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatPrice(model.pricing.input) }}</td>
                  <td class="whitespace-nowrap px-5 py-4 text-right font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatPrice(model.pricing.output) }}</td>
                  <td class="whitespace-nowrap px-5 py-4 text-right font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatPrice(model.pricing.cachedInput) }}</td>
                  <td class="whitespace-nowrap px-5 py-4 text-right font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatPrice(model.pricing.cachedOutput) }}</td>
                  <td class="whitespace-nowrap px-5 py-4 text-right">
                    <button type="button" class="btn btn-secondary btn-sm" @click="handleConfigure(model)">
                      <Icon name="cog" size="sm" />
                      {{ t('common.settings') }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>

      <EmptyState
        v-if="!loading && models.length === 0 && viewMode === 'grid'"
        :title="t('admin.modelMarket.emptyTitle')"
        :description="t('admin.modelMarket.emptyDescription')"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { getModels, type ModelMarketCategory, type ModelMarketModel } from '@/api/admin/models'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelPricingCard from '@/components/admin/model/ModelPricingCard.vue'

type ViewMode = 'grid' | 'list'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

const keyword = ref('')
const category = ref<ModelMarketCategory>('all')
const viewMode = ref<ViewMode>('grid')
const models = ref<ModelMarketModel[]>([])
const loading = ref(false)
const responseTotal = ref(0)
const availableChannels = ref(0)
let requestSeq = 0
let searchTimer: number | null = null

const categoryOptions = computed(() => [
  { value: 'recommended' as const, label: t('admin.modelMarket.categories.recommended') },
  { value: 'all' as const, label: t('admin.modelMarket.categories.all') },
  { value: 'platform' as const, label: t('admin.modelMarket.categories.platform') },
])

function formatPrice(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-'
  return `$${value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')}`
}

async function loadModels(): Promise<void> {
  const seq = ++requestSeq
  loading.value = true
  try {
    const result = await getModels({
      keyword: keyword.value,
      category: category.value,
    })
    if (seq !== requestSeq) return
    models.value = result.models
    responseTotal.value = result.total
    availableChannels.value = result.availableChannels
  } catch (error) {
    if (seq !== requestSeq) return
    const message = (error as { message?: string })?.message || t('admin.modelMarket.loadFailed')
    appStore.showError(message)
  } finally {
    if (seq === requestSeq) loading.value = false
  }
}

function setCategory(nextCategory: ModelMarketCategory): void {
  if (category.value === nextCategory) return
  category.value = nextCategory
}

function handleConfigure(model: ModelMarketModel): void {
  void router.push({
    name: 'AdminChannels',
    query: { model: model.name, platform: model.platform },
  })
}

watch(category, () => {
  loadModels()
})

watch(keyword, () => {
  if (searchTimer) window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    loadModels()
  }, 250)
})

onMounted(() => {
  loadModels()
})
</script>
