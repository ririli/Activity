import request from './request'

// ---- 花朵点亮记录 ----
export interface LitLogItem {
  id: number
  pool_type: number
  lit_count: number
  created_at: string
}

export interface LitLogsResponse {
  list: LitLogItem[]
  total: number
  page: number
}

// ---- 戒指召唤记录 ----
export interface RingRecordItem {
  id: number
  item_name: string
  source: string
  created_at: string
}

export interface RingRecordsResponse {
  list: RingRecordItem[]
  total: number
  page: number
}

// ---- 抽奖历史（复用 lottery API 的类型） ----
import type { DrawResponse } from './lottery'

export interface DrawHistoryItem {
  draw_log_id: number
  pool_type: number
  pool_name: string
  draw_mode: number
  requested_count: number
  actual_count: number
  cost_petals: number
  refund_petals: number
  lit_flowers: number
  rewards: DrawResponse['rewards']
  created_at: string
}

export interface DrawHistoryResponse {
  total: number
  page: number
  page_size: number
  list: DrawHistoryItem[]
}

// ---- API 函数 ----

// 查询花朵点亮记录
export function fetchLitLogs(page = 1): Promise<LitLogsResponse> {
  return request.get('/flower/lit-logs', { params: { page } }).then((res) => res.data)
}

// 查询戒指召唤记录
export function fetchRingRecords(page = 1): Promise<RingRecordsResponse> {
  return request.get('/rewards/rings', { params: { page } }).then((res) => res.data)
}

// 查询抽奖历史
export function fetchDrawHistory(page = 1): Promise<DrawHistoryResponse> {
  return request.get('/lottery/draw/history', { params: { page } }).then((res) => res.data)
}
