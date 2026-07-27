import { afterEach, describe, expect, it, vi } from 'vitest'
import { extractCompletionText, extractStreamText, generateImage } from '../gateway'

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('AI gateway response parsing', () => {
  it('extracts chat completion text from string and multipart responses', () => {
    expect(extractCompletionText({ choices: [{ message: { content: 'hello' } }] })).toBe('hello')
    expect(extractCompletionText({ choices: [{ message: { content: [{ type: 'text', text: 'one' }, { type: 'text', text: ' two' }] } }] })).toBe('one two')
  })

  it('extracts deltas from chat-completions and responses events', () => {
    expect(extractStreamText({ choices: [{ delta: { content: 'next' } }] })).toBe('next')
    expect(extractStreamText({ type: 'response.output_text.delta', delta: ' token' })).toBe(' token')
  })

  it('ignores provider metadata without displayable text', () => {
    expect(extractStreamText({ choices: [{ delta: { role: 'assistant' } }] })).toBe('')
  })

  it('sends the image prompt exactly as provided and ignores provider rewrites', async () => {
    const prompt = '严格使用这段用户提示词，不添加任何内容'
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: [{ b64_json: 'aGVsbG8=', revised_prompt: 'provider rewrite' }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    const results = await generateImage({
      apiKey: 'test-key',
      model: 'gpt-image-2',
      prompt,
      mode: 'text',
      size: '1024x1024',
      quality: 'auto',
      outputFormat: 'png',
    })

    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(JSON.parse(request.body as string).prompt).toBe(prompt)
    expect(results).toHaveLength(1)
    expect(results[0]).not.toHaveProperty('revisedPrompt')
  })
})
