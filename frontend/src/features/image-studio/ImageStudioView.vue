<template>
  <AppLayout>
    <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-card dark:border-dark-700 dark:bg-dark-900">
      <header class="flex flex-col gap-4 border-b border-gray-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700">
        <div class="flex items-center">
          <div>
            <h1 class="text-lg font-bold text-gray-950 dark:text-white">AI 生图工作台</h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ selectedApiKey ? `${selectedApiKey.name} · ${form.model}` : modelBindingMessage }}</p>
          </div>
        </div>
        <div class="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
          <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-emerald-500"></i>运行 {{ activeCount }}</span>
          <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-amber-400"></i>等待 {{ queuedCount }}</span>
          <span>图库保留 7 天，请尽快下载</span>
        </div>
      </header>

      <div class="grid min-h-[680px] grid-cols-1 xl:grid-cols-[380px_minmax(0,1fr)]">
        <section class="border-b border-gray-200 bg-gray-50 p-5 xl:border-b-0 xl:border-r dark:border-dark-700 dark:bg-dark-950/60">
          <div class="mb-5 grid grid-cols-2 overflow-hidden rounded-md border border-gray-300 dark:border-dark-600">
            <button type="button" class="flex h-10 items-center justify-center gap-2 text-sm font-medium" :class="form.mode === 'text' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-950' : 'bg-white text-gray-600 dark:bg-dark-900 dark:text-gray-300'" @click="form.mode = 'text'">
              <Icon name="sparkles" size="sm" />文生图
            </button>
            <button type="button" class="flex h-10 items-center justify-center gap-2 border-l border-gray-300 text-sm font-medium dark:border-dark-600" :class="form.mode === 'edit' ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-950' : 'bg-white text-gray-600 dark:bg-dark-900 dark:text-gray-300'" @click="form.mode = 'edit'">
              <Icon name="edit" size="sm" />图生图
            </button>
          </div>

          <form class="space-y-4" @submit.prevent="submitJobs">
            <label class="block">
              <span class="input-label">API 密钥</span>
              <select v-model.number="form.apiKeyId" class="input rounded-md" :disabled="loadingKeys || !compatibleImageKeys.length">
                <option :value="0">{{ loadingKeys ? '正在读取密钥...' : '自动选择可用密钥' }}</option>
                <option v-for="key in compatibleImageKeys" :key="key.id" :value="key.id">{{ key.name || `API Key #${key.id}` }}</option>
              </select>
            </label>

            <label class="block">
              <span class="input-label">模型</span>
              <select v-model="form.model" class="input rounded-md" :disabled="!imageModels.length">
                <option v-if="!imageModels.length" value="">暂无已定价生图模型</option>
                <option v-for="model in imageModels" :key="model" :value="model">{{ model }}</option>
              </select>
              <p v-if="modelBindingMessage" class="mt-2 text-xs text-amber-600 dark:text-amber-400">{{ modelBindingMessage }}</p>
            </label>

            <label class="block">
              <span class="input-label flex items-center justify-between"><span>{{ form.mode === 'edit' ? '修改要求' : '画面描述' }}</span><span class="text-xs font-normal text-gray-400">{{ form.prompt.length }} / 4000</span></span>
              <textarea v-model="form.prompt" class="input min-h-32 resize-y rounded-md leading-6" maxlength="4000" :placeholder="form.mode === 'edit' ? '保留主体构图，将背景替换为雨夜东京街头' : '雨后的上海弄堂，电影感夜景，湿润路面映出霓虹灯光'" required></textarea>
            </label>

            <div v-if="form.mode === 'edit'" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="input-label mb-0">参考图片</span>
                <span class="text-xs text-gray-400">{{ references.length }} / {{ referenceLimit }}</span>
              </div>
              <input ref="referenceInput" class="hidden" type="file" accept="image/png,image/jpeg,image/webp" multiple @change="addReferences" />
              <button type="button" class="flex min-h-20 w-full items-center justify-center gap-2 rounded-md border border-dashed border-gray-300 bg-white text-sm text-gray-500 transition-colors hover:border-primary-500 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-400" :disabled="referenceLimit === 0 || references.length >= referenceLimit" @click="referenceInput?.click()">
                <Icon name="upload" size="md" />
                {{ referenceLimit === 0 ? '当前模型不支持图生图' : references.length ? '继续添加参考图' : '选择参考图片' }}
              </button>
              <div v-if="references.length" class="grid grid-cols-4 gap-2">
                <div v-for="reference in references" :key="reference.id" class="relative aspect-square overflow-hidden rounded-md border border-gray-200 dark:border-dark-700">
                  <img :src="reference.previewUrl" :alt="reference.file.name" class="h-full w-full object-cover" />
                  <button type="button" class="absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-black/70 text-white" :title="`移除 ${reference.file.name}`" @click="removeReference(reference.id)">
                    <Icon name="x" size="xs" />
                  </button>
                </div>
              </div>
            </div>

            <fieldset>
              <legend class="input-label">选择比例</legend>
              <div class="overflow-x-auto rounded-md border border-gray-300 bg-white dark:border-dark-600 dark:bg-dark-900">
                <div class="grid min-w-[342px] grid-cols-9">
                <label v-for="option in aspectRatioOptions" :key="option.value" class="cursor-pointer border-r border-gray-200 last:border-r-0 dark:border-dark-700">
                  <input v-model="form.aspectRatio" class="peer sr-only" type="radio" :value="option.value" @change="applyAspectRatio(option.value)" />
                  <span class="flex h-16 flex-col items-center justify-center gap-1.5 text-xs text-gray-500 peer-checked:bg-gray-100 peer-checked:text-gray-950 dark:text-gray-400 dark:peer-checked:bg-dark-700 dark:peer-checked:text-white">
                    <i class="block rounded-[3px] border-2 border-current" :style="{ width: `${option.iconWidth}px`, height: `${option.iconHeight}px` }"></i>
                    <b class="font-medium">{{ option.label }}</b>
                  </span>
                </label>
                </div>
              </div>
            </fieldset>

            <div class="grid grid-cols-2 gap-3">
              <label class="block">
                <span class="input-label">质量</span>
                <select v-model="form.quality" class="input rounded-md">
                  <option value="auto">自动</option>
                  <option value="high">高</option>
                  <option value="medium">中</option>
                  <option value="low">低</option>
                </select>
              </label>
              <label class="block">
                <span class="input-label">格式</span>
                <select v-model="form.outputFormat" class="input rounded-md">
                  <option value="png">PNG</option>
                  <option value="jpeg">JPEG</option>
                  <option value="webp">WebP</option>
                </select>
              </label>
            </div>

            <fieldset>
              <legend class="input-label">生成数量</legend>
              <div class="grid grid-cols-4 overflow-hidden rounded-md border border-gray-300 dark:border-dark-600">
                <label v-for="count in MAX_BATCH_SIZE" :key="count" class="border-r border-gray-300 last:border-r-0 dark:border-dark-600" :class="count <= remainingQueueCapacity ? 'cursor-pointer' : 'cursor-not-allowed'">
                  <input v-model.number="form.count" class="peer sr-only" type="radio" :value="count" :disabled="count > remainingQueueCapacity" />
                  <span class="grid h-10 place-items-center text-sm text-gray-500 peer-checked:bg-gray-900 peer-checked:text-white peer-disabled:opacity-40 dark:text-gray-400 dark:peer-checked:bg-white dark:peer-checked:text-gray-950">{{ count }}</span>
                </label>
              </div>
            </fieldset>

            <button type="submit" class="flex h-12 w-full items-center justify-center gap-2 rounded-md bg-rose-600 px-4 text-sm font-bold text-white transition-colors hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-50" :disabled="!canSubmit">
              <Icon name="sparkles" size="md" />
              {{ remainingQueueCapacity > 0 ? '加入生成队列' : '队列已满' }}
            </button>
          </form>
        </section>

        <section class="min-w-0 p-5 sm:p-7">
          <div class="border-b border-gray-200 pb-6 dark:border-dark-700">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-center gap-3">
                <h2 class="text-base font-bold text-gray-950 dark:text-white">任务队列</h2>
                <span class="rounded-full border border-gray-300 px-2 py-0.5 text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ pendingCount }} / {{ MAX_QUEUE_SIZE }}</span>
              </div>
              <div class="grid w-full grid-cols-4 gap-2 sm:w-auto sm:min-w-72">
                <div v-for="slot in MAX_CONCURRENCY" :key="slot" class="flex h-10 items-center justify-center gap-2 rounded-md border text-xs" :class="slot <= activeCount ? 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300' : 'border-gray-200 text-gray-400 dark:border-dark-700'">
                  <i class="h-2 w-2 rounded-full" :class="slot <= activeCount ? 'animate-pulse bg-emerald-500' : 'bg-gray-300 dark:bg-dark-600'"></i>
                  {{ slot <= activeCount ? '生成中' : '空闲' }}
                </div>
              </div>
            </div>

            <div v-if="visibleJobs.length" class="mt-4 divide-y divide-gray-100 border-y border-gray-100 dark:divide-dark-700 dark:border-dark-700">
              <article v-for="job in visibleJobs" :key="job.id" class="grid min-h-14 grid-cols-[28px_minmax(0,1fr)_auto] items-center gap-3 py-2">
                <span class="flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold" :class="jobStatusClass(job.status)">
                  <Icon v-if="job.status === 'completed'" name="check" size="xs" />
                  <Icon v-else-if="job.status === 'failed'" name="exclamationCircle" size="xs" />
                  <span v-else-if="job.status === 'running'" class="h-3 w-3 animate-spin rounded-full border-2 border-emerald-200 border-t-emerald-600"></span>
                  <span v-else>{{ job.status === 'canceled' ? '×' : '…' }}</span>
                </span>
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium text-gray-800 dark:text-gray-100">{{ job.prompt }}</p>
                  <p class="truncate text-xs text-gray-400">{{ job.error || `${job.mode === 'edit' ? '图生图' : '文生图'} · ${job.aspectRatio === 'auto' ? '智能' : job.aspectRatio} · ${jobStatusLabel(job.status)}` }}</p>
                </div>
                <button v-if="job.status === 'queued' || job.status === 'running'" type="button" class="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-red-500 dark:hover:bg-dark-700" title="取消任务" @click="cancelJob(job)">
                  <Icon name="x" size="sm" />
                </button>
              </article>
            </div>
            <p v-else class="mt-4 py-4 text-center text-xs text-gray-400">队列空闲</p>
          </div>

          <div class="pt-6">
            <div class="mb-5 flex items-center justify-between gap-3">
              <div class="flex items-center gap-3">
                <h2 class="text-base font-bold text-gray-950 dark:text-white">生成图库</h2>
                <span class="rounded-full border border-gray-300 px-2 py-0.5 text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ images.length }}</span>
              </div>
              <button type="button" class="btn btn-secondary btn-icon rounded-md" title="刷新图库" @click="reloadImages">
                <Icon name="refresh" size="sm" />
              </button>
            </div>

            <div v-if="images.length" class="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
              <article v-for="image in images" :key="image.id" class="group overflow-hidden rounded-md border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
                <div class="relative aspect-square w-full overflow-hidden bg-gray-100 dark:bg-dark-950">
                  <button type="button" class="block h-full w-full" title="预览图片" @click="selectedImage = image">
                  <img :src="imageUrl(image)" :alt="image.prompt" class="h-full w-full object-cover transition-transform duration-200 group-hover:scale-[1.02]" loading="lazy" />
                  </button>
                  <span class="absolute right-2 top-2 flex gap-1 opacity-100 sm:opacity-0 sm:group-hover:opacity-100">
                    <button type="button" class="flex h-8 w-8 items-center justify-center rounded-md border border-white/50 bg-black/70 text-white" title="下载原图" @click.stop="downloadImage(image)"><Icon name="download" size="sm" /></button>
                    <button type="button" class="flex h-8 w-8 items-center justify-center rounded-md border border-white/50 bg-black/70 text-white hover:bg-red-600" title="删除图片" @click.stop="removeImage(image)"><Icon name="trash" size="sm" /></button>
                  </span>
                </div>
                <div class="p-3">
                  <p class="line-clamp-2 min-h-10 text-sm leading-5 text-gray-700 dark:text-gray-200">{{ image.prompt }}</p>
                  <div class="mt-2 flex items-center justify-between gap-2 text-xs text-gray-400">
                    <span>{{ image.mode === 'edit' ? '图生图' : '文生图' }} · {{ image.aspectRatio === 'auto' ? '智能' : (image.aspectRatio || image.size) }}</span>
                    <span class="text-emerald-600 dark:text-emerald-400">{{ expiryLabel(image.expiresAt) }}</span>
                  </div>
                </div>
              </article>
            </div>

            <div v-else class="grid min-h-[420px] items-center gap-8 lg:grid-cols-[minmax(260px,520px)_minmax(220px,320px)] lg:justify-center">
              <div class="relative aspect-[4/3] overflow-hidden border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-800">
                <img src="/assets/studio-sample.jpg" alt="AI 生图工作台示例" class="h-full w-full object-cover" />
                <span class="absolute bottom-4 left-4 bg-gray-950/80 px-2 py-1 text-xs font-medium text-white">STUDIO SAMPLE</span>
              </div>
              <div>
                <i class="mb-4 block h-1 w-12 bg-rose-600"></i>
                <h3 class="text-xl font-bold text-gray-950 dark:text-white">开始第一张创作</h3>
                <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">在左侧填写画面描述，任务会进入 4 路并发队列，结果保留 7 天，请尽快下载。</p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>

    <BaseDialog :show="!!selectedImage" :title="selectedImage?.model || '图片预览'" width="extra-wide" :z-index="60" @close="selectedImage = null">
      <div v-if="selectedImage" class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_280px]">
        <div class="grid min-h-80 place-items-center rounded-md bg-gray-950 p-3">
          <img :src="imageUrl(selectedImage)" :alt="selectedImage.prompt" class="max-h-[65vh] max-w-full object-contain" />
        </div>
        <div class="flex min-w-0 flex-col">
          <p class="whitespace-pre-wrap break-words text-sm leading-6 text-gray-700 dark:text-gray-200">{{ selectedImage.prompt }}</p>
          <dl class="mt-5 space-y-2 border-t border-gray-200 pt-4 text-sm dark:border-dark-700">
            <div class="flex justify-between gap-4"><dt class="text-gray-400">模式</dt><dd>{{ selectedImage.mode === 'edit' ? '图生图' : '文生图' }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-gray-400">比例</dt><dd>{{ selectedImage.aspectRatio === 'auto' ? '智能' : (selectedImage.aspectRatio || selectedImage.size) }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-gray-400">质量</dt><dd>{{ selectedImage.quality }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-gray-400">创建时间</dt><dd>{{ formatDate(selectedImage.createdAt) }}</dd></div>
          </dl>
          <div class="mt-auto grid grid-cols-[1fr_auto] gap-2 pt-6">
            <button type="button" class="btn btn-primary rounded-md" @click="downloadImage(selectedImage)"><Icon name="download" size="sm" />下载原图</button>
            <button type="button" class="btn btn-danger btn-icon rounded-md" title="删除图片" @click="removeImage(selectedImage)"><Icon name="trash" size="sm" /></button>
          </div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { keysAPI } from '@/api/keys'
import { userChannelsAPI, type UserPricedModel } from '@/api/channels'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'
import { getImageTask, submitImageTask, type GatewayImageResult, type GenerateImageOptions } from '@/features/ai/gateway'
import { compatibleApiKeys, selectCompatibleApiKey } from '@/features/ai/modelCatalog'
import { imageSizeForAspectRatio, isLikelyImageModel, promptWithAspectRatio, referenceImageLimitForModel, type ImageAspectRatio } from './capabilities'
import { createReactiveImageJob, MAX_BATCH_SIZE, MAX_CONCURRENCY, MAX_QUEUE_SIZE, recoverImageJob, remainingImageQueueCapacity, shouldPersistImageJobFailure } from './queue'
import { cleanupExpiredStudioImages, cleanupExpiredStudioJobs, deleteStudioImage, loadStudioImages, loadStudioJobs, saveStudioImage, saveStudioJob, STUDIO_RETENTION_MS } from './storage'
import type { ImageJob, ImageJobStatus, ImageMode, StudioImage } from './types'

interface ReferenceDraft {
  id: string
  file: File
  previewUrl: string
}

interface QueuePayload {
  job: ImageJob
  options: Omit<GenerateImageOptions, 'signal'>
}

const aspectRatioOptions = [
  { label: '智能', value: 'auto' as ImageAspectRatio, iconWidth: 17, iconHeight: 17 },
  { label: '21:9', value: '21:9' as ImageAspectRatio, iconWidth: 22, iconHeight: 8 },
  { label: '16:9', value: '16:9' as ImageAspectRatio, iconWidth: 22, iconHeight: 10 },
  { label: '3:2', value: '3:2' as ImageAspectRatio, iconWidth: 20, iconHeight: 13 },
  { label: '4:3', value: '4:3' as ImageAspectRatio, iconWidth: 18, iconHeight: 14 },
  { label: '1:1', value: '1:1' as ImageAspectRatio, iconWidth: 16, iconHeight: 16 },
  { label: '3:4', value: '3:4' as ImageAspectRatio, iconWidth: 14, iconHeight: 18 },
  { label: '2:3', value: '2:3' as ImageAspectRatio, iconWidth: 13, iconHeight: 20 },
  { label: '9:16', value: '9:16' as ImageAspectRatio, iconWidth: 10, iconHeight: 22 },
]

const appStore = useAppStore()
const authStore = useAuthStore()
const form = reactive({
  apiKeyId: 0,
  model: 'gpt-image-2',
  mode: 'text' as ImageMode,
  prompt: '',
  aspectRatio: 'auto' as ImageAspectRatio,
  quality: 'auto',
  outputFormat: 'png',
  count: 1,
})
const apiKeys = ref<ApiKey[]>([])
const models = ref<UserPricedModel[]>([])
const images = ref<StudioImage[]>([])
const jobs = ref<ImageJob[]>([])
const references = ref<ReferenceDraft[]>([])
const selectedImage = ref<StudioImage | null>(null)
const loadingKeys = ref(false)
const referenceInput = ref<HTMLInputElement | null>(null)
const queuePayloads = new Map<string, QueuePayload>()
const activeJobIds = new Set<string>()
const objectUrls = new Map<string, string>()
const ownerUserId = computed(() => {
  const id = authStore.user?.id
  return typeof id === 'number' && Number.isInteger(id) && id > 0 ? id : null
})

const isImageKey = (key: ApiKey) => {
  if (key.group?.allow_image_generation === false) return false
  const platform = key.group?.platform?.toLowerCase()
  return !platform || platform === 'openai' || platform === 'grok'
}
const compatibleImageKeys = computed(() => compatibleApiKeys(apiKeys.value, models.value, form.model, isImageKey))
const selectedApiKey = computed(() => selectCompatibleApiKey(apiKeys.value, models.value, form.model, form.apiKeyId, isImageKey))
const imageModels = computed(() => models.value.map((model) => model.name).filter(isLikelyImageModel))
const modelBindingMessage = computed(() => selectedApiKey.value
  ? ''
  : form.model.trim()
    ? `请先创建或绑定「${form.model}」所属分组的 API 密钥`
    : '请选择一个已定价生图模型')
const referenceLimit = computed(() => referenceImageLimitForModel(form.model))
const activeCount = computed(() => jobs.value.filter((job) => job.status === 'running').length)
const queuedCount = computed(() => jobs.value.filter((job) => job.status === 'queued').length)
const pendingCount = computed(() => activeCount.value + queuedCount.value)
const remainingQueueCapacity = computed(() => remainingImageQueueCapacity(pendingCount.value))
const visibleJobs = computed(() => jobs.value.slice(0, MAX_QUEUE_SIZE))
const canSubmit = computed(() => Boolean(
  ownerUserId.value && selectedApiKey.value && form.model.trim() && form.prompt.trim() &&
  form.count <= remainingQueueCapacity.value &&
  (form.mode === 'text' || (references.value.length > 0 && referenceLimit.value > 0)),
))

watch([() => form.model, apiKeys, models], () => {
  const nextID = selectedApiKey.value?.id || 0
  if (form.apiKeyId !== nextID) form.apiKeyId = nextID
  trimReferencesToLimit()
}, { deep: true })
watch(() => form.mode, (mode) => {
  if (mode === 'text') clearReferences()
})
watch(remainingQueueCapacity, (capacity) => {
  const maximumBatchSize = Math.min(MAX_BATCH_SIZE, capacity)
  if (maximumBatchSize > 0 && form.count > maximumBatchSize) form.count = maximumBatchSize
})

async function loadKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
    apiKeys.value = response.items || []
    await loadModels()
  } catch (error: any) {
    appStore.showError(error?.message || '读取 API 密钥失败')
  } finally {
    loadingKeys.value = false
  }
}

async function loadModels() {
  try {
    models.value = await userChannelsAPI.getPricedModels()
    const firstImageModel = models.value.find((model) => isLikelyImageModel(model.name))
    if (!imageModels.value.some((model) => model.toLowerCase() === form.model.trim().toLowerCase())) {
      form.model = firstImageModel?.name || ''
    }
  } catch (error: any) {
    models.value = []
    appStore.showError(error?.message || '读取已定价生图模型失败')
  }
}

function addReferences(event: Event) {
  const input = event.target as HTMLInputElement
  const files = [...(input.files || [])]
  input.value = ''
  let totalBytes = references.value.reduce((sum, reference) => sum + reference.file.size, 0)
  for (const file of files) {
    if (references.value.length >= referenceLimit.value) {
      appStore.showError(`当前模型最多支持 ${referenceLimit.value} 张参考图`)
      break
    }
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 20 * 1024 * 1024) {
      appStore.showError(`${file.name} 必须是 20 MB 以内的 PNG、JPEG 或 WebP 图片`)
      continue
    }
    if (totalBytes + file.size > 80 * 1024 * 1024) {
      appStore.showError('参考图片合计不能超过 80 MB')
      break
    }
    totalBytes += file.size
    references.value.push({ id: crypto.randomUUID(), file, previewUrl: URL.createObjectURL(file) })
  }
}

function removeReference(id: string) {
  const reference = references.value.find((item) => item.id === id)
  if (reference) URL.revokeObjectURL(reference.previewUrl)
  references.value = references.value.filter((item) => item.id !== id)
}

function clearReferences() {
  references.value.forEach((reference) => URL.revokeObjectURL(reference.previewUrl))
  references.value = []
}

function trimReferencesToLimit() {
  if (references.value.length <= referenceLimit.value) return
  references.value.slice(referenceLimit.value).forEach((reference) => URL.revokeObjectURL(reference.previewUrl))
  references.value = references.value.slice(0, referenceLimit.value)
  appStore.showInfo(`已按模型上限保留 ${referenceLimit.value} 张参考图`)
}

function applyAspectRatio(ratio: ImageAspectRatio) {
  form.prompt = promptWithAspectRatio(form.prompt, ratio)
}

async function submitJobs() {
  const owner = ownerUserId.value
  if (!owner || !canSubmit.value || !selectedApiKey.value) return
  const jobCount = Math.min(form.count, MAX_BATCH_SIZE, remainingQueueCapacity.value)
  if (jobCount === 0) return
  const apiKey = selectedApiKey.value
  const aspectRatio = form.aspectRatio
  const size = imageSizeForAspectRatio(form.aspectRatio)
  const options: Omit<GenerateImageOptions, 'signal'> = {
    apiKey: apiKey.key,
    model: form.model.trim(),
    prompt: form.prompt.trim(),
    mode: form.mode,
    size,
    quality: form.quality,
    outputFormat: form.outputFormat,
    references: form.mode === 'edit' ? references.value.map((reference) => reference.file) : undefined,
  }
  const newJobs: ImageJob[] = []
  for (let index = 0; index < jobCount; index += 1) {
    const job = createReactiveImageJob({
      id: crypto.randomUUID(),
      ownerUserId: owner,
      prompt: options.prompt,
      model: options.model,
      mode: options.mode,
      size: options.size,
      aspectRatio,
      apiKeyId: apiKey.id,
      quality: options.quality,
      outputFormat: options.outputFormat,
      references: options.references,
      status: 'queued',
      createdAt: Date.now() + index,
    })
    jobs.value.unshift(job)
    queuePayloads.set(job.id, { job, options })
    newJobs.push(job)
  }
  // Register every job in memory before awaiting IndexedDB writes. This keeps
  // rapid consecutive submissions from interleaving half-created batches.
  await Promise.all(newJobs.map((job) => saveStudioJob(owner, job)))
  appStore.showSuccess(`${jobCount} 个生图任务已加入队列`)
  pumpQueue()
}

function pumpQueue() {
  const owner = ownerUserId.value
  if (!owner) return
  while (activeJobIds.size < MAX_CONCURRENCY) {
    const payload = [...queuePayloads.values()].find((item) => (
      item.job.ownerUserId === owner &&
      !activeJobIds.has(item.job.id) &&
      (item.job.status === 'queued' || (item.job.status === 'running' && Boolean(item.job.remoteTaskId)))
    ))
    if (!payload) break
    void runJob(payload)
  }
}

function waitForImageTask(signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException('Aborted', 'AbortError'))
      return
    }
    const timeout = window.setTimeout(() => {
      signal.removeEventListener('abort', abort)
      resolve()
    }, 3000)
    const abort = () => {
      window.clearTimeout(timeout)
      reject(new DOMException('Aborted', 'AbortError'))
    }
    signal.addEventListener('abort', abort, { once: true })
  })
}

async function storeJobResults(job: ImageJob, options: QueuePayload['options'], results: GatewayImageResult[], signal: AbortSignal) {
  for (const [index, result] of results.entries()) {
    let blob = result.blob
    if (!blob && result.url) {
      try {
        const response = await fetch(result.url, { signal })
        if (response.ok) blob = await response.blob()
      } catch (error: any) {
        if (error?.name === 'AbortError') throw error
        // Keep the provider URL when it cannot be cached due to CORS.
      }
    }
    const image: StudioImage = {
      id: `${job.id}-${index}`,
      ownerUserId: job.ownerUserId,
      prompt: options.prompt,
      model: options.model,
      mode: options.mode,
      size: options.size,
      aspectRatio: job.aspectRatio,
      quality: options.quality,
      outputFormat: options.outputFormat,
      createdAt: Date.now(),
      expiresAt: Date.now() + STUDIO_RETENTION_MS,
      blob,
      remoteUrl: result.url,
    }
    await saveStudioImage(job.ownerUserId, image)
    if (ownerUserId.value === job.ownerUserId) {
      const existingIndex = images.value.findIndex((item) => item.id === image.id)
      if (existingIndex >= 0) images.value.splice(existingIndex, 1)
      images.value.unshift(image)
    }
  }
}

async function runJob(payload: QueuePayload) {
  const { job, options } = payload
  if (ownerUserId.value !== job.ownerUserId) {
    queuePayloads.delete(job.id)
    return
  }
  if (activeJobIds.has(job.id)) return
  activeJobIds.add(job.id)
  const controller = new AbortController()
  job.abortController = controller
  job.status = 'running'
  await saveStudioJob(job.ownerUserId, job)
  try {
    if (!job.remoteTaskId) {
      const submitted = await submitImageTask({ ...options, signal: controller.signal })
      job.remoteTaskId = submitted.taskId
      await saveStudioJob(job.ownerUserId, job)
    }

    const taskId = job.remoteTaskId
    if (!taskId) throw new Error('服务端未返回生图任务编号')
    while (!controller.signal.aborted && job.status === 'running') {
      const task = await getImageTask(options.apiKey, taskId, options.outputFormat, controller.signal)
      if (task.status === 'processing') {
        await waitForImageTask(controller.signal)
        continue
      }
      if (task.status === 'failed') throw new Error(task.error || '图片生成失败')
      await storeJobResults(job, options, task.results || [], controller.signal)
      if (controller.signal.aborted || job.status !== 'running') return
      job.status = 'completed'
      await saveStudioJob(job.ownerUserId, job)
      break
    }
  } catch (error: any) {
    // Refresh, navigation and logout only detach this browser's poller. The
    // server-side task keeps running and is resumed from remoteTaskId next time.
    if (shouldPersistImageJobFailure(error, job.status)) {
      job.status = 'failed'
      job.error = error?.message || '图片生成失败'
      appStore.showError(job.error || '图片生成失败')
      await saveStudioJob(job.ownerUserId, job)
    }
  } finally {
    job.abortController = undefined
    activeJobIds.delete(job.id)
    if (job.status !== 'running') queuePayloads.delete(job.id)
    if (ownerUserId.value === job.ownerUserId) pumpQueue()
  }
}

function cancelJob(job: ImageJob) {
  if (job.status === 'queued') {
    job.status = 'canceled'
    queuePayloads.delete(job.id)
    void saveStudioJob(job.ownerUserId, job)
  } else if (job.status === 'running') {
    job.status = 'canceled'
    queuePayloads.delete(job.id)
    void saveStudioJob(job.ownerUserId, job)
    job.abortController?.abort()
  }
}

function jobStatusLabel(status: ImageJobStatus): string {
  return ({ queued: '等待', running: '生成中', completed: '完成', failed: '失败', canceled: '已取消' })[status]
}

function jobStatusClass(status: ImageJobStatus): string {
  if (status === 'completed') return 'bg-emerald-600 text-white'
  if (status === 'failed') return 'bg-red-600 text-white'
  if (status === 'running') return 'border border-emerald-300 text-emerald-600 dark:border-emerald-800'
  return 'border border-gray-300 text-gray-400 dark:border-dark-600'
}

function imageUrl(image: StudioImage): string {
  if (image.blob) {
    const existing = objectUrls.get(image.id)
    if (existing) return existing
    const next = URL.createObjectURL(image.blob)
    objectUrls.set(image.id, next)
    return next
  }
  return image.remoteUrl || ''
}

async function reloadJobs() {
  const owner = ownerUserId.value
  if (!owner) {
    jobs.value = []
    queuePayloads.clear()
    return
  }
  await cleanupExpiredStudioJobs(owner)
  const storedJobs = await loadStudioJobs(owner)
  if (ownerUserId.value !== owner) return
  const resumedCount = storedJobs.filter((job) => job.status === 'running').length
  jobs.value = storedJobs.map(recoverImageJob)
  if (resumedCount) {
    appStore.showInfo(`${resumedCount} 个未完成任务已恢复，将继续生成`)
  }
  queuePayloads.clear()
  for (const job of jobs.value) {
    if (job.status !== 'queued' && !(job.status === 'running' && job.remoteTaskId)) {
      continue
    }
    const key = apiKeys.value.find((item) => item.id === job.apiKeyId && isImageKey(item))
    if (!key) {
      job.status = 'failed'
      job.error = '原 API 密钥已不可用，请重新选择模型和密钥'
      void saveStudioJob(owner, job)
      continue
    }
    queuePayloads.set(job.id, {
      job,
      options: {
        apiKey: key.key,
        model: job.model,
        prompt: job.prompt,
        mode: job.mode,
        size: job.size,
        quality: job.quality,
        outputFormat: job.outputFormat,
        references: job.references,
      },
    })
  }
  pumpQueue()
}

async function reloadImages() {
  const owner = ownerUserId.value
  if (!owner) {
    images.value = []
    return
  }
  await cleanupExpiredStudioImages(owner)
  const storedImages = await loadStudioImages(owner)
  if (ownerUserId.value === owner) images.value = storedImages
}

async function removeImage(image: StudioImage) {
  const owner = ownerUserId.value
  if (!owner || image.ownerUserId !== owner) return
  if (!window.confirm('确定从本地图库删除这张图片吗？')) return
  const deleted = await deleteStudioImage(owner, image.id)
  if (!deleted || ownerUserId.value !== owner) return
  images.value = images.value.filter((item) => item.id !== image.id)
  const url = objectUrls.get(image.id)
  if (url) URL.revokeObjectURL(url)
  objectUrls.delete(image.id)
  if (selectedImage.value?.id === image.id) selectedImage.value = null
  appStore.showSuccess('图片已删除')
}

function downloadImage(image: StudioImage) {
  const anchor = document.createElement('a')
  anchor.href = imageUrl(image)
  anchor.download = `sub2api-${image.id}.${image.outputFormat === 'jpeg' ? 'jpg' : image.outputFormat}`
  anchor.target = '_blank'
  anchor.rel = 'noopener'
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
}

function expiryLabel(expiresAt: number): string {
  const days = Math.max(0, Math.ceil((expiresAt - Date.now()) / (24 * 60 * 60 * 1000)))
  return `${days} 天后清理`
}

function formatDate(value: number): string {
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(value)
}

onMounted(async () => {
  await loadKeys()
  await Promise.all([reloadImages(), reloadJobs()])
})

watch(ownerUserId, async (nextOwner, previousOwner) => {
  if (nextOwner === previousOwner) return
  jobs.value.forEach((job) => job.abortController?.abort())
  activeJobIds.clear()
  queuePayloads.clear()
  jobs.value = []
  images.value = []
  selectedImage.value = null
  objectUrls.forEach((url) => URL.revokeObjectURL(url))
  objectUrls.clear()
  if (!nextOwner) return
  await loadKeys()
  if (ownerUserId.value !== nextOwner) return
  await Promise.all([reloadImages(), reloadJobs()])
})

onBeforeUnmount(() => {
  jobs.value.forEach((job) => job.abortController?.abort())
  activeJobIds.clear()
  queuePayloads.clear()
  clearReferences()
  objectUrls.forEach((url) => URL.revokeObjectURL(url))
  objectUrls.clear()
})
</script>
