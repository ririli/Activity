import { fetchEventSource } from '@microsoft/fetch-event-source'
import type { FetchEventSourceInit } from '@microsoft/fetch-event-source'
import { createAuthSSE } from './sse-client'

vi.mock('@microsoft/fetch-event-source', () => ({ fetchEventSource: vi.fn(() => new Promise(() => {})) }))

test('SSE waits for Retry-After plus jitter and cancels pending reconnects', async () => {
  vi.useFakeTimers()
  const random = vi.spyOn(Math, 'random').mockReturnValue(0.5)
  vi.mocked(fetchEventSource).mockClear()
  const client = createAuthSSE('/api/notification/v1/barrage/stream', () => 'token', {})
  try {
    const options = vi.mocked(fetchEventSource).mock.calls[0][1] as FetchEventSourceInit
    await expect(options.onopen!(new Response('', { status: 429, headers: { 'Retry-After': '7' } }))).rejects.toThrow()
    expect(() => options.onerror!(new Error('limited'))).toThrow()
    await vi.advanceTimersByTimeAsync(7499)
    expect(fetchEventSource).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(fetchEventSource).toHaveBeenCalledTimes(2)
    const second = vi.mocked(fetchEventSource).mock.calls[1][1] as FetchEventSourceInit
    expect(() => second.onerror!(new Error('offline'))).toThrow()
    client.close()
    await vi.advanceTimersByTimeAsync(60000)
    expect(fetchEventSource).toHaveBeenCalledTimes(2)
  } finally {
    client.close()
    random.mockRestore()
    vi.useRealTimers()
  }
})
