import type { ChatPreferences, Conversation } from './types'

const CONVERSATIONS_KEY = 'sub2api_chat_conversations_v2'
const PREFERENCES_KEY = 'sub2api_chat_preferences_v1'

export const defaultChatPreferences: ChatPreferences = {
  apiKeyId: null,
  defaultModel: 'gpt-5.4',
  systemPrompt: '',
  temperature: 0.7,
}

function conversationCacheKey(userId?: number | string | null): string {
  return `${CONVERSATIONS_KEY}:${String(userId || 'anonymous')}`
}

export function loadConversations(userId?: number | string | null): Conversation[] {
  try {
    const parsed = JSON.parse(localStorage.getItem(conversationCacheKey(userId)) || '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.map((conversation) => ({
      ...conversation,
      messages: Array.isArray(conversation.messages) ? conversation.messages.map((message: any) => ({
        ...message,
        attachments: Array.isArray(message.attachments) ? message.attachments.filter((attachment: any) => !attachment.dataUrl) : undefined,
      })) : [],
    }))
  } catch {
    return []
  }
}

export function saveConversations(conversations: Conversation[], userId?: number | string | null): void {
  const recent = [...conversations]
    .sort((a, b) => b.updatedAt - a.updatedAt)
    .slice(0, 80)
  try {
    localStorage.setItem(conversationCacheKey(userId), JSON.stringify(recent))
  } catch {
    const textOnly = recent.map((conversation) => ({
      ...conversation,
      messages: conversation.messages.map((message) => ({ ...message, attachments: undefined })),
    }))
    try {
      localStorage.setItem(conversationCacheKey(userId), JSON.stringify(textOnly))
    } catch {
      // The in-memory conversation remains usable when browser storage is unavailable.
    }
  }
}

export function loadChatPreferences(): ChatPreferences {
  try {
    const saved = JSON.parse(localStorage.getItem(PREFERENCES_KEY) || '{}') as Partial<ChatPreferences>
    return { ...defaultChatPreferences, ...saved }
  } catch {
    return { ...defaultChatPreferences }
  }
}

export function saveChatPreferences(preferences: ChatPreferences): void {
  try {
    localStorage.setItem(PREFERENCES_KEY, JSON.stringify(preferences))
  } catch {
    // Settings continue to work for the current page session.
  }
}
