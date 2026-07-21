<template>
  <div class="rounded-xl border p-3" :class="toneClasses[tone]">
    <p class="truncate text-xs font-semibold text-gray-500 dark:text-gray-400" :title="label">
      {{ label }}
    </p>
    <p class="mt-1 text-lg font-black tabular-nums text-gray-950 dark:text-white">
      {{ formattedValue }}
    </p>
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ unit }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type MetricTone = 'input' | 'output' | 'cachedInput' | 'cachedOutput'

const props = withDefaults(defineProps<{
  label: string
  value: number | null | undefined
  tone: MetricTone
  unit?: string
}>(), {
  unit: '/ 1M token',
})

const toneClasses: Record<MetricTone, string> = {
  input: 'border-emerald-100 bg-emerald-50/50 dark:border-emerald-500/20 dark:bg-emerald-950/10',
  output: 'border-amber-100 bg-amber-50/55 dark:border-amber-500/20 dark:bg-amber-950/10',
  cachedInput: 'border-gray-100 bg-gray-50/70 dark:border-dark-700 dark:bg-dark-900/40',
  cachedOutput: 'border-cyan-100 bg-cyan-50/45 dark:border-cyan-500/20 dark:bg-cyan-950/10',
}

const formattedValue = computed(() => {
  if (props.value === null || props.value === undefined || !Number.isFinite(props.value)) return '-'
  const raw = props.value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')
  return `$${raw}`
})
</script>
