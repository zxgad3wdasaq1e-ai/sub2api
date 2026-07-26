import type { ChatPreferences, Conversation } from './types'

const CONVERSATIONS_KEY = 'sub2api_chat_conversations_v1'
const PREFERENCES_KEY = 'sub2api_chat_preferences_v1'

export const defaultChatPreferences: ChatPreferences = {
  apiKeyId: null,
  defaultModel: 'gpt-5.4',
  systemPrompt: '',
  temperature: 0.7,
}

export function loadConversations(): Conversation[] {
  try {
    const parsed = JSON.parse(localStorage.getItem(CONVERSATIONS_KEY) || '[]')
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function saveConversations(conversations: Conversation[]): void {
  const recent = [...conversations]
    .sort((a, b) => b.updatedAt - a.updatedAt)
    .slice(0, 80)
  try {
    localStorage.setItem(CONVERSATIONS_KEY, JSON.stringify(recent))
  } catch {
    const textOnly = recent.map((conversation) => ({
      ...conversation,
      messages: conversation.messages.map((message) => ({ ...message, attachments: undefined })),
    }))
    try {
      localStorage.setItem(CONVERSATIONS_KEY, JSON.stringify(textOnly))
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
