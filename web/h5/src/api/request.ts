import axios from 'axios'
import { showDialog, showToast } from 'vant'
import { clearPendingIdempotencyKeys } from './idempotency'

// —— Token 存储键 ——
const ACCESS_TOKEN_KEY = 'lottery_access_token'
const REFRESH_TOKEN_KEY = 'lottery_refresh_token'

// —— Token 读写工具函数 ——
export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function setTokens(accessToken: string, refreshToken: string): void {
  clearPendingIdempotencyKeys()
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  clearPendingIdempotencyKeys()
}

// 向后兼容：旧代码中 getToken 返回 access token
export function getToken(): string | null {
  return getAccessToken()
}

// 创建 Axios 实例
const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: { 'Content-Type': 'application/json' }
})

// 请求拦截器：自动附加 access token
request.interceptors.request.use(
  (config) => {
    const token = getAccessToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// —— 并发刷新竞态控制 ——
let isRefreshing = false
let refreshSubscribers: ((token: string) => void)[] = []

function onRefreshed(token: string) {
  refreshSubscribers.forEach((cb) => cb(token))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

// 执行 token 刷新（不经过拦截器，避免死循环）
async function doRefresh(): Promise<string> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    throw new Error('无 refresh token')
  }

  // 用独立的 axios 实例调用 refresh，不走拦截器
  const res = await axios.post('/api/v1/auth/refresh', {
    refresh_token: refreshToken,
  })

  const authResult = res.data.data as {
    access_token: string
    refresh_token: string
    expires_in: number
  }

  setTokens(authResult.access_token, authResult.refresh_token)
  return authResult.access_token
}

// 清除登录状态并跳转
function forceLogout(message: string) {
  clearTokens()
  showDialog({
    title: '提示',
    message,
  }).then(() => {
    window.location.href = '/login'
  })
}

// 响应拦截器：统一处理错误 + 自动刷新
request.interceptors.response.use(
  (response) => {
    const { data } = response
    // 业务错误
    if (data.code !== 0) {
      // Refresh Token 无效 → 强制重新登录
      if (data.code === 10010) {
        forceLogout(data.message || '刷新令牌已过期，请重新登录')
        return Promise.reject(new Error(data.message))
      }
      // Access Token 无效（被黑名单等）→ 强制重新登录
      if (data.code === 10002) {
        forceLogout(data.message || '会话已过期，请重新登录')
        return Promise.reject(new Error(data.message))
      }
      // 请求过于频繁（限流）
      if (data.code === 60005) {
        showToast(data.message || '请求过于频繁，请稍后再试')
        return Promise.reject(new Error(data.message))
      }
      return Promise.reject(new Error(data.message))
    }
    return data
  },
  async (error) => {
    const responseData = axios.isAxiosError(error)
      ? (error.response?.data as { code?: number; message?: string } | undefined)
      : undefined

    const originalRequest = error.config

    // Access Token 过期（10009）→ 自动刷新
    if (
      responseData?.code === 10009 &&
      originalRequest &&
      !(originalRequest as any)._retry
    ) {
      // 如果已有刷新正在进行中，排队等待
      if (isRefreshing) {
        return new Promise((resolve) => {
          addRefreshSubscriber((newToken: string) => {
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            resolve(request(originalRequest))
          })
        })
      }

      (originalRequest as any)._retry = true
      isRefreshing = true

      try {
        const newToken = await doRefresh()
        isRefreshing = false
        onRefreshed(newToken)

        // 用新 token 重试原请求
        originalRequest.headers.Authorization = `Bearer ${newToken}`
        return request(originalRequest)
      } catch (refreshError) {
        isRefreshing = false
        refreshSubscribers = []
        forceLogout('会话已过期，请重新登录')
        return Promise.reject(refreshError)
      }
    }

    // Refresh Token 无效（10010）
    if (responseData?.code === 10010) {
      forceLogout(responseData.message || '刷新令牌已过期，请重新登录')
    }

    // Token 无效（10002）
    if (responseData?.code === 10002) {
      forceLogout(responseData.message || '会话已过期，请重新登录')
    }

    const message = responseData?.message || error.message || '网络异常，请稍后重试'
    return Promise.reject(new Error(message))
  }
)

export default request
