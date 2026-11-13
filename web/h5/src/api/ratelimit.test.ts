import { http, HttpResponse } from 'msw'
import { createPinia, setActivePinia } from 'pinia'
import { server } from '../test/server'
import request, { ApiError, getAccessToken, retryAfterMs, setAccessToken } from './request'
import { withIdempotency } from './idempotency'
import { useAuthStore } from '../stores/auth'

describe('rate limit and refresh handling', () => {
  test('preserves status, retry delay and operation ID without automatic retries', async () => {
    const keys: string[] = []
    server.use(http.post('*/api/activity/v1/draw', async ({ request }) => {
      const body = await request.json() as { operation_id: string }
      keys.push(body.operation_id)
      if (keys.length === 1) return HttpResponse.json({ error: 'RATE_LIMITED' }, {
        status: 429, headers: { 'Retry-After': '3' },
      })
      return HttpResponse.json({ ok: true })
    }))
    const send = (operation_id: string) => request.post('/activity/v1/draw', { operation_id })
    await expect(withIdempotency('test.draw', { count: 30 }, send)).rejects.toMatchObject({
      status: 429, code: 'RATE_LIMITED', retryAfterMs: 3000,
    })
    expect(keys).toHaveLength(1)
    await withIdempotency('test.draw', { count: 30 }, send)
    expect(keys).toHaveLength(2)
    expect(keys[1]).toBe(keys[0])
  })

  test.each([429, 503])('refresh HTTP %s preserves existing session', async (status) => {
    server.use(
      http.get('*/api/activity/v1/records/state', () => HttpResponse.json({ error: 'INVALID_TOKEN' }, { status: 401 })),
      http.post('*/api/identity/v1/auth/refresh', () => HttpResponse.json({ error: 'RATE_LIMITED' }, {
        status, headers: { 'Retry-After': '7' },
      })),
    )
    setAccessToken('existing-token')
    await expect(request.get('/activity/v1/records/state')).rejects.toMatchObject({ status, retryAfterMs: 7000 })
    expect(getAccessToken()).toBe('existing-token')

    setActivePinia(createPinia())
    const store = useAuthStore()
    store.accessToken = 'existing-token'
    store.user = { user_id: '42', session_id: 'session', nickname: 'name' }
    await expect(store.refreshSession()).rejects.toBeInstanceOf(ApiError)
    expect(store.accessToken).toBe('existing-token')
    expect(store.user?.user_id).toBe('42')
  })

  test('initial restoration propagates temporary failures instead of invalidating auth', async () => {
    server.use(http.post('*/api/identity/v1/auth/refresh', () => HttpResponse.json({}, { status: 503 })))
    setActivePinia(createPinia())
    const store = useAuthStore()
    await expect(store.initFromStorage()).rejects.toMatchObject({ status: 503 })
  })

  test('only explicit authentication failure clears restored state', async () => {
    server.use(http.post('*/api/identity/v1/auth/refresh', () => HttpResponse.json({ error: 'INVALID_REFRESH' }, { status: 401 })))
    setActivePinia(createPinia())
    const store = useAuthStore()
    store.accessToken = 'expired'
    setAccessToken('expired')
    await expect(store.refreshSession()).resolves.toBe(false)
    expect(store.accessToken).toBeNull()
    expect(getAccessToken()).toBeNull()
  })

  test('parses both forms of Retry-After', () => {
    expect(retryAfterMs('3')).toBe(3000)
    expect(retryAfterMs(new Date(Date.now() + 10000).toUTCString())).toBeGreaterThan(8000)
    expect(retryAfterMs('invalid')).toBe(0)
  })
})
