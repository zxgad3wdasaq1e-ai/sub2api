import { buildGatewayUrl } from '@/api/client'

export interface GatewayModel {
  id: string
  owned_by?: string
}

export interface ChatAttachmentInput {
  dataUrl: string
}

export interface ChatCompletionMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
  attachments?: ChatAttachmentInput[]
}

export interface StreamChatOptions {
  apiKey: string
  model: string
  messages: ChatCompletionMessage[]
  temperature?: number
  signal: AbortSignal
  onDelta: (text: string) => void
}

export interface GenerateImageOptions {
  apiKey: string
  model: string
  prompt: string
  mode: 'text' | 'edit'
  size: string
  quality: string
  outputFormat: string
  references?: File[]
  signal?: AbortSignal
}

export interface GatewayImageResult {
  blob?: Blob
  url?: string
}

function extractItems(payload: unknown): unknown[] {
  if (Array.isArray(payload)) return payload
  if (!payload || typeof payload !== 'object') return []
  const root = payload as Record<string, unknown>
  if (Array.isArray(root.data)) return root.data
  if (Array.isArray(root.items)) return root.items
  if (Array.isArray(root.output)) return root.output
  if (root.data && typeof root.data === 'object') return extractItems(root.data)
  if (root.result && typeof root.result === 'object') return extractItems(root.result)
  return []
}

function errorMessage(payload: unknown, fallback: string): string {
  if (!payload || typeof payload !== 'object') return fallback
  const root = payload as Record<string, unknown>
  if (typeof root.message === 'string') return root.message
  if (typeof root.detail === 'string') return root.detail
  if (root.error && typeof root.error === 'object') {
    const error = root.error as Record<string, unknown>
    if (typeof error.message === 'string') return error.message
  }
  return fallback
}

export async function parseGatewayError(response: Response): Promise<Error> {
  const fallback = `Request failed (${response.status})`
  try {
    return new Error(errorMessage(await response.json(), fallback))
  } catch {
    return new Error(fallback)
  }
}

export async function listGatewayModels(apiKey: string, signal?: AbortSignal): Promise<GatewayModel[]> {
  const response = await fetch(buildGatewayUrl('/v1/models'), {
    headers: { Authorization: `Bearer ${apiKey}`, Accept: 'application/json' },
    signal,
  })
  if (!response.ok) throw await parseGatewayError(response)
  return extractItems(await response.json())
    .filter((item): item is GatewayModel => Boolean(
      item && typeof item === 'object' && typeof (item as GatewayModel).id === 'string',
    ))
    .sort((a, b) => a.id.localeCompare(b.id))
}

function toApiMessage(message: ChatCompletionMessage): Record<string, unknown> {
  if (message.role === 'user' && message.attachments?.length) {
    return {
      role: message.role,
      content: [
        { type: 'text', text: message.content || 'Please inspect the attached image.' },
        ...message.attachments.map((attachment) => ({
          type: 'image_url',
          image_url: { url: attachment.dataUrl },
        })),
      ],
    }
  }
  return { role: message.role, content: message.content }
}

function textParts(value: unknown): string {
  if (typeof value === 'string') return value
  if (!Array.isArray(value)) return ''
  return value.map((part) => {
    if (!part || typeof part !== 'object') return ''
    const record = part as Record<string, unknown>
    return typeof record.text === 'string' ? record.text : ''
  }).join('')
}

export function extractStreamText(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return ''
  const root = payload as Record<string, unknown>
  if (typeof root.delta === 'string' && root.type === 'response.output_text.delta') return root.delta
  if (typeof root.output_text === 'string') return root.output_text
  const choices = Array.isArray(root.choices) ? root.choices : []
  const choice = choices[0] as Record<string, unknown> | undefined
  const delta = choice?.delta as Record<string, unknown> | undefined
  return textParts(delta?.content)
}

export function extractCompletionText(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return ''
  const root = payload as Record<string, unknown>
  if (typeof root.output_text === 'string') return root.output_text
  const choices = Array.isArray(root.choices) ? root.choices : []
  const choice = choices[0] as Record<string, unknown> | undefined
  const message = choice?.message as Record<string, unknown> | undefined
  return textParts(message?.content)
}

export async function streamChatCompletion(options: StreamChatOptions): Promise<void> {
  const body: Record<string, unknown> = {
    model: options.model,
    messages: options.messages.map(toApiMessage),
    stream: true,
  }
  if (typeof options.temperature === 'number' && !/^(o\d|gpt-5)/i.test(options.model)) {
    body.temperature = options.temperature
  }

  const response = await fetch(buildGatewayUrl('/v1/chat/completions'), {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${options.apiKey}`,
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    },
    body: JSON.stringify(body),
    signal: options.signal,
  })
  if (!response.ok) throw await parseGatewayError(response)

  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes('text/event-stream')) {
    const text = extractCompletionText(await response.json())
    if (text) options.onDelta(text)
    return
  }
  if (!response.body) throw new Error('The server returned an unreadable response stream')

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  const consumeLine = (line: string) => {
    if (!line.startsWith('data:')) return
    const data = line.slice(5).trim()
    if (!data || data === '[DONE]') return
    try {
      const text = extractStreamText(JSON.parse(data))
      if (text) options.onDelta(text)
    } catch {
      // Keepalive and provider-specific non-JSON events are intentionally ignored.
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done })
    const lines = buffer.split(/\r?\n/)
    buffer = done ? '' : lines.pop() || ''
    lines.forEach(consumeLine)
    if (done) break
  }
  if (buffer) consumeLine(buffer)
}

function base64ToBlob(base64: string, mimeType: string): Blob {
  const normalized = base64.includes(',') ? base64.slice(base64.indexOf(',') + 1) : base64
  const binary = atob(normalized)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index)
  return new Blob([bytes], { type: mimeType })
}

function imageMimeType(format: string): string {
  const normalized = format.toLowerCase()
  if (normalized === 'jpg' || normalized === 'jpeg') return 'image/jpeg'
  if (normalized === 'webp') return 'image/webp'
  return 'image/png'
}

function parseImageResults(payload: unknown, outputFormat: string): GatewayImageResult[] {
  return extractItems(payload).flatMap((item): GatewayImageResult[] => {
    if (!item || typeof item !== 'object') return []
    const record = item as Record<string, unknown>
    if (typeof record.b64_json === 'string' && record.b64_json) {
      return [{ blob: base64ToBlob(record.b64_json, imageMimeType(outputFormat)) }]
    }
    if (typeof record.result === 'string' && record.result) {
      const format = typeof record.output_format === 'string' ? record.output_format : outputFormat
      return [{ blob: base64ToBlob(record.result, imageMimeType(format)) }]
    }
    if (typeof record.url === 'string' && record.url) return [{ url: record.url }]
    return []
  })
}

export async function generateImage(options: GenerateImageOptions): Promise<GatewayImageResult[]> {
  const headers: Record<string, string> = { Authorization: `Bearer ${options.apiKey}` }
  let body: BodyInit
  let endpoint = '/v1/images/generations'

  if (options.mode === 'edit') {
    endpoint = '/v1/images/edits'
    const form = new FormData()
    form.append('model', options.model)
    form.append('prompt', options.prompt)
    form.append('n', '1')
    form.append('size', options.size)
    form.append('quality', options.quality)
    form.append('response_format', 'b64_json')
    form.append('output_format', options.outputFormat)
    for (const file of options.references || []) form.append('image[]', file, file.name)
    body = form
  } else {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify({
      model: options.model,
      prompt: options.prompt,
      n: 1,
      size: options.size,
      quality: options.quality,
      response_format: 'b64_json',
      output_format: options.outputFormat,
    })
  }

  const response = await fetch(buildGatewayUrl(endpoint), {
    method: 'POST',
    headers,
    body,
    signal: options.signal,
  })
  if (!response.ok) throw await parseGatewayError(response)
  const results = parseImageResults(await response.json(), options.outputFormat)
  if (!results.length) throw new Error('The image API returned no usable image data')
  return results
}
