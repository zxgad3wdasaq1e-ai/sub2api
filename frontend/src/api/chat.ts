import { apiClient } from './client'
import { buildApiUrl } from './url'

export interface ChatAttachmentDTO {
  id: string
  name: string
  content_type: string
  byte_size: number
  sha256?: string
  preview_url?: string
  created_at: string
  expires_at: string
}
export interface ChatPartDTO {
  type: 'text' | 'image'
  text?: string
  attachment_id?: string
  name?: string
  content_type?: string
  byte_size?: number
  preview_url?: string
}
export interface ChatMessageDTO {
  id: string
  role: 'system' | 'user' | 'assistant'
  status: string
  parts: ChatPartDTO[]
  created_at: string
}
export interface ChatConversationDTO {
  id: string
  title: string
  model: string
  system_prompt?: string
  created_at: string
  updated_at: string
}
export interface ChatRunDTO {
  id: string
  conversation_id: string
  user_message_id: string
  assistant_message_id: string
  status: string
  model: string
  created_at: string
  error?: string
}
export interface ChatKeyOptionDTO { id: number; name: string; group_id: number | null; status: string }

export async function listChatKeyOptions(): Promise<ChatKeyOptionDTO[]> {
  const { data } = await apiClient.get<{ items: ChatKeyOptionDTO[] }>('/chat/keys')
  return data.items || []
}

export async function listChatConversations(): Promise<ChatConversationDTO[]> {
  const { data } = await apiClient.get<{ items: ChatConversationDTO[] }>('/chat/conversations')
  return data.items || []
}
export async function createChatConversation(input: { title?: string; model?: string; system_prompt?: string }): Promise<ChatConversationDTO> {
  const { data } = await apiClient.post<ChatConversationDTO>('/chat/conversations', input)
  return data
}
export async function getChatConversation(id: string): Promise<{ conversation: ChatConversationDTO; messages: ChatMessageDTO[] }> {
  const { data } = await apiClient.get<{ conversation: ChatConversationDTO; messages: ChatMessageDTO[] }>(`/chat/conversations/${encodeURIComponent(id)}`)
  return data
}
export async function deleteChatConversation(id: string): Promise<void> { await apiClient.delete(`/chat/conversations/${encodeURIComponent(id)}`) }

export async function uploadChatAttachment(file: File): Promise<ChatAttachmentDTO> {
  const form = new FormData()
  form.append('file', file, file.name)
  const { data } = await apiClient.post<ChatAttachmentDTO>('/chat/attachments', form, { headers: { 'Content-Type': 'multipart/form-data' } })
  return data
}
export async function getChatAttachmentContent(id: string, signal?: AbortSignal): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/chat/attachments/${encodeURIComponent(id)}/content`, { responseType: 'blob', signal })
  return data
}
export async function deleteChatAttachment(id: string): Promise<void> { await apiClient.delete(`/chat/attachments/${encodeURIComponent(id)}`) }

export interface StreamChatRunOptions {
  conversationId: string
  content: string
  model: string
  systemPrompt?: string
  apiKeyId: number | null
  temperature?: number
  attachments: Array<{ attachment_id: string }>
  signal: AbortSignal
  onDelta: (text: string) => void
}
export async function streamChatRun(options: StreamChatRunOptions): Promise<ChatRunDTO> {
  const token = localStorage.getItem('auth_token')
  const response = await fetch(buildApiUrl(`/chat/conversations/${encodeURIComponent(options.conversationId)}/runs`), {
    method: 'POST',
    headers: {
      Authorization: token ? `Bearer ${token}` : '',
      'Content-Type': 'application/json',
      Accept: 'text/event-stream, application/json',
    },
    body: JSON.stringify({
      content: options.content,
      model: options.model,
      system_prompt: options.systemPrompt,
      api_key_id: options.apiKeyId,
      temperature: options.temperature,
      attachments: options.attachments,
      idempotency_key: crypto.randomUUID(),
    }),
    signal: options.signal,
  })
  if (!response.ok) {
    let message = `Request failed (${response.status})`
    try {
      const payload = await response.json()
      message = payload?.message || payload?.error?.message || message
    } catch { /* keep status fallback */ }
    throw new Error(message)
  }
  const reader = response.body?.getReader()
  if (!reader) return { id: response.headers.get('X-Chat-Run-ID') || '', conversation_id: options.conversationId, user_message_id: '', assistant_message_id: '', status: 'completed', model: options.model, created_at: new Date().toISOString() }
  const decoder = new TextDecoder()
  let buffer = ''
  const consume = (line: string) => {
    if (!line.startsWith('data:')) return
    const data = line.slice(5).trim()
    if (!data || data === '[DONE]') return
    try {
      const payload = JSON.parse(data)
      const delta = payload?.choices?.[0]?.delta?.content
      if (typeof delta === 'string') options.onDelta(delta)
      if (payload?.type === 'response.output_text.delta' && typeof payload.delta === 'string') options.onDelta(payload.delta)
    } catch { /* provider keepalive */ }
  }
  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
    const lines = buffer.split(/\r?\n/)
    buffer = done ? '' : lines.pop() || ''
    lines.forEach(consume)
    if (done) break
  }
  if (buffer) consume(buffer)
  return { id: response.headers.get('X-Chat-Run-ID') || '', conversation_id: options.conversationId, user_message_id: response.headers.get('X-Chat-User-Message-ID') || '', assistant_message_id: response.headers.get('X-Chat-Assistant-Message-ID') || '', status: 'completed', model: options.model, created_at: new Date().toISOString() }
}

export const chatAPI = { keys: listChatKeyOptions, list: listChatConversations, create: createChatConversation, get: getChatConversation, remove: deleteChatConversation, uploadAttachment: uploadChatAttachment, getAttachmentContent: getChatAttachmentContent, removeAttachment: deleteChatAttachment, streamRun: streamChatRun }
