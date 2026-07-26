const REFERENCE_IMAGE_RULES = [
  { value: 'pro-image', limit: 14 },
  { value: 'flash-image', limit: 3 },
  { value: 'gpt-image-', limit: 16 },
  { value: 'grok-imagine', limit: 3 },
  { value: 'dall-e-3', limit: 0 },
  { value: 'dall-e-2', limit: 1 },
] as const

export function referenceImageLimitForModel(model: string): number {
  const normalized = model.trim().toLowerCase()
  return REFERENCE_IMAGE_RULES.find((rule) => normalized.includes(rule.value))?.limit ?? 1
}

export function isLikelyImageModel(model: string): boolean {
  return /(image|imagen|dall-e|nano-banana)/i.test(model)
}

export const IMAGE_ASPECT_RATIOS = ['auto', '21:9', '16:9', '3:2', '4:3', '1:1', '3:4', '2:3', '9:16'] as const
export type ImageAspectRatio = typeof IMAGE_ASPECT_RATIOS[number]

/** Maps the richer studio ratio picker to sizes accepted by OpenAI-compatible image APIs. */
export function imageSizeForAspectRatio(ratio: ImageAspectRatio): string {
  if (ratio === 'auto') return 'auto'
  if (ratio === '1:1') return '1024x1024'
  const [width, height] = ratio.split(':').map(Number)
  return width > height ? '1536x1024' : '1024x1536'
}
