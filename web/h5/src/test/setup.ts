import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/vue'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { resetApiState, server } from './server'
import { resetActivitySession } from '@/api/activitySession'
import { clearRecordCaches } from '@/api/records'
import { clearSession } from '@/api/request'
import { resetTryBeforePayCache } from '@/api/tbp'
import { resetStageChoiceCache } from '@/api/stage'

class MemoryStorage implements Storage {
  private values = new Map<string, string>()

  get length() { return this.values.size }
  clear() { this.values.clear() }
  getItem(key: string) { return this.values.get(key) ?? null }
  key(index: number) { return [...this.values.keys()][index] ?? null }
  removeItem(key: string) { this.values.delete(key) }
  setItem(key: string, value: string) { this.values.set(key, String(value)) }
}

const storage = new MemoryStorage()
Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: storage })
Object.defineProperty(window, 'localStorage', { configurable: true, value: storage })
Object.defineProperty(window.HTMLMediaElement.prototype, 'play', {
  configurable: true,
  value: vi.fn().mockResolvedValue(undefined),
})
Object.defineProperty(window.HTMLMediaElement.prototype, 'pause', {
  configurable: true,
  value: vi.fn(),
})

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
  // jsdom's AbortSignal is not an undici AbortSignal. fetch-event-source passes
  // the signal through to fetch, which undici rejects in Node 24. The MSW
  // interceptor does not need the signal for these short test responses.
  const intercepted = globalThis.fetch
  globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
    if (init && init.signal) {
      const { signal: _signal, ...rest } = init
      return intercepted(input, rest)
    }
    return intercepted(input, init)
  }) as typeof fetch
})

afterEach(() => {
  cleanup()
  localStorage.clear()
  clearSession()
  resetActivitySession()
  clearRecordCaches()
  resetTryBeforePayCache()
  resetStageChoiceCache()
  resetApiState()
  server.resetHandlers()
})

afterAll(() => server.close())
