import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, del } = vi.hoisted(() => ({
  get: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    delete: del,
  },
}))

import { deleteImageAsset, getImageAssetContent, listImageAssets } from '@/api/imageAssets'

describe('image assets api', () => {
  beforeEach(() => {
    get.mockReset()
    del.mockReset()
  })

  it('loads the authenticated user image library', async () => {
    get.mockResolvedValue({ data: { items: [{ id: 'imgasset_1', url: 'https://cdn.test/one.png' }] } })

    await expect(listImageAssets()).resolves.toEqual([{ id: 'imgasset_1', url: 'https://cdn.test/one.png' }])
    expect(get).toHaveBeenCalledWith('/user/image-assets', { signal: undefined })
  })

  it('deletes one asset through the user-scoped endpoint', async () => {
    del.mockResolvedValue({})

    await deleteImageAsset('imgasset_1/unsafe')

    expect(del).toHaveBeenCalledWith('/user/image-assets/imgasset_1%2Funsafe')
  })

  it('loads protected image content as a blob', async () => {
    const blob = new Blob(['image-bytes'], { type: 'image/png' })
    get.mockResolvedValue({ data: blob })

    await expect(getImageAssetContent('imgasset_1/unsafe')).resolves.toBe(blob)
    expect(get).toHaveBeenCalledWith('/user/image-assets/imgasset_1%2Funsafe/content', {
      signal: undefined,
      responseType: 'blob',
    })
  })
})
