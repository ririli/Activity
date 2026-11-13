import { fetchEventSource } from '@microsoft/fetch-event-source'
import { retryAfterMs } from './request'

export interface AuthSSECallbacks {
  onOpen?: () => void
  onEvent?: (event: string, data: string) => void
  onAuthRequired?: () => void
  onUnavailable?: () => void
  onError?: (reconnectCount: number) => void
  onReconnectMax?: () => void
}

/**
 * SSE with Bearer auth via fetch. Control events (ready/auth_required/
 * unavailable) are surfaced separately from barrage data; the caller owns the
 * auth refresh and reconnect policy.
 */
export function createAuthSSE(
  url: string,
  getToken: () => string | null,
  callbacks: AuthSSECallbacks,
  maxReconnect = 5,
): { close: () => void } {
  let controller = new AbortController()
  let stopped = false
  let intentional = false
  let reconnectCount = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  let serverDelay = 0

  function retry(error: unknown) {
    if (intentional || stopped) return
    reconnectCount += 1
    callbacks.onError?.(reconnectCount)
    if (reconnectCount >= maxReconnect) {
      callbacks.onReconnectMax?.()
      return
    }
    if (reconnectTimer !== null) return
    const delay = Math.max(serverDelay, Math.min(1000 * Math.pow(2, reconnectCount - 1), 30000)) + Math.floor(Math.random() * 1000)
    serverDelay = 0
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
    throw error
  }

  function connect() {
    if (stopped) return
    const token = getToken()
    if (!token) {
      callbacks.onAuthRequired?.()
      return
    }
    intentional = false
    controller = new AbortController()
    const current = controller
    fetchEventSource(url, {
      headers: { Authorization: `Bearer ${token}` },
      signal: current.signal,
      openWhenHidden: true,

      onopen: async (response) => {
        if (response.ok && response.headers.get('content-type')?.includes('text/event-stream')) {
          reconnectCount = 0
          callbacks.onOpen?.()
          return
        }
        if (response.status === 401) {
          intentional = true
          callbacks.onAuthRequired?.()
          current.abort()
          return
        }
        serverDelay = retryAfterMs(response.headers.get('Retry-After'))
        throw new Error(`SSE 连接失败: HTTP ${response.status}`)
      },

      onmessage: (event) => {
        const type = event.event || 'barrage'
        if (type === 'ready') {
          reconnectCount = 0
          callbacks.onOpen?.()
          return
        }
        if (type === 'auth_required') {
          intentional = true
          callbacks.onAuthRequired?.()
          current.abort()
          return
        }
        if (type === 'unavailable') {
          intentional = true
          callbacks.onUnavailable?.()
          current.abort()
          return
        }
        if (event.data) callbacks.onEvent?.(type, event.data)
      },

      onclose: () => {
        if (!stopped && !intentional) {
          intentional = true
          current.abort()
          intentional = false
          retry(new Error('SSE closed'))
        }
      },

      onerror: (error) => {
        if (intentional || stopped) return
        retry(error)
      },
    }).catch(() => {
      // Errors are classified by retry/control-event callbacks above.
    })
  }

  connect()

  return {
    close: () => {
      stopped = true
      intentional = true
      controller.abort()
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
    },
  }
}
