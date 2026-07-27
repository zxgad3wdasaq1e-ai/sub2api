import { afterEach, describe, expect, it, vi } from 'vitest'
import { extractCompletionText, extractStreamText, generateImage, getImageTask, submitImageTask } from '../gateway'

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

  it('sends image edits as multipart image inputs without a JSON content type', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      data: [{ b64_json: 'aGVsbG8=' }],
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const first = new File(['first'], 'first.png', { type: 'image/png' })
    const second = new File(['second'], 'second.png', { type: 'image/png' })

    await generateImage({
      apiKey: 'test-key',
      model: 'gpt-image-2',
      prompt: 'replace the background',
      mode: 'edit',
      size: '1024x1024',
      quality: 'auto',
      outputFormat: 'png',
      references: [first, second],
    })

    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(request.headers).toEqual({ Authorization: 'Bearer test-key' })
    const form = request.body as FormData
    expect(form.get('model')).toBe('gpt-image-2')
    expect(form.get('prompt')).toBe('replace the background')
    expect(form.get('image')).toMatchObject({ name: 'first.png', size: 5, type: 'image/png' })
    expect(form.get('image[]')).toMatchObject({ name: 'second.png', size: 6, type: 'image/png' })
  })

  it('submits text generation as a server-side image task', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      task_id: 'imgtask_123',
      status: 'processing',
    }), { status: 202, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    const task = await submitImageTask({
      apiKey: 'test-key',
      model: 'gpt-image-2',
      prompt: '画幅比例 21:9 生成一个男孩',
      mode: 'text',
      size: '1536x1024',
      quality: 'auto',
      outputFormat: 'png',
    })

    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/v1/images/generations/async')
    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(JSON.parse(request.body as string).prompt).toBe('画幅比例 21:9 生成一个男孩')
    expect(task).toEqual({ taskId: 'imgtask_123', status: 'processing' })
  })

  it('submits image edits as a multipart server-side task', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: 'imgtask_edit',
      status: 'processing',
    }), { status: 202, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const reference = new File(['source'], 'source.png', { type: 'image/png' })

    await submitImageTask({
      apiKey: 'test-key',
      model: 'gpt-image-2',
      prompt: '只替换背景',
      mode: 'edit',
      size: '1024x1024',
      quality: 'auto',
      outputFormat: 'png',
      references: [reference],
    })

    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/v1/images/edits/async')
    const request = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect(request.headers).toEqual({ Authorization: 'Bearer test-key' })
    expect((request.body as FormData).get('image')).toMatchObject({ name: 'source.png' })
  })

  it('polls processing, completed and failed image tasks', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        task_id: 'imgtask_123',
        status: 'processing',
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        task_id: 'imgtask_123',
        status: 'completed',
        result: { data: [{ b64_json: 'aGVsbG8=' }] },
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        task_id: 'imgtask_failed',
        status: 'failed',
        error: { message: 'provider rejected the request' },
      }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(getImageTask('test-key', 'imgtask_123', 'png')).resolves.toEqual({
      taskId: 'imgtask_123',
      status: 'processing',
    })
    const completed = await getImageTask('test-key', 'imgtask_123', 'png')
    expect(completed.status).toBe('completed')
    expect(completed.results).toHaveLength(1)
    await expect(getImageTask('test-key', 'imgtask_failed', 'png')).resolves.toEqual({
      taskId: 'imgtask_failed',
      status: 'failed',
      error: 'provider rejected the request',
    })
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain('/v1/images/tasks/imgtask_123')
    expect((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).toEqual({
      Authorization: 'Bearer test-key',
      Accept: 'application/json',
    })
  })
})
