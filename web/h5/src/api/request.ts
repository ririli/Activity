import axios from 'axios'
import { showDialog, showToast } from 'vant'
import { clearPendingIdempotencyKeys } from './idempotency'

// Access is intentionally memory-only. The rotating Refresh token is an
// HttpOnly cookie managed by identity-service, never readable by JavaScript.
let accessToken: string | null = null
let refreshPromise: Promise<string> | null = null

const LEGACY_ACCESS_KEY = 'lottery_access_token'
const LEGACY_REFRESH_KEY = 'lottery_refresh_token'
const REFRESH_LOCK_KEY = 'activity_refresh_lock'
const AUTH_PREFIX = '/identity/v1/auth/'

function browserStorage(): Storage | null {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage
  } catch {
    return null
  }
}

// Remove the old protocol's persisted tokens once; users without the new
// HttpOnly cookie simply go through the login page.
const legacyStorage = browserStorage()
legacyStorage?.removeItem(LEGACY_ACCESS_KEY)
legacyStorage?.removeItem(LEGACY_REFRESH_KEY)

export function getAccessToken(): string | null {
  return accessToken
}

// Backward-compatible alias used by the router and existing views.
export function getToken(): string | null {
  return accessToken
}

export function setAccessToken(token: string | null): void {
  accessToken = token
}

export function clearSession(): void {
  accessToken = null
  clearPendingIdempotencyKeys()
  browserStorage()?.removeItem(LEGACY_ACCESS_KEY)
  browserStorage()?.removeItem(LEGACY_REFRESH_KEY)
}

type LockManagerLike = {
  request<T>(name: string, callback: () => Promise<T>): Promise<T>
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function withRefreshLock<T>(callback: () => Promise<T>): Promise<T> {
  const locks = (typeof navigator !== 'undefined'
    ? (navigator as Navigator & { locks?: LockManagerLike }).locks
    : undefined)
  if (locks?.request) {
    return locks.request('activity-refresh', callback)
  }
  // Fallback for browsers without Web Locks. The cookie itself is shared by
  // all tabs; the lock only prevents concurrent refresh rotations.
  const storage = browserStorage()
  if (!storage) return callback()
  const owner = `${Date.now()}-${Math.random().toString(36).slice(2)}`
  for (let attempt = 0; attempt < 50; attempt += 1) {
    if (storage.getItem(REFRESH_LOCK_KEY) === null) {
      storage.setItem(REFRESH_LOCK_KEY, owner)
      try {
        return await callback()
      } finally {
        if (storage.getItem(REFRESH_LOCK_KEY) === owner) storage.removeItem(REFRESH_LOCK_KEY)
      }
    }
    await sleep(50)
  }
  return callback()
}

// Exchanges the HttpOnly Refresh cookie for a new Access token. This is also
// used on page load: there is no Access token in persistent storage.
export function refreshAccessToken(): Promise<string> {
  if (refreshPromise) return refreshPromise
  refreshPromise = withRefreshLock(async () => {
    const response = await axios.post(
      '/api/identity/v1/auth/refresh',
      undefined,
      {
        headers: { 'X-Activity-CSRF': '1' },
        timeout: 10000,
        withCredentials: true,
      },
    ).catch((error: unknown) => { throw toApiError(error) })
    const token = (response.data as { access_token?: string }).access_token
    if (!token) throw new Error('登录状态刷新失败')
    accessToken = token
    return token
  }).finally(() => {
    refreshPromise = null
  })
  return refreshPromise
}

const request = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

request.interceptors.request.use((config) => {
  if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`
  return config
})

export class ApiError extends Error {
  constructor(message: string, public readonly status?: number, public readonly code?: string,
    public readonly retryAfterMs = 0) {
    super(message)
    this.name = 'ApiError'
  }
}
export function retryAfterMs(value: string | null | undefined): number {
  if (!value) return 0
  const seconds = Number(value)
  if (Number.isFinite(seconds)) return Math.max(0, seconds * 1000)
  const date = Date.parse(value)
  return Number.isNaN(date) ? 0 : Math.max(0, date - Date.now())
}
export function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error
  if (axios.isAxiosError(error)) {
    const status = error.response?.status
    const code = (error.response?.data as { error?: string } | undefined)?.error
    return new ApiError(messageFor(status, code), status, code,
      retryAfterMs(error.response?.headers['retry-after'] as string | undefined))
  }
  return new ApiError(error instanceof Error ? error.message : '网络异常，请稍后重试')
}
export function isAuthenticationError(error: unknown): boolean {
  return toApiError(error).status === 401
}

const ERROR_TEXT: Record<string, string> = {
  INVALID_REQUEST: '请求参数不正确',
  INVALID_TOKEN: '登录状态已失效，请重新登录',
  INVALID_SESSION: '会话已失效，请重新登录',
  INVALID_REFRESH: '登录状态已过期，请重新登录',
  INVALID_CREDENTIALS: '账号或密码错误',
  ACCOUNT_EXISTS: '账号已存在',
  RATE_LIMITED: '操作过于频繁，请稍后重试',
  LOGIN_LIMITED: '登录失败次数过多，请稍后重试',
  OPERATION_IN_PROGRESS: '操作进行中，请稍后重试',
  OPERATION_RESULT_UNKNOWN: '处理结果未知，请使用相同操作重试',
  ACTIVITY_NOT_ACTIVE: '活动未开始或已结束',
  INSUFFICIENT_PETALS: '花瓣不足',
  INSUFFICIENT_OR_OVERFLOWING_ASSET: '金币不足或花瓣不足',
  INSUFFICIENT_GOLD: '金币不足，请先购买金币',
  ROUND_FULL: '本轮花朵已满',
  STAGE_NOT_REACHED: '尚未达到该阶段',
  NO_CHEST_AVAILABLE: '暂无可开启的宝箱',
  ALREADY_PURCHASED: '该档位已购买',
  ACCOUNT_INITIALIZING: '账号初始化中，请稍后重试',
  ACCOUNT_DISABLED: '账号已被禁用',
}

function messageFor(status: number | undefined, code: string | undefined): string {
  if (code && ERROR_TEXT[code]) return ERROR_TEXT[code]
  if (status === 401) return '登录状态已失效，请重新登录'
  if (status === 403) return '请求来源校验失败'
  if (status === 429) return '操作过于频繁，请稍后重试'
  if (status === 503) return '服务暂时不可用，请稍后重试'
  return code ? `请求失败：${code}` : '网络异常，请稍后重试'
}

let loggingOut = false

function forceLogout(message: string): void {
  if (loggingOut) return
  loggingOut = true
  clearSession()
  showDialog({ title: '提示', message }).then(() => {
    window.location.href = '/login'
  }).finally(() => {
    loggingOut = false
  })
}

function isAuthEndpoint(url: string | undefined): boolean {
  return !!url && url.includes(AUTH_PREFIX)
}

request.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (!axios.isAxiosError(error)) return Promise.reject(error)
    const status = error.response?.status
    const code = (error.response?.data as { error?: string } | undefined)?.error
    const original = error.config as (typeof error.config & { _retry?: boolean }) | undefined

    if (isAuthEndpoint(original?.url) && (original?.url?.includes('/login') || original?.url?.includes('/register'))) {
      return Promise.reject(toApiError(error))
    }

    if (status === 401 && original && !original._retry && !isAuthEndpoint(original.url)) {
      original._retry = true
      try {
        const token = await refreshAccessToken()
        original.headers = original.headers ?? {}
        original.headers.Authorization = `Bearer ${token}`
        return request(original)
      } catch (refreshError) {
        const failure = toApiError(refreshError)
        if (isAuthenticationError(failure)) forceLogout('登录状态已过期，请重新登录')
        else showToast(failure.message)
        return Promise.reject(failure)
      }
    }

    if (status === 401) {
      forceLogout(messageFor(status, code))
    } else if (status === 429) {
      showToast(messageFor(status, code))
    }
    return Promise.reject(toApiError(error))
  },
)

export default request
