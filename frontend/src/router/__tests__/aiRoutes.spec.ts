import { describe, expect, it } from 'vitest'
import router from '../index'

describe('AI workspace routes', () => {
  it('exposes chat and image studio to authenticated users', () => {
    expect(router.getRoutes().find((route) => route.path === '/chat')?.meta.requiresAuth).toBe(true)
    expect(router.getRoutes().find((route) => route.path === '/chat')?.meta.requiresAdmin).toBe(false)
    expect(router.getRoutes().find((route) => route.path === '/image-studio')?.meta.requiresAuth).toBe(true)
    expect(router.getRoutes().find((route) => route.path === '/image-studio')?.meta.requiresAdmin).toBe(false)
  })

  it('restricts usage ranking to administrators', () => {
    expect(router.getRoutes().find((route) => route.path === '/usage-ranking')?.meta.requiresAdmin).toBe(true)
  })
})
