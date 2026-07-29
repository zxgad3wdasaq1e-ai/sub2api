<template>
  <AppLayout>
    <div class="relative flex h-[calc(100dvh-8rem)] min-h-[540px] overflow-hidden rounded-lg border border-gray-200 bg-white shadow-card dark:border-dark-700 dark:bg-dark-900">
      <button
        v-if="mobileOpen"
        type="button"
        class="absolute inset-0 z-20 bg-black/40 lg:hidden"
        aria-label="关闭会话列表"
        @click="mobileOpen = false"
      ></button>

      <aside
        class="absolute inset-y-0 left-0 z-30 flex w-72 flex-col border-r border-gray-200 bg-gray-50 transition-transform dark:border-dark-700 dark:bg-dark-950 lg:static lg:translate-x-0"
        :class="mobileOpen ? 'translate-x-0' : '-translate-x-full'"
      >
        <div class="flex h-16 items-center gap-2 border-b border-gray-200 px-3 dark:border-dark-700">
          <button type="button" class="btn btn-primary min-w-0 flex-1 rounded-md" @click="newConversation">
            <Icon name="plus" size="sm" />
            新对话
          </button>
          <button type="button" class="btn btn-secondary btn-icon rounded-md" title="聊天设置" @click="settingsOpen = true">
            <Icon name="cog" size="sm" />
          </button>
        </div>

        <div class="p-3">
          <label class="relative block">
            <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-2.5 text-gray-400" />
            <input v-model="conversationQuery" class="input rounded-md py-2 pl-9" type="search" placeholder="搜索对话" />
          </label>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
          <div v-if="!filteredConversations.length" class="px-3 py-8 text-center text-xs text-gray-500 dark:text-gray-400">
            暂无历史对话
          </div>
          <div v-for="group in conversationGroups" :key="group.label" class="mb-4">
            <p class="px-2 pb-1 text-xs font-medium text-gray-400">{{ group.label }}</p>
            <div
              v-for="conversation in group.items"
              :key="conversation.id"
              class="group mb-1 flex items-center rounded-md"
              :class="conversation.id === activeId ? 'bg-primary-50 text-primary-800 dark:bg-primary-500/10 dark:text-primary-200' : 'hover:bg-gray-100 dark:hover:bg-dark-800'"
            >
              <button type="button" class="flex min-w-0 flex-1 items-center gap-2 px-3 py-2.5 text-left" @click="selectConversation(conversation.id)">
                <Icon name="chat" size="sm" class="flex-shrink-0" />
                <span class="truncate text-sm">{{ conversation.title }}</span>
              </button>
              <button type="button" class="mr-1 rounded p-1.5 text-gray-400 opacity-0 hover:bg-white hover:text-red-500 group-hover:opacity-100 dark:hover:bg-dark-700" title="删除对话" @click="removeConversation(conversation.id)">
                <Icon name="trash" size="xs" />
              </button>
            </div>
          </div>
        </div>

        <div class="border-t border-gray-200 p-3 dark:border-dark-700">
          <label class="mb-1 block text-xs text-gray-500 dark:text-gray-400">API 密钥</label>
          <select v-model.number="preferences.apiKeyId" class="input rounded-md py-2" :disabled="loadingKeys || !compatibleKeys.length">
            <option :value="null">{{ loadingKeys ? '正在读取密钥...' : '自动选择可用密钥' }}</option>
            <option v-for="key in compatibleKeys" :key="key.id" :value="key.id">{{ key.name || `API Key #${key.id}` }}</option>
          </select>
          <p v-if="modelBindingMessage" class="mt-2 text-xs text-amber-600 dark:text-amber-400">{{ modelBindingMessage }}</p>
        </div>
      </aside>

      <section class="flex min-w-0 flex-1 flex-col">
        <header class="flex h-16 flex-shrink-0 items-center justify-between gap-3 border-b border-gray-200 px-3 sm:px-5 dark:border-dark-700">
          <div class="flex min-w-0 items-center gap-2">
            <button type="button" class="btn btn-ghost btn-icon rounded-md lg:hidden" title="打开会话列表" @click="mobileOpen = true">
              <Icon name="menu" size="md" />
            </button>
            <div class="min-w-0">
              <p class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ currentConversation?.title || 'AI 对话' }}</p>
              <p class="text-xs" :class="selectedApiKey ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                {{ selectedApiKey ? `已连接 ${selectedApiKey.name}` : modelBindingMessage }}
              </p>
            </div>
          </div>
          <div class="w-44 sm:w-64">
            <select v-model="currentModel" class="input rounded-md py-2" aria-label="聊天模型" :disabled="!models.length">
              <option v-if="!models.length" value="">暂无已定价模型</option>
              <option v-for="model in models" :key="model.name" :value="model.name">{{ model.name }}</option>
            </select>
          </div>
        </header>

        <div ref="messagesElement" class="min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-8">
          <div v-if="!currentConversation?.messages.length" class="mx-auto flex h-full max-w-2xl flex-col items-center justify-center text-center">
            <div class="mb-5 flex h-14 w-14 items-center justify-center rounded-lg bg-primary-600 text-white shadow-glow">
              <Icon name="chat" size="xl" />
            </div>
            <h1 class="text-2xl font-bold text-gray-950 dark:text-white">AI 对话</h1>
            <p class="mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">选择模型后开始对话，也可以附加图片进行多模态分析。</p>
            <div class="mt-7 grid w-full max-w-xl grid-cols-1 gap-2 sm:grid-cols-3">
              <button v-for="prompt in starterPrompts" :key="prompt" type="button" class="rounded-md border border-gray-200 px-3 py-3 text-left text-sm text-gray-600 transition-colors hover:border-primary-400 hover:bg-primary-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-primary-500/10" @click="draft = prompt">
                {{ prompt }}
              </button>
            </div>
          </div>

          <div v-else class="mx-auto max-w-3xl space-y-7">
            <article v-for="message in currentConversation.messages" :key="message.id" class="group flex gap-3" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
              <div v-if="message.role === 'assistant'" class="mt-0.5 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-md bg-primary-600 text-white">
                <Icon name="sparkles" size="sm" />
              </div>
              <div class="min-w-0" :class="message.role === 'user' ? 'max-w-[85%]' : 'max-w-[calc(100%-2.75rem)] flex-1'">
                <div v-if="message.attachments?.length" class="mb-2 flex flex-wrap justify-end gap-2">
                  <img v-for="attachment in message.attachments" v-show="attachment.previewUrl" :key="attachment.id" :src="attachment.previewUrl" :alt="attachment.name" class="h-28 w-28 rounded-md border border-gray-200 object-cover dark:border-dark-700" />
                </div>
                <div
                  v-if="message.role === 'user'"
                  class="whitespace-pre-wrap rounded-lg rounded-br-sm bg-primary-600 px-4 py-3 text-sm leading-6 text-white"
                >{{ message.content }}</div>
                <div v-else class="relative rounded-lg rounded-tl-sm border px-4 py-3" :class="message.error ? 'border-red-200 bg-red-50 dark:border-red-900/60 dark:bg-red-950/20' : 'border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800/70'">
                  <div v-if="message.content" class="chat-markdown text-sm leading-7 text-gray-800 dark:text-gray-100" v-html="renderMarkdown(message.content)"></div>
                  <div v-else-if="message.pending" class="flex items-center gap-2 py-1 text-sm text-gray-500">
                    <span class="h-2 w-2 animate-pulse rounded-full bg-primary-500"></span>
                    正在生成
                  </div>
                  <button v-if="message.content" type="button" class="absolute -bottom-8 left-0 rounded p-1.5 text-gray-400 opacity-0 hover:bg-gray-100 hover:text-gray-700 group-hover:opacity-100 dark:hover:bg-dark-700 dark:hover:text-gray-200" title="复制回复" @click="copyMessage(message.content)">
                    <Icon name="copy" size="xs" />
                  </button>
                </div>
              </div>
            </article>
          </div>
        </div>

        <div class="flex-shrink-0 border-t border-gray-200 bg-white p-3 sm:px-6 sm:py-4 dark:border-dark-700 dark:bg-dark-900">
          <div class="mx-auto max-w-3xl">
            <div v-if="attachments.length" class="mb-2 flex gap-2 overflow-x-auto pb-1">
              <div v-for="attachment in attachments" :key="attachment.id" class="relative flex-shrink-0">
                <img :src="attachment.previewUrl" :alt="attachment.name" class="h-16 w-16 rounded-md border border-gray-200 object-cover dark:border-dark-700" />
                <button type="button" class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-gray-900 text-white" :title="`移除 ${attachment.name}`" @click="removeAttachment(attachment.id)">
                  <Icon name="x" size="xs" />
                </button>
              </div>
            </div>
            <div class="flex items-end gap-2 rounded-lg border border-gray-300 bg-white p-2 focus-within:border-primary-500 focus-within:ring-2 focus-within:ring-primary-500/20 dark:border-dark-600 dark:bg-dark-800">
              <input ref="fileInput" class="hidden" type="file" accept="image/png,image/jpeg,image/webp" multiple @change="handleFiles" />
              <button type="button" class="btn btn-ghost btn-icon flex-shrink-0 rounded-md" title="添加图片" :disabled="busy" @click="fileInput?.click()">
                <Icon name="upload" size="md" />
              </button>
              <textarea v-model="draft" rows="1" class="max-h-40 min-h-[40px] min-w-0 flex-1 resize-none border-0 bg-transparent px-2 py-2 text-sm leading-6 text-gray-900 outline-none placeholder:text-gray-400 dark:text-white" placeholder="输入消息" @keydown.enter.exact.prevent="sendMessage" @keydown.enter.shift.exact.stop></textarea>
              <button v-if="busy" type="button" class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md bg-gray-900 text-white dark:bg-white dark:text-gray-950" title="停止生成" @click="stopGeneration">
                <span class="h-3 w-3 rounded-sm bg-current"></span>
              </button>
              <button v-else type="button" class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-md bg-primary-600 text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-40" title="发送" :disabled="!canSend" @click="sendMessage">
                <Icon name="arrowUp" size="md" />
              </button>
            </div>
          </div>
        </div>
      </section>
    </div>

    <BaseDialog :show="settingsOpen" title="聊天设置" width="normal" @close="settingsOpen = false">
      <div class="space-y-5">
        <label class="block">
          <span class="input-label">默认模型</span>
          <select v-model="preferences.defaultModel" class="input" :disabled="!models.length">
            <option v-for="model in models" :key="model.name" :value="model.name">{{ model.name }}</option>
          </select>
        </label>
        <label class="block">
          <span class="input-label">系统提示词</span>
          <textarea v-model="preferences.systemPrompt" class="input min-h-28 resize-y" placeholder="可选，为每次请求添加系统指令"></textarea>
        </label>
        <label class="block">
          <span class="input-label flex items-center justify-between"><span>温度</span><span>{{ preferences.temperature.toFixed(1) }}</span></span>
          <input v-model.number="preferences.temperature" class="w-full accent-primary-600" type="range" min="0" max="2" step="0.1" />
        </label>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="settingsOpen = false">取消</button>
        <button type="button" class="btn btn-primary" @click="saveSettings">保存设置</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { userChannelsAPI, type UserPricedModel } from '@/api/channels'
import { chatAPI, type ChatMessageDTO } from '@/api/chat'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { ApiKey } from '@/types'
import { compatibleApiKeys, findPricedModel, selectCompatibleApiKey } from '@/features/ai/modelCatalog'
import { loadChatPreferences, loadConversations, saveChatPreferences, saveConversations } from './storage'
import type { ChatAttachment, ChatMessage, ChatPreferences, Conversation } from './types'

const appStore = useAppStore()
const authStore = useAuthStore()
const preferences = ref<ChatPreferences>(loadChatPreferences())
const conversations = ref<Conversation[]>(loadConversations(authStore.user?.id))
const activeId = ref('')
const apiKeys = ref<ApiKey[]>([])
const models = ref<UserPricedModel[]>([])
const loadingKeys = ref(false)
const busy = ref(false)
const mobileOpen = ref(false)
const settingsOpen = ref(false)
const conversationQuery = ref('')
const draft = ref('')
const attachments = ref<ChatAttachment[]>([])
const serverReady = ref(false)
const uploadingAttachment = ref(false)
const messagesElement = ref<HTMLElement | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
let abortController: AbortController | null = null
let saveTimer: ReturnType<typeof setTimeout> | null = null
const attachmentObjectUrls = new Set<string>()

const starterPrompts = ['总结这段内容的关键结论', '帮我设计一个可执行的方案', '分析图片中的信息']

function createConversation(): Conversation {
  const now = Date.now()
  return {
    id: crypto.randomUUID(),
    title: '新对话',
    model: preferences.value.defaultModel,
    messages: [],
    createdAt: now,
    updatedAt: now,
  }
}

function ensureActiveConversation() {
  const existing = conversations.value.find((item) => item.id === activeId.value)
  if (existing) return
  const latest = [...conversations.value].sort((a, b) => b.updatedAt - a.updatedAt)[0]
  if (latest) activeId.value = latest.id
  else {
    const conversation = createConversation()
    conversations.value.push(conversation)
    activeId.value = conversation.id
  }
}

ensureActiveConversation()

const currentConversation = computed(() => conversations.value.find((item) => item.id === activeId.value))
const currentModel = computed({
  get: () => currentConversation.value?.model || preferences.value.defaultModel,
  set: (value: string) => {
    if (currentConversation.value) currentConversation.value.model = value
  },
})
const selectedApiKey = computed(() => {
  return selectCompatibleApiKey(apiKeys.value, models.value, currentModel.value, preferences.value.apiKeyId)
})
const compatibleKeys = computed(() => compatibleApiKeys(apiKeys.value, models.value, currentModel.value))
const modelBindingMessage = computed(() => selectedApiKey.value
  ? ''
  : currentModel.value
    ? `请先创建或绑定「${currentModel.value}」所属分组的 API 密钥`
    : '请选择一个已定价模型')
const filteredConversations = computed(() => {
  const query = conversationQuery.value.trim().toLowerCase()
  return [...conversations.value]
    .filter((conversation) => conversation.messages.length > 0)
    .filter((conversation) => !query || conversation.title.toLowerCase().includes(query))
    .sort((a, b) => b.updatedAt - a.updatedAt)
})
const conversationGroups = computed(() => {
  const day = 24 * 60 * 60 * 1000
  const groups = [
    { label: '今天', items: [] as Conversation[] },
    { label: '最近 7 天', items: [] as Conversation[] },
    { label: '更早', items: [] as Conversation[] },
  ]
  for (const conversation of filteredConversations.value) {
    const age = Date.now() - conversation.updatedAt
    if (age < day) groups[0].items.push(conversation)
    else if (age < day * 7) groups[1].items.push(conversation)
    else groups[2].items.push(conversation)
  }
  return groups.filter((group) => group.items.length)
})
const canSend = computed(() => Boolean(
  serverReady.value && selectedApiKey.value && currentModel.value.trim() && !uploadingAttachment.value && (draft.value.trim() || attachments.value.length),
))

watch(conversations, () => {
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = setTimeout(() => saveConversations(conversations.value, authStore.user?.id), 200)
}, { deep: true })

watch(() => preferences.value.apiKeyId, () => {
  saveChatPreferences(preferences.value)
})

watch([currentModel, apiKeys, models], () => {
  const nextID = selectedApiKey.value?.id ?? null
  if (preferences.value.apiKeyId !== nextID) preferences.value.apiKeyId = nextID
}, { deep: true })

function renderMarkdown(content: string): string {
  return DOMPurify.sanitize(marked.parse(content, { breaks: true }) as string)
}

async function scrollToBottom() {
  await nextTick()
  if (messagesElement.value) messagesElement.value.scrollTop = messagesElement.value.scrollHeight
}

function newConversation() {
  if (busy.value) stopGeneration()
  void (async () => {
    try {
      const created = await chatAPI.create({ model: preferences.value.defaultModel, system_prompt: preferences.value.systemPrompt })
      conversations.value.unshift({ id: created.id, title: created.title, model: created.model || preferences.value.defaultModel, messages: [], createdAt: Date.parse(created.created_at), updatedAt: Date.parse(created.updated_at) })
      activeId.value = created.id
      mobileOpen.value = false
      void scrollToBottom()
    } catch (error: any) {
      appStore.showError(error?.message || '创建会话失败')
    }
  })()
}

function selectConversation(id: string) {
  if (busy.value) stopGeneration()
  activeId.value = id
  mobileOpen.value = false
  void refreshConversation(id)
  void scrollToBottom()
}

function removeConversation(id: string) {
  if (!window.confirm('确定删除这个对话吗？')) return
  if (busy.value && id === activeId.value) stopGeneration()
  void (async () => {
    try {
      await chatAPI.remove(id)
      conversations.value = conversations.value.filter((conversation) => conversation.id !== id)
      ensureActiveConversation()
    } catch (error: any) {
      appStore.showError(error?.message || '删除会话失败')
    }
  })()
}

async function loadKeys() {
  loadingKeys.value = true
  try {
    const options = await chatAPI.keys()
    apiKeys.value = options.map((item) => ({ id: item.id, name: item.name, group_id: item.group_id, status: item.status, key: '' } as ApiKey))
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
    const fallbackModel = models.value[0]?.name || ''
    if (!findPricedModel(models.value, preferences.value.defaultModel)) preferences.value.defaultModel = fallbackModel
    if (!findPricedModel(models.value, currentModel.value)) currentModel.value = fallbackModel
    saveChatPreferences(preferences.value)
  } catch (error: any) {
    models.value = []
    appStore.showError(error?.message || '读取已定价模型失败')
  }
}

function messageFromDTO(message: ChatMessageDTO): ChatMessage {
  const text = (message.parts || []).filter((part) => part.type === 'text').map((part) => part.text || '').join('')
  const attachments = (message.parts || []).filter((part) => part.type === 'image' && part.attachment_id).map((part) => ({
    id: part.attachment_id as string,
    attachmentId: part.attachment_id as string,
    name: part.name || 'image',
    contentType: part.content_type || 'image/*',
    byteSize: part.byte_size || 0,
    previewUrl: '',
  }))
  return { id: message.id, role: message.role === 'system' ? 'assistant' : message.role, content: text, attachments: attachments.length ? attachments : undefined, createdAt: Date.parse(message.created_at), pending: message.status === 'pending', error: message.status === 'failed' || message.status === 'cancelled', status: message.status }
}

async function refreshConversation(id: string) {
  try {
    const payload = await chatAPI.get(id)
    const conversation = conversations.value.find((item) => item.id === id)
    if (!conversation) return
    conversation.title = payload.conversation.title
    conversation.model = payload.conversation.model || conversation.model
    conversation.createdAt = Date.parse(payload.conversation.created_at)
    conversation.updatedAt = Date.parse(payload.conversation.updated_at)
    conversation.messages.flatMap((message) => message.attachments || []).forEach((attachment) => {
      if (attachment.previewUrl) {
        URL.revokeObjectURL(attachment.previewUrl)
        attachmentObjectUrls.delete(attachment.previewUrl)
      }
    })
    conversation.messages = payload.messages.filter((message) => message.role !== 'system').map(messageFromDTO)
    await hydrateAttachmentPreviews(conversation.messages)
  } catch (error: any) {
    appStore.showError(error?.message || '读取会话失败')
  }
}

async function hydrateAttachmentPreviews(messages: ChatMessage[]) {
  const attachments = messages.flatMap((message) => message.attachments || []).filter((attachment) => !attachment.previewUrl)
  await Promise.all(attachments.map(async (attachment) => {
    try {
      const url = URL.createObjectURL(await chatAPI.getAttachmentContent(attachment.attachmentId))
      attachmentObjectUrls.add(url)
      attachment.previewUrl = url
    } catch { /* message metadata remains available when preview loading fails */ }
  }))
}

async function loadServerConversations() {
  try {
    const items = await chatAPI.list()
    conversations.value = items.map((item) => ({ id: item.id, title: item.title, model: item.model || preferences.value.defaultModel, messages: [], createdAt: Date.parse(item.created_at), updatedAt: Date.parse(item.updated_at) }))
    if (!conversations.value.length) {
      const created = await chatAPI.create({ model: preferences.value.defaultModel, system_prompt: preferences.value.systemPrompt })
      conversations.value = [{ id: created.id, title: created.title, model: created.model || preferences.value.defaultModel, messages: [], createdAt: Date.parse(created.created_at), updatedAt: Date.parse(created.updated_at) }]
    }
    activeId.value = conversations.value[0].id
    await refreshConversation(activeId.value)
    serverReady.value = true
  } catch (error: any) {
    serverReady.value = false
    appStore.showError(error?.message || '读取服务端会话失败')
  }
}

async function handleFiles(event: Event) {
  const input = event.target as HTMLInputElement
  const files = [...(input.files || [])]
  input.value = ''
  uploadingAttachment.value = true
  for (const file of files) {
    if (attachments.value.length >= 4) {
      appStore.showError('每条消息最多添加 4 张图片')
      break
    }
    if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 10 * 1024 * 1024) {
      appStore.showError(`${file.name} 必须是 10 MB 以内的 PNG、JPEG 或 WebP 图片`)
      continue
    }
    try {
      const uploaded = await chatAPI.uploadAttachment(file)
      const previewUrl = URL.createObjectURL(file)
      attachmentObjectUrls.add(previewUrl)
      attachments.value.push({ id: uploaded.id, attachmentId: uploaded.id, name: uploaded.name, contentType: uploaded.content_type, byteSize: uploaded.byte_size, previewUrl })
    } catch (error: any) {
      appStore.showError(error?.message || `${file.name} 上传失败`)
    }
  }
  uploadingAttachment.value = false
}

function removeAttachment(id: string) {
  const removed = attachments.value.find((attachment) => attachment.id === id)
  attachments.value = attachments.value.filter((attachment) => attachment.id !== id)
  if (removed?.previewUrl) {
    URL.revokeObjectURL(removed.previewUrl)
    attachmentObjectUrls.delete(removed.previewUrl)
  }
  void chatAPI.removeAttachment?.(id)
}

async function sendMessage() {
  if (busy.value || !canSend.value || !currentConversation.value || !selectedApiKey.value) return
  const conversation = currentConversation.value
  const text = draft.value.trim()
  const sentAttachments = [...attachments.value]
  const userMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'user',
    content: text,
    attachments: sentAttachments.length ? sentAttachments : undefined,
    createdAt: Date.now(),
  }
  const assistantMessage: ChatMessage = {
    id: crypto.randomUUID(),
    role: 'assistant',
    content: '',
    createdAt: Date.now(),
    pending: true,
  }
  conversation.messages.push(userMessage, assistantMessage)
  conversation.updatedAt = Date.now()
  if (conversation.title === '新对话') conversation.title = (text || sentAttachments[0]?.name || '图片分析').slice(0, 42)
  draft.value = ''
  attachments.value = []
  busy.value = true
  abortController = new AbortController()
  void scrollToBottom()

  try {
    await chatAPI.streamRun({
      conversationId: conversation.id,
      content: text,
      model: currentModel.value,
      systemPrompt: preferences.value.systemPrompt,
      apiKeyId: selectedApiKey.value.id,
      temperature: preferences.value.temperature,
      attachments: sentAttachments.map((attachment) => ({ attachment_id: attachment.attachmentId })),
      signal: abortController.signal,
      onDelta: (delta) => {
        assistantMessage.content += delta
        void scrollToBottom()
      },
    })
    if (!assistantMessage.content) throw new Error('模型未返回文本内容')
    await refreshConversation(conversation.id)
  } catch (error: any) {
    if (error?.name !== 'AbortError') {
      assistantMessage.error = true
      assistantMessage.content ||= error?.message || '请求失败'
    }
  } finally {
    assistantMessage.pending = false
    conversation.updatedAt = Date.now()
    busy.value = false
    abortController = null
    void scrollToBottom()
  }
}

function stopGeneration() {
  abortController?.abort()
}

async function copyMessage(content: string) {
  try {
    await navigator.clipboard.writeText(content)
    appStore.showSuccess('已复制回复')
  } catch {
    appStore.showError('复制失败')
  }
}

function saveSettings() {
  saveChatPreferences(preferences.value)
  settingsOpen.value = false
  appStore.showSuccess('聊天设置已保存')
}

onMounted(() => {
  void loadKeys()
  void loadServerConversations()
  void scrollToBottom()
})

onBeforeUnmount(() => {
  abortController?.abort()
  if (saveTimer) clearTimeout(saveTimer)
  saveConversations(conversations.value, authStore.user?.id)
  attachmentObjectUrls.forEach((url) => URL.revokeObjectURL(url))
  attachmentObjectUrls.clear()
})
</script>

<style scoped>
.chat-markdown :deep(p) {
  margin: 0 0 0.75rem;
}

.chat-markdown :deep(p:last-child) {
  margin-bottom: 0;
}

.chat-markdown :deep(pre) {
  margin: 0.75rem 0;
  overflow-x: auto;
  border-radius: 0.375rem;
  background: #111827;
  padding: 0.875rem;
  color: #f3f4f6;
}

.chat-markdown :deep(code:not(pre code)) {
  border-radius: 0.25rem;
  background: rgb(148 163 184 / 18%);
  padding: 0.125rem 0.3rem;
}

.chat-markdown :deep(ul),
.chat-markdown :deep(ol) {
  margin: 0.6rem 0;
  padding-left: 1.4rem;
}

.chat-markdown :deep(ul) {
  list-style: disc;
}

.chat-markdown :deep(ol) {
  list-style: decimal;
}

.chat-markdown :deep(a) {
  color: #0d9488;
  text-decoration: underline;
}

.chat-markdown :deep(blockquote) {
  margin: 0.75rem 0;
  border-left: 3px solid #14b8a6;
  padding-left: 0.875rem;
  color: #64748b;
}
</style>
