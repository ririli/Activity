import request, { refreshAccessToken } from './request'

export interface LoginParams {
  account: string
  password: string
}

export interface RegisterParams {
  account: string
  password: string
  nickname?: string
}

// Login/register return only the Access token; Refresh is an HttpOnly cookie.
export interface AuthResult {
  access_token: string
  user_id: string
  nickname: string
}

export interface ProfileResult {
  user_id: string
  session_id: string
  nickname: string
}

export function login(params: LoginParams): Promise<AuthResult> {
  return request.post('/identity/v1/auth/login', params).then((res) => res.data as AuthResult)
}

export function register(params: RegisterParams): Promise<AuthResult> {
  return request.post('/identity/v1/auth/register', params).then((res) => res.data as AuthResult)
}

export function getProfile(): Promise<ProfileResult> {
  return request.get('/identity/v1/me').then((res) => res.data as ProfileResult)
}

// Restores the in-memory Access token from the HttpOnly Refresh cookie. Used on
// every page load because no token survives a reload in persistent storage.
export async function restoreSession(): Promise<AuthResult | null> {
  const token = await refreshAccessToken()
  if (!token) return null
  const profile = await getProfile()
  return {
    access_token: token,
    user_id: profile.user_id,
    nickname: profile.nickname,
  }
}

export function logout(): Promise<void> {
  return request.post('/identity/v1/auth/logout', undefined, {
    headers: { 'X-Activity-CSRF': '1' },
    withCredentials: true,
  }).then(() => undefined)
}
