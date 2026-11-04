import request from './request'

// 登录请求
export interface LoginParams {
  username: string
  password: string
}

// 注册请求
export interface RegisterParams {
  username: string
  password: string
  nickname?: string
}

// 认证返回（登录/注册/刷新 共用）
export interface AuthResult {
  access_token: string
  refresh_token: string
  expires_in: number  // access token 剩余有效秒数
  user_id: number
  username: string
  nickname: string
  role: number  // 0=普通用户, 1=管理员
}

// 用户资料
export interface ProfileResult {
  user_id: number
  username: string
  nickname: string
  role: number  // 0=普通用户, 1=管理员
}

// 资产余额
export interface BalanceResult {
  gold: number
  petals: number
}

// 登录
export function login(params: LoginParams): Promise<AuthResult> {
  return request.post('/auth/login', params).then((res) => res.data)
}

// 注册
export function register(params: RegisterParams): Promise<AuthResult> {
  return request.post('/auth/register', params).then((res) => res.data)
}

// 刷新令牌
export function refreshTokens(refreshToken: string): Promise<AuthResult> {
  return request.post('/auth/refresh', { refresh_token: refreshToken }).then((res) => res.data)
}

// 获取用户资料
export function getProfile(): Promise<ProfileResult> {
  return request.get('/auth/profile').then((res) => res.data)
}

// 退出登录（传入 refresh_token 供服务端清理）
export function logout(refreshToken?: string): Promise<void> {
  return request.post('/auth/logout', { refresh_token: refreshToken || '' })
}
