import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, restoreSession as restoreApi, logout as logoutApi } from '@/api/auth'
import { setAccessToken, clearSession, isAuthenticationError } from '@/api/request'
import { resetActivitySession } from '@/api/activitySession'
import type { LoginParams, RegisterParams, AuthResult, ProfileResult } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const user = ref<ProfileResult | null>(null)
  let restorePromise: Promise<boolean> | null = null

  const isLoggedIn = computed(() => !!accessToken.value)
  const nickname = computed(() => user.value?.nickname || '')
  // The identity API no longer exposes a role; admin UI is not available.
  const isAdmin = computed(() => false)

  // Backward-compatible alias used by the SSE client and existing views.
  const token = computed(() => accessToken.value)

  function apply(result: AuthResult) {
    accessToken.value = result.access_token
    setAccessToken(result.access_token)
    user.value = {
      user_id: result.user_id,
      session_id: '',
      nickname: result.nickname,
    }
  }

  async function login(params: LoginParams): Promise<AuthResult> {
    const result = await loginApi(params)
    apply(result)
    return result
  }

  async function register(params: RegisterParams): Promise<AuthResult> {
    const result = await registerApi(params)
    apply(result)
    return result
  }

  // Used by the SSE client when the server asks the browser to re-authenticate.
  async function refreshSession(): Promise<boolean> {
    try {
      const result = await restoreApi()
      if (!result) return false
      apply(result)
      return true
    } catch (error) {
      if (!isAuthenticationError(error)) throw error
      accessToken.value = null
      user.value = null
      clearSession()
      return false
    }
  }

  async function logout(): Promise<void> {
    try {
      await logoutApi()
    } catch {
      // The server cookie is cleared on a best-effort basis; local state must
      // still be cleared even when the session was already invalid.
    } finally {
      accessToken.value = null
      user.value = null
      clearSession()
      resetActivitySession()
    }
  }

  // Restores the session from the HttpOnly Refresh cookie. Concurrency-safe so
  // the router guard, Home, and Leaderboard can all await the same refresh.
  async function initFromStorage(): Promise<boolean> {
    if (accessToken.value && user.value) return true
    if (!restorePromise) {
      restorePromise = (async () => {
        try {
          const result = await restoreApi()
          if (!result) {
            accessToken.value = null
            user.value = null
            clearSession()
            return false
          }
          apply(result)
          return true
        } catch (error) {
          if (!isAuthenticationError(error)) throw error
          accessToken.value = null
          user.value = null
          clearSession()
          return false
        } finally {
          restorePromise = null
        }
      })()
    }
    return restorePromise
  }

  return {
    accessToken,
    token,
    user,
    isLoggedIn,
    isAdmin,
    nickname,
    login,
    register,
    logout,
    initFromStorage,
    refreshSession,
  }
})
