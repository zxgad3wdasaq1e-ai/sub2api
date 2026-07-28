import { describe, expect, it } from 'vitest'
import type { UserPricedModel } from '@/api/channels'
import type { ApiKey } from '@/types'
import { compatibleApiKeys, selectCompatibleApiKey } from '../modelCatalog'

function key(id: number, groupID: number | null, status: ApiKey['status'] = 'active'): ApiKey {
  return { id, group_id: groupID, status } as ApiKey
}

const catalog: UserPricedModel[] = [
  { name: 'gpt-5.4', platforms: ['openai'], group_ids: [3, 7] },
  { name: 'gpt-image-2', platforms: ['openai'], group_ids: [9] },
]

describe('priced model API key binding', () => {
  it('only returns active keys bound to a model group', () => {
    expect(compatibleApiKeys([
      key(1, null),
      key(2, 3),
      key(3, 7, 'inactive'),
      key(4, 9),
    ], catalog, 'GPT-5.4').map((item) => item.id)).toEqual([2])
  })

  it('keeps a compatible preferred key and otherwise selects the first match', () => {
    const keys = [key(1, 3), key(2, 7)]
    expect(selectCompatibleApiKey(keys, catalog, 'gpt-5.4', 2)?.id).toBe(2)
    expect(selectCompatibleApiKey(keys, catalog, 'gpt-5.4', 99)?.id).toBe(1)
    expect(selectCompatibleApiKey(keys, catalog, 'missing', 1)).toBeNull()
  })
})
