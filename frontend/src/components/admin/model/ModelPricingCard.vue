<template>
  <article class="card card-hover p-4">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <h3 class="truncate text-2xl font-black text-gray-950 dark:text-white" :title="model.name">
          {{ model.name }}
        </h3>
        <div class="mt-3 flex flex-wrap items-center gap-2">
          <span class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-bold uppercase text-emerald-700 ring-1 ring-emerald-100 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/20">
            {{ model.type }}
          </span>
          <span class="inline-flex items-center rounded-full bg-gray-50 px-2 py-0.5 text-xs font-semibold text-gray-600 ring-1 ring-gray-100 dark:bg-dark-900 dark:text-gray-300 dark:ring-dark-700">
            {{ typeText }}
          </span>
          <span v-if="model.typeLabel" class="inline-flex items-center rounded-full bg-sky-50 px-2 py-0.5 text-xs font-semibold text-sky-700 ring-1 ring-sky-100 dark:bg-sky-500/10 dark:text-sky-300 dark:ring-sky-500/20">
            {{ model.typeLabel }}
          </span>
        </div>
      </div>

      <span
        class="inline-flex flex-shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-semibold"
        :class="model.status === 'disabled'
          ? 'bg-gray-100 text-gray-500 dark:bg-dark-900 dark:text-gray-400'
          : 'bg-teal-50 text-teal-700 ring-1 ring-teal-100 dark:bg-teal-500/10 dark:text-teal-300 dark:ring-teal-500/20'"
      >
        {{ model.status === 'disabled' ? t('common.disabled') : t('common.available') }}
      </span>
    </div>

    <div class="mt-5 flex items-center justify-between gap-3">
      <p class="text-xs font-bold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
        {{ t('admin.modelMarket.tokenPricing') }}
      </p>
      <p class="text-xs font-bold uppercase tracking-[0.18em] text-gray-400 dark:text-gray-500">
        / 1M TOKEN
      </p>
    </div>

    <div class="mt-3 grid grid-cols-2 gap-3 xl:grid-cols-4">
      <ModelPriceMetric :label="t('admin.modelMarket.pricing.input')" :value="model.pricing.input" tone="input" />
      <ModelPriceMetric :label="t('admin.modelMarket.pricing.output')" :value="model.pricing.output" tone="output" />
      <ModelPriceMetric :label="t('admin.modelMarket.pricing.cachedInput')" :value="model.pricing.cachedInput" tone="cachedInput" />
      <ModelPriceMetric :label="t('admin.modelMarket.pricing.cachedOutput')" :value="model.pricing.cachedOutput" tone="cachedOutput" />
    </div>

    <div class="mt-4 flex justify-end">
      <button type="button" class="btn btn-secondary btn-sm" @click="$emit('configure', model)">
        <Icon name="cog" size="sm" />
        {{ t('common.settings') }}
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelMarketModel } from '@/api/admin/models'
import Icon from '@/components/icons/Icon.vue'
import ModelPriceMetric from './ModelPriceMetric.vue'

const props = defineProps<{
  model: ModelMarketModel
}>()

defineEmits<{
  configure: [model: ModelMarketModel]
}>()

const { t } = useI18n()

const typeText = computed(() => {
  if (props.model.type === 'OFFICIAL') return t('admin.modelMarket.types.official')
  if (props.model.type === 'ADAPTED') return t('admin.modelMarket.types.adapted')
  return props.model.typeLabel || props.model.type
})
</script>
