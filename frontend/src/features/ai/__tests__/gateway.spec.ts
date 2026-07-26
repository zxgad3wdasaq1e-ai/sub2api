import { describe, expect, it } from 'vitest'
import { extractCompletionText, extractStreamText } from '../gateway'

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
})
