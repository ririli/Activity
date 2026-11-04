import { fetchEventSource } from '@microsoft/fetch-event-source'

/**
 * 创建带鉴权的 SSE 连接（替代不支持自定义 Header 的 EventSource）。
 *
 * 使用 @microsoft/fetch-event-source 库，底层基于 fetch + ReadableStream，
 * 支持 Authorization 头等原生 EventSource 不具备的能力。
 *
 * @param url          SSE 端点地址
 * @param getToken     获取最新 token 的函数，每次连接（包括重连）前调用
 * @param callbacks    事件回调
 * @param maxReconnect 最大重连次数
 * @returns { close } 用于主动断开连接
 */
export function createAuthSSE(
  url: string,
  getToken: () => string | null,
  callbacks: {
    onOpen?: () => void
    onMessage: (data: string) => void
    onError?: (reconnectCount: number) => void
    onReconnectMax?: () => void
  },
  maxReconnect = 5
): { close: () => void } {
  const controller = new AbortController()
  let reconnectCount = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function connect() {
    // 每次连接前获取最新 token，避免重连时使用已过期的旧 token
    const token = getToken()
    if (!token) {
      callbacks.onReconnectMax?.()
      return
    }

    fetchEventSource(url, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      signal: controller.signal,
      openWhenHidden: true, // 页面切后台时保持连接

      onopen: async (response) => {
        if (response.ok && response.headers.get('content-type')?.includes('text/event-stream')) {
          reconnectCount = 0
          callbacks.onOpen?.()
          return
        }
        // 非 SSE 响应，抛错触发重连
        throw new Error(`SSE 连接失败: HTTP ${response.status}`)
      },

      onmessage: (event) => {
        if (event.data) {
          callbacks.onMessage(event.data)
        }
      },

      onclose: () => {
        // 服务端主动关闭连接（正常情况不触发）
      },

      onerror: (err) => {
        reconnectCount++
        callbacks.onError?.(reconnectCount)

        if (reconnectCount >= maxReconnect) {
          callbacks.onReconnectMax?.()
          controller.abort()
          throw err // 不再重试
        }

        // 手动管理重连：抛弃库的默认重试，自己用指数退避 + 每次拿最新 token
        // 这样可以保证即使 access token 过期且已被刷新，重连使用的是新 token
        const delay = Math.min(1000 * Math.pow(2, reconnectCount - 1), 30000)
        reconnectTimer = setTimeout(connect, delay)
        throw err // 阻止库的默认重试，使用我们自己的重连逻辑
      },
    })
  }

  connect()

  return {
    close: () => {
      controller.abort()
      if (reconnectTimer != null) clearTimeout(reconnectTimer)
    },
  }
}
