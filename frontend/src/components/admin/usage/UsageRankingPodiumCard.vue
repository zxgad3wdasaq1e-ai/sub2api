<template>
  <article
    class="relative overflow-hidden rounded-2xl border p-5 shadow-card transition-all duration-300 hover:-translate-y-0.5 hover:shadow-card-hover"
    :class="toneClasses[tone]"
  >
    <div class="pointer-events-none absolute inset-x-0 top-0 h-1/2 bg-white/35 dark:bg-white/5"></div>
    <div class="relative space-y-5">
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <p class="text-xs font-bold uppercase tracking-[0.22em] text-gray-500 dark:text-gray-400">
            {{ t('admin.usageRanking.topRank', { rank }) }}
          </p>
          <h3 class="mt-2 truncate text-lg font-bold text-gray-900 dark:text-white" :title="email">
            {{ maskedEmail }}
          </h3>
        </div>
        <div class="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-2xl bg-white text-xl font-black text-gray-900 shadow-sm dark:bg-dark-900 dark:text-white">
          {{ rank }}
        </div>
      </div>

      <div class="rounded-2xl border border-white/70 bg-white/80 p-4 dark:border-white/10 dark:bg-dark-900/70">
        <p class="text-xs font-bold uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">
          {{ t('admin.usageRanking.tokenUsage') }}
        </p>
        <p class="mt-2 text-2xl font-black tabular-nums text-gray-950 dark:text-white">
          {{ displayTokens }}
        </p>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

type PodiumTone = 'gold' | 'silver' | 'bronze'

defineProps<{
  rank: number
  email: string
  maskedEmail: string
  displayTokens: string
  tone: PodiumTone
}>()

const { t } = useI18n()

const toneClasses: Record<PodiumTone, string> = {
  gold: 'border-amber-200 bg-gradient-to-br from-amber-50 via-white to-orange-50 dark:border-amber-500/25 dark:from-amber-950/30 dark:via-dark-800 dark:to-orange-950/20',
  silver: 'border-sky-200 bg-gradient-to-br from-sky-50 via-white to-slate-50 dark:border-sky-500/25 dark:from-sky-950/30 dark:via-dark-800 dark:to-slate-900/40',
  bronze: 'border-emerald-200 bg-gradient-to-br from-emerald-50 via-white to-teal-50 dark:border-emerald-500/25 dark:from-emerald-950/30 dark:via-dark-800 dark:to-teal-950/20',
}
</script>
