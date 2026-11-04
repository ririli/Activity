import request from './request'
import { idempotencyOperations, withIdempotency } from './idempotency'

// ---- 奖池配置 ----
export interface PoolConfig {
  pool_type: number   // 1=白昼, 2=星夜
  pool_name: string
  cost_per_draw: number
  status: number
  prizes: PoolPrize[]
}

export interface PoolPrize {
  id: number
  pool_id: number
  item_id: number
  item_name: string
  quantity: number
  probability: number
  is_choice: number
  choice_ids: string | null
  sort_order: number
}

// ---- 抽奖请求 ----
export interface DrawRequest {
  pool_type: number   // 1=白昼许愿, 2=星夜许愿
  count: number       // 1/10/30
}

// ---- 奖励项 ----
export interface RewardItem {
  reward_record_id?: number
  item_id: number
  item_name: string
  quantity: number
  is_choice: boolean
  choice_ids: number[] | null
  pool_type: number
}

export interface DrawPendingChoice {
  reward_record_id: number
  item_id: number
  item_name: string
  quantity: number
  choice_ids: number[]
}

export interface ConfirmDrawChoiceResult {
  reward_record_id: number
  item_id: number
  item_name: string
  quantity: number
}

// ---- 抽奖响应 ----
export interface DrawResponse {
  draw_log_id: number
  rewards: RewardItem[]
  lit_flowers: number
  actual_count: number
  requested_count: number
  refund_petals: number
  round_status: number
  lit_count_after: number
}

// ---- 查询所有活跃奖池 ----
export function fetchPools(): Promise<PoolConfig[]> {
  return request.get('/lottery/pools').then((res) => res.data)
}

// ---- 执行抽奖 ----
export function draw(params: DrawRequest): Promise<DrawResponse> {
  return withIdempotency(idempotencyOperations.lotteryDraw, params, (idempotencyKey) =>
    request.post('/lottery/draw', params, {
      headers: { 'Idempotency-Key': idempotencyKey },
    }).then((res) => res.data),
  )
}

export function fetchPendingDrawChoices(): Promise<DrawPendingChoice[]> {
  return request.get('/lottery/draw/choices/pending').then((res) => res.data)
}

export function confirmDrawChoice(
  rewardRecordId: number,
  selectedItemId: number,
): Promise<ConfirmDrawChoiceResult> {
  return request.post('/lottery/draw/choices/confirm', {
    reward_record_id: rewardRecordId,
    selected_item_id: selectedItemId,
  }).then((res) => res.data)
}
