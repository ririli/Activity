import request from './request'
import { idempotencyOperations, withIdempotency } from './idempotency'
import { activityEditionId, activityRules, ensureActivity, type Prize } from './activitySession'
import { choiceIDs, fetchAllRewards, type BackendPrize, type BackendReward } from './rewards'

export interface PoolConfig {
  pool_type: number
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

export interface DrawRequest {
  pool_type: number
  count: number
}

export interface RewardItem {
  reward_record_id?: string
  item_id: number
  item_name: string
  quantity: number
  is_choice: boolean
  choice_ids: number[] | null
  pool_type: number
}

export interface DrawPendingChoice {
  reward_record_id: string
  item_id: number
  item_name: string
  quantity: number
  choice_ids: number[]
}

export interface ConfirmDrawChoiceResult {
  reward_record_id: string
  item_id: number
  item_name: string
  quantity: number
}

export interface DrawResponse {
  draw_log_id: string
  rewards: RewardItem[]
  lit_flowers: number
  actual_count: number
  requested_count: number
  refund_petals: number
  round_status: number
  lit_count_after: number
}

interface BackendStep {
  prize: BackendPrize
  flowers_before: number
  flowers_after: number
  probability_lit: number
  gold_value: number
  grant_id?: string
}

interface BackendDraw {
  draw_id: string
  results: BackendStep[]
  consumed_petals: number
  refunded_petals: number
  requested_count: number
  actual_count: number
  flowers: number
}

function poolCode(poolType: number): 'DAY' | 'NIGHT' {
  return poolType === 2 ? 'NIGHT' : 'DAY'
}

function poolType(code: string): number {
  return code === 'NIGHT' ? 2 : 1
}

function poolName(type: number): string {
  return type === 2 ? '星夜花园' : '白昼花园'
}

function toPoolPrize(prize: Prize, index: number, type: number): PoolPrize {
  const choices = prize.choices ?? []
  return {
    id: index + 1,
    pool_id: type,
    item_id: Number(prize.item_id),
    item_name: prize.name,
    quantity: prize.quantity,
    probability: prize.weight ?? 0,
    is_choice: choices.length > 0 ? 1 : 0,
    choice_ids: choices.length > 0 ? choices.map((choice) => choice.item_id).join(',') : null,
    sort_order: index + 1,
  }
}

export async function fetchPools(): Promise<PoolConfig[]> {
  await ensureActivity()
  const rules = activityRules()
  if (!rules) return []
  return Object.entries(rules.pools)
    .map(([code, pool]) => {
      const type = poolType(code)
      return {
        pool_type: type,
        pool_name: poolName(type),
        cost_per_draw: pool.cost,
        status: 1,
        prizes: pool.prizes.map((prize, index) => toPoolPrize(prize, index, type)),
      }
    })
    .sort((a, b) => a.pool_type - b.pool_type)
}

function toReward(step: BackendStep, type: number): RewardItem {
  const choices = step.prize.choices ?? []
  return {
    reward_record_id: step.grant_id,
    item_id: Number(step.prize.item_id),
    item_name: step.prize.name,
    quantity: step.prize.quantity,
    is_choice: choices.length > 0,
    choice_ids: choices.length > 0 ? choiceIDs(step.prize) : null,
    pool_type: type,
  }
}

export function draw(params: DrawRequest): Promise<DrawResponse> {
  return withIdempotency(idempotencyOperations.lotteryDraw, params, (operationId) =>
    ensureActivity()
      .then(() => request.post('/activity/v1/draw', {
        edition_id: activityEditionId(),
        operation_id: operationId,
        pool_code: poolCode(params.pool_type),
        count: params.count,
      }))
      .then((res) => {
        const data = res.data as BackendDraw
        const results = data.results ?? []
        const lit = results.reduce((sum, item) => sum + Math.max(0, item.flowers_after - item.flowers_before), 0)
        return {
          draw_log_id: data.draw_id,
          rewards: results.map((item) => toReward(item, params.pool_type)),
          lit_flowers: lit,
          actual_count: data.actual_count,
          requested_count: data.requested_count,
          refund_petals: data.refunded_petals,
          round_status: data.flowers >= 18 ? 2 : 1,
          lit_count_after: data.flowers,
        }
      }),
  )
}

function pendingFromReward(reward: BackendReward): DrawPendingChoice | null {
  if (!reward.candidate_snapshot?.length) return null
  const prize = reward.candidate_snapshot[0]
  return {
    reward_record_id: reward.grant_id,
    item_id: Number(prize.item_id),
    item_name: prize.name,
    quantity: prize.quantity,
    choice_ids: choiceIDs(prize),
  }
}

export async function fetchPendingDrawChoices(): Promise<DrawPendingChoice[]> {
  const rewards = await fetchAllRewards('PENDING_CHOICE')
  return rewards
    .filter((reward) => ['DAY_DRAW', 'NIGHT_DRAW', 'PREVIEW'].includes(reward.source_type))
    .map(pendingFromReward)
    .filter((choice): choice is DrawPendingChoice => choice !== null)
}

export async function confirmDrawChoice(
  rewardRecordId: string,
  selectedItemId: number,
): Promise<ConfirmDrawChoiceResult> {
  await ensureActivity()
  const data = await request.post('/activity/v1/select', {
    edition_id: activityEditionId(),
    id: rewardRecordId,
    item_id: String(selectedItemId),
  }).then((res) => res.data as { grant_id: string; item_id: string; item_name?: string; quantity?: number })
  return {
    reward_record_id: data.grant_id,
    item_id: Number(data.item_id),
    item_name: data.item_name ?? '',
    quantity: data.quantity ?? 1,
  }
}
