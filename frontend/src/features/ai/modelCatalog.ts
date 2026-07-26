import type { UserPricedModel } from '@/api/channels'
import type { ApiKey } from '@/types'

export function findPricedModel(catalog: UserPricedModel[], model: string): UserPricedModel | null {
  const normalized = model.trim().toLowerCase()
  if (!normalized) return null
  return catalog.find((item) => item.name.trim().toLowerCase() === normalized) || null
}

export function compatibleApiKeys(
  keys: ApiKey[],
  catalog: UserPricedModel[],
  model: string,
  predicate?: (key: ApiKey) => boolean,
): ApiKey[] {
  const entry = findPricedModel(catalog, model)
  if (!entry) return []
  const groupIDs = new Set(entry.group_ids.map(Number))
  return keys.filter((key) => (
    key.status === 'active' &&
    key.group_id !== null &&
    groupIDs.has(Number(key.group_id)) &&
    (!predicate || predicate(key))
  ))
}

export function selectCompatibleApiKey(
  keys: ApiKey[],
  catalog: UserPricedModel[],
  model: string,
  preferredID?: number | null,
  predicate?: (key: ApiKey) => boolean,
): ApiKey | null {
  const compatible = compatibleApiKeys(keys, catalog, model, predicate)
  return compatible.find((key) => key.id === Number(preferredID)) || compatible[0] || null
}
