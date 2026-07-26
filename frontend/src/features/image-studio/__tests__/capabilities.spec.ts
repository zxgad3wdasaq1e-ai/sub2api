import { describe, expect, it } from 'vitest'
import { imageSizeForAspectRatio, isLikelyImageModel, referenceImageLimitForModel } from '../capabilities'

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

  it('maps studio aspect ratios to compatible gateway sizes', () => {
    expect(imageSizeForAspectRatio('auto')).toBe('auto')
    expect(imageSizeForAspectRatio('21:9')).toBe('1536x1024')
    expect(imageSizeForAspectRatio('1:1')).toBe('1024x1024')
    expect(imageSizeForAspectRatio('9:16')).toBe('1024x1536')
  })
})
