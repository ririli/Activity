import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, register as registerApi, getProfile, logout as logoutApi } from '@/api/auth'
import { setTokens, clearTokens, getAccessToken, getRefreshToken } from '@/api/request'
import type { LoginParams, RegisterParams, AuthResult, ProfileResult } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  // ---- 状态 ----
  const accessToken = ref<string | null>(getAccessToken())
  const refreshToken = ref<string | null>(getRefreshToken())
  const user = ref<ProfileResult | null>(null)

  // ---- 计算属性 ----
  const isLoggedIn = computed(() => !!accessToken.value)
  const nickname = computed(() => user.value?.nickname || user.value?.username || '')
  const isAdmin = computed(() => user.value?.role === 1)

  // 向后兼容：旧代码中 authStore.token 返回 access token
  const token = computed(() => accessToken.value)

  // ---- 方法 ----

  // 登录
  async function login(params: LoginParams): Promise<AuthResult> {
    const result = await loginApi(params)
    accessToken.value = result.access_token
    refreshToken.value = result.refresh_token
    user.value = {
      user_id: result.user_id,
      username: result.username,
      nickname: result.nickname,
      role: result.role,
    }
    setTokens(result.access_token, result.refresh_token)
    return result
  }

  // 注册（注册后自动登录）
  async function register(params: RegisterParams): Promise<AuthResult> {
    const result = await registerApi(params)
    accessToken.value = result.access_token
    refreshToken.value = result.refresh_token
    user.value = {
      user_id: result.user_id,
      username: result.username,
      nickname: result.nickname,
      role: result.role,
    }
    setTokens(result.access_token, result.refresh_token)
    return result
  }

  // 退出登录
  async function logout(): Promise<void> {
    try {
      await logoutApi(refreshToken.value || undefined)
    } catch {
      // 即便 API 调用失败也清除本地状态
    }
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    clearTokens()
  }

  // 从 localStorage 恢复登录状态
  async function initFromStorage(): Promise<boolean> {
    const savedToken = getAccessToken()
    if (!savedToken) return false

    try {
      accessToken.value = savedToken
      const profile = await getProfile()
      user.value = profile
      return true
    } catch {
      // Token 无效或已过期（拦截器会处理清理和跳转）
      accessToken.value = null
      refreshToken.value = null
      clearTokens()
      return false
    }
  }

  return {
    accessToken,
    refreshToken,
    token,        // 向后兼容
    user,
    isLoggedIn,
    isAdmin,
    nickname,
    login,
    register,
    logout,
    initFromStorage,
  }
})
