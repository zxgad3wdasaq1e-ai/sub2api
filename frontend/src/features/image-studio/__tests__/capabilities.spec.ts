import { describe, expect, it } from 'vitest'
import { isLikelyImageModel, referenceImageLimitForModel } from '../capabilities'

describe('image studio model capabilities', () => {
  it('uses the reference limits from the standalone image studio', () => {
    expect(referenceImageLimitForModel('gpt-image-2')).toBe(16)
    expect(referenceImageLimitForModel('gemini-2.5-flash-image')).toBe(3)
    expect(referenceImageLimitForModel('gemini-3-pro-image-preview')).toBe(14)
    expect(referenceImageLimitForModel('grok-imagine-image')).toBe(3)
    expect(referenceImageLimitForModel('dall-e-3')).toBe(0)
    expect(referenceImageLimitForModel('custom-model')).toBe(1)
  })

  it('recognizes common image model names', () => {
    expect(isLikelyImageModel('gpt-image-2')).toBe(true)
    expect(isLikelyImageModel('gemini-2.5-flash-image')).toBe(true)
    expect(isLikelyImageModel('gpt-5.4')).toBe(false)
  })
})
