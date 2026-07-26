<template>
  <AppLayout>
    <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-card dark:border-dark-700 dark:bg-dark-900">
      <header class="flex flex-col gap-4 border-b border-gray-200 px-5 py-4 sm:flex-row sm:items-center sm:justify-between dark:border-dark-700">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-md bg-rose-600 text-white">
            <Icon name="sparkles" size="lg" />
          </div>
          <div>
            <h1 class="text-lg font-bold text-gray-950 dark:text-white">AI 生图工作台</h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ selectedApiKey ? `${selectedApiKey.name} · ${form.model || '选择模型'}` : '等待可用 API 密钥' }}</p>
          </div>
        </div>
        <div class="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
          <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-emerald-500"></i>运行 {{ activeCount }}</span>
          <span class="flex items-center gap-1.5"><i class="h-2 w-2 rounded-full bg-amber-400"></i>等待 {{ queuedCount }}</span>
          <span>图库保留 15 天</span>
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
              <select v-model.number="form.apiKeyId" class="input rounded-md" :disabled="loadingKeys">
                <option :value="0">{{ loadingKeys ? '正在读取密钥...' : '请选择密钥' }}</option>
                <option v-for="key in imageApiKeys" :key="key.id" :value="key.id">{{ key.name || `API Key #${key.id}` }}</option>
              </select>
            </label>

            <label class="block">
              <span class="input-label">模型</span>
              <input v-model.trim="form.model" class="input rounded-md" list="image-studio-models" placeholder="gpt-image-2" @change="trimReferencesToLimit" />
              <datalist id="image-studio-models">
                <option v-for="model in imageModels" :key="model" :value="model" />
              </datalist>
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
              <legend class="input-label">画面尺寸</legend>
              <div class="grid grid-cols-3 overflow-hidden rounded-md border border-gray-300 dark:border-dark-600">
                <label v-for="option in sizeOptions" :key="option.value" class="cursor-pointer border-r border-gray-300 last:border-r-0 dark:border-dark-600">
                  <input v-model="form.size" class="peer sr-only" type="radio" :value="option.value" />
                  <span class="flex h-11 flex-col items-center justify-center text-xs text-gray-500 peer-checked:bg-primary-600 peer-checked:text-white dark:text-gray-400">
                    <b>{{ option.label }}</b><small>{{ option.value }}</small>
                  </span>
                </label>
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
                <label v-for="count in 4" :key="count" class="cursor-pointer border-r border-gray-300 last:border-r-0 dark:border-dark-600">
                  <input v-model.number="form.count" class="peer sr-only" type="radio" :value="count" />
                  <span class="grid h-10 place-items-center text-sm text-gray-500 peer-checked:bg-gray-900 peer-checked:text-white dark:text-gray-400 dark:peer-checked:bg-white dark:peer-checked:text-gray-950">{{ count }}</span>
                </label>
              </div>
            </fieldset>

            <button type="submit" class="flex h-12 w-full items-center justify-center gap-2 rounded-md bg-rose-600 px-4 text-sm font-bold text-white transition-colors hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-50" :disabled="!canSubmit">
              <Icon name="sparkles" size="md" />
              加入生成队列
            </button>
          </form>
        </section>

        <section class="min-w-0 p-5 sm:p-7">
          <div class="border-b border-gray-200 pb-6 dark:border-dark-700">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-center gap-3">
                <h2 class="text-base font-bold text-gray-950 dark:text-white">任务队列</h2>
                <span class="rounded-full border border-gray-300 px-2 py-0.5 text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">{{ pendingCount }}</span>
              </div>
              <div class="grid w-full grid-cols-4 gap-2 sm:w-auto sm:min-w-72">
                <div v-for="slot in 4" :key="slot" class="flex h-10 items-center justify-center gap-2 rounded-md border text-xs" :class="slot <= activeCount ? 'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300' : 'border-gray-200 text-gray-400 dark:border-dark-700'">
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
                  <p class="truncate text-xs text-gray-400">{{ job.error || `${job.mode === 'edit' ? '图生图' : '文生图'} · ${job.size} · ${jobStatusLabel(job.status)}` }}</p>
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
                    <span>{{ image.mode === 'edit' ? '图生图' : '文生图' }} · {{ image.size }}</span>
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
                <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">在左侧填写画面描述，任务会进入 4 路并发队列，结果自动保存在当前浏览器。</p>
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
          <p class="whitespace-pre-wrap break-words text-sm leading-6 text-gray-700 dark:text-gray-200">{{ selectedImage.revisedPrompt || selectedImage.prompt }}</p>
          <dl class="mt-5 space-y-2 border-t border-gray-200 pt-4 text-sm dark:border-dark-700">
            <div class="flex justify-between gap-4"><dt class="text-gray-400">模式</dt><dd>{{ selectedImage.mode === 'edit' ? '图生图' : '文生图' }}</dd></div>
            <div class="flex justify-between gap-4"><dt class="text-gray-400">尺寸</dt><dd>{{ selectedImage.size }}</dd></div>
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
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'
import { generateImage, listGatewayModels, type GenerateImageOptions, type GatewayModel } from '@/features/ai/gateway'
import { isLikelyImageModel, referenceImageLimitForModel } from './capabilities'
import { cleanupExpiredStudioImages, deleteStudioImage, loadStudioImages, saveStudioImage } from './storage'
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

const RETENTION_MS = 15 * 24 * 60 * 60 * 1000
const MAX_CONCURRENCY = 4
const DEFAULT_IMAGE_MODELS = ['gpt-image-2', 'gpt-image-1.5', 'gpt-image-1', 'gemini-3.1-flash-image', 'gemini-2.5-flash-image', 'grok-imagine-image', 'dall-e-3']
const sizeOptions = [
  { label: '方形', value: '1024x1024' },
  { label: '横向', value: '1536x1024' },
  { label: '竖向', value: '1024x1536' },
]

const appStore = useAppStore()
const form = reactive({
  apiKeyId: 0,
  model: 'gpt-image-2',
  mode: 'text' as ImageMode,
  prompt: '',
  size: '1024x1024',
  quality: 'auto',
  outputFormat: 'png',
  count: 1,
})
const apiKeys = ref<ApiKey[]>([])
const models = ref<GatewayModel[]>([])
const images = ref<StudioImage[]>([])
const jobs = ref<ImageJob[]>([])
const references = ref<ReferenceDraft[]>([])
const selectedImage = ref<StudioImage | null>(null)
const loadingKeys = ref(false)
const referenceInput = ref<HTMLInputElement | null>(null)
const queuePayloads = new Map<string, QueuePayload>()
const objectUrls = new Map<string, string>()

const imageApiKeys = computed(() => apiKeys.value.filter((key) => {
  if (key.group?.allow_image_generation === false) return false
  const platform = key.group?.platform?.toLowerCase()
  return !platform || platform === 'openai' || platform === 'grok'
}))
const selectedApiKey = computed(() => imageApiKeys.value.find((key) => key.id === Number(form.apiKeyId)) || null)
const imageModels = computed(() => {
  const discovered = models.value.map((model) => model.id).filter(isLikelyImageModel)
  return [...new Set([...discovered, ...DEFAULT_IMAGE_MODELS])]
})
const referenceLimit = computed(() => referenceImageLimitForModel(form.model))
const activeCount = computed(() => jobs.value.filter((job) => job.status === 'running').length)
const queuedCount = computed(() => jobs.value.filter((job) => job.status === 'queued').length)
const pendingCount = computed(() => activeCount.value + queuedCount.value)
const visibleJobs = computed(() => jobs.value.slice(0, 12))
const canSubmit = computed(() => Boolean(
  selectedApiKey.value && form.model.trim() && form.prompt.trim() &&
  (form.mode === 'text' || (references.value.length > 0 && referenceLimit.value > 0)),
))

watch(() => form.apiKeyId, () => void loadModels())
watch(() => form.mode, (mode) => {
  if (mode === 'text') clearReferences()
})

async function loadKeys() {
  loadingKeys.value = true
  try {
    const response = await keysAPI.list(1, 100, { status: 'active', sort_by: 'created_at', sort_order: 'desc' })
    apiKeys.value = response.items || []
    if (!selectedApiKey.value && imageApiKeys.value.length) form.apiKeyId = imageApiKeys.value[0].id
    await loadModels()
  } catch (error: any) {
    appStore.showError(error?.message || '读取 API 密钥失败')
  } finally {
    loadingKeys.value = false
  }
}

async function loadModels() {
  if (!selectedApiKey.value) {
    models.value = []
    return
  }
  try {
    models.value = await listGatewayModels(selectedApiKey.value.key)
    if (!form.model && imageModels.value.length) form.model = imageModels.value[0]
  } catch (error: any) {
    models.value = []
    appStore.showError(error?.message || '读取模型列表失败')
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

function submitJobs() {
  if (!canSubmit.value || !selectedApiKey.value) return
  const options: Omit<GenerateImageOptions, 'signal'> = {
    apiKey: selectedApiKey.value.key,
    model: form.model.trim(),
    prompt: form.prompt.trim(),
    mode: form.mode,
    size: form.size,
    quality: form.quality,
    outputFormat: form.outputFormat,
    references: form.mode === 'edit' ? references.value.map((reference) => reference.file) : undefined,
  }
  for (let index = 0; index < form.count; index += 1) {
    const job: ImageJob = {
      id: crypto.randomUUID(),
      prompt: options.prompt,
      model: options.model,
      mode: options.mode,
      size: options.size,
      status: 'queued',
      createdAt: Date.now() + index,
    }
    jobs.value.unshift(job)
    queuePayloads.set(job.id, { job, options })
  }
  appStore.showSuccess(`${form.count} 个生图任务已加入队列`)
  pumpQueue()
}

function pumpQueue() {
  while (activeCount.value < MAX_CONCURRENCY) {
    const payload = [...queuePayloads.values()].find((item) => item.job.status === 'queued')
    if (!payload) break
    void runJob(payload)
  }
}

async function runJob(payload: QueuePayload) {
  const { job, options } = payload
  const controller = new AbortController()
  job.abortController = controller
  job.status = 'running'
  try {
    const results = await generateImage({ ...options, signal: controller.signal })
    for (const result of results) {
      const image: StudioImage = {
        id: crypto.randomUUID(),
        prompt: options.prompt,
        revisedPrompt: result.revisedPrompt,
        model: options.model,
        mode: options.mode,
        size: options.size,
        quality: options.quality,
        outputFormat: options.outputFormat,
        createdAt: Date.now(),
        expiresAt: Date.now() + RETENTION_MS,
        blob: result.blob,
        remoteUrl: result.url,
      }
      images.value.unshift(image)
      await saveStudioImage(image)
    }
    job.status = 'completed'
  } catch (error: any) {
    if (error?.name === 'AbortError') job.status = 'canceled'
    else {
      job.status = 'failed'
      job.error = error?.message || '图片生成失败'
      appStore.showError(job.error || '图片生成失败')
    }
  } finally {
    job.abortController = undefined
    queuePayloads.delete(job.id)
    pumpQueue()
  }
}

function cancelJob(job: ImageJob) {
  if (job.status === 'queued') {
    job.status = 'canceled'
    queuePayloads.delete(job.id)
  } else if (job.status === 'running') {
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
  if (image.remoteUrl) return image.remoteUrl
  if (!image.blob) return ''
  const existing = objectUrls.get(image.id)
  if (existing) return existing
  const next = URL.createObjectURL(image.blob)
  objectUrls.set(image.id, next)
  return next
}

async function reloadImages() {
  await cleanupExpiredStudioImages()
  images.value = await loadStudioImages()
}

async function removeImage(image: StudioImage) {
  if (!window.confirm('确定从本地图库删除这张图片吗？')) return
  await deleteStudioImage(image.id)
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
  await Promise.all([loadKeys(), reloadImages()])
})

onBeforeUnmount(() => {
  jobs.value.forEach((job) => job.abortController?.abort())
  clearReferences()
  objectUrls.forEach((url) => URL.revokeObjectURL(url))
  objectUrls.clear()
})
</script>
