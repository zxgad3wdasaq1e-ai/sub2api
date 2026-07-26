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
