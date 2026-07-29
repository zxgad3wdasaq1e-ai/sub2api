export interface ChatAttachment {
  id: string
  name: string
  attachmentId: string
  contentType: string
  byteSize: number
  previewUrl: string
}

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  attachments?: ChatAttachment[]
  createdAt: number
  pending?: boolean
  error?: boolean
  status?: string
}

export interface Conversation {
  id: string
  title: string
  model: string
  messages: ChatMessage[]
  createdAt: number
  updatedAt: number
}

export interface ChatPreferences {
  apiKeyId: number | null
  defaultModel: string
  systemPrompt: string
  temperature: number
}
