import request from './request'

// ---- 先抽后付状态 ----
export interface TryBeforePaySession {
  session_id: number
  status: number
  total_cost: number
  created_at: string
}

export interface TryBeforePayStatus {
  available: boolean
  session?: TryBeforePaySession
  preview?: TryBeforePayPreview
}

// ---- 先抽后付预览 ----
export interface TryBeforePayRewardSummary {
  item_id: number
  item_name: string
  total_quantity: number
  is_choice: boolean
}

export interface TryBeforePayPreview {
  session_id: number
  total_draws: number
  total_cost: number
  reward_summary: TryBeforePayRewardSummary[]
  estimated_lit_flowers: number
  estimated_box_chances: number
}

export interface TryBeforePayConfirmResult {
  session_id: number
  total_cost: number
  draw_log_id: number
}

// ---- 查询先抽后付入口状态 ----
export function fetchTryBeforePayStatus(): Promise<TryBeforePayStatus> {
  return request.get('/tbp/status').then((res) => res.data)
}

// ---- 生成先抽后付预览 ----
export function previewTryBeforePay(): Promise<TryBeforePayPreview> {
  return request.post('/tbp/preview').then((res) => res.data)
}

// ---- 确认先抽后付 ----
export function confirmTryBeforePay(): Promise<TryBeforePayConfirmResult> {
  return request.post('/tbp/confirm').then((res) => res.data)
}

// ---- 放弃先抽后付预览 ----
export function abandonTryBeforePay(): Promise<void> {
  return request.post('/tbp/abandon').then(() => undefined)
}
