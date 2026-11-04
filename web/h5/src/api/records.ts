import request from './request'
import { activityEditionId, ensureActivity } from './activitySession'
import type { DrawResponse, RewardItem } from './lottery'
import { fetchAllRewards, isRingReward } from './rewards'

export interface LitLogItem {
  id: string
  pool_type: number
  lit_count: number
  created_at: string
}

export interface LitLogsResponse {
  list: LitLogItem[]
  total: number
  page: number
}

export interface RingRecordItem {
  id: string
  item_name: string
  source: string
  created_at: string
}

export interface RingRecordsResponse {
  list: RingRecordItem[]
  total: number
  page: number
}

export interface DrawHistoryItem {
  draw_log_id: string
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

interface BackendStep {
  prize: {
    item_id: string
    name: string
    quantity: number
    choices?: Array<{ item_id: string; name: string }>
  }
}

interface BackendDraw {
  draw_id: string
  pool_code: string
  mode: string
  requested_count: number
  actual_count: number
  consumed_petals: number
  refunded_petals: number
  flowers_added: number
  reward_snapshot_json: BackendStep[] | string
  occurred_at: string
}

interface BackendFlower {
  log_id: string
  draw_id: string
  draw_no: number
  pool_code: string
  flowers_added: number
  flowers_after: number
  created_at: string
}

const PAGE_SIZE = 20
let drawsCache: { at: number; items: DrawHistoryItem[] } | null = null
let ringsCache: { at: number; items: RingRecordItem[] } | null = null
let flowersCache: { at: number; items: LitLogItem[] } | null = null
const CACHE_MS = 3000

async function allPages<T>(kind: string): Promise<T[]> {
  await ensureActivity()
  const edition = activityEditionId()
  const out: T[] = []
  let before = ''
  for (let page = 0; page < 100; page += 1) {
    const rows = await request.get(`/activity/v1/records/${kind}`, {
      params: { edition_id: edition, before: before || undefined },
    }).then((res) => res.data as T[])
    out.push(...rows)
    if (rows.length < PAGE_SIZE) break
    const last = rows[rows.length - 1] as { draw_id?: string; log_id?: string; grant_id?: string; chest_id?: string; preview_id?: string }
    before = last.draw_id ?? last.log_id ?? last.grant_id ?? ''
    if (!before) break
  }
  return out
}

function poolType(code: string): number {
  return code === 'NIGHT' ? 2 : 1
}

function drawRewards(raw: BackendDraw['reward_snapshot_json']): RewardItem[] {
  let steps: BackendStep[] = []
  if (typeof raw === 'string') {
    try {
      steps = JSON.parse(raw) as BackendStep[]
    } catch {
      steps = []
    }
  } else {
    steps = raw ?? []
  }
  return steps.map((step) => {
    const choices = step.prize.choices ?? []
    return {
      item_id: Number(step.prize.item_id),
      item_name: step.prize.name,
      quantity: step.prize.quantity,
      is_choice: choices.length > 0,
      choice_ids: choices.length > 0 ? choices.map((choice) => Number(choice.item_id)) : null,
      pool_type: 0,
    }
  })
}

function toDrawHistory(item: BackendDraw): DrawHistoryItem {
  const type = poolType(item.pool_code)
  return {
    draw_log_id: item.draw_id,
    pool_type: type,
    pool_name: type === 2 ? '星夜花园' : '白昼花园',
    draw_mode: item.mode === 'PREVIEW' ? 1 : 0,
    requested_count: item.requested_count,
    actual_count: item.actual_count,
    cost_petals: item.consumed_petals,
    refund_petals: item.refunded_petals,
    lit_flowers: item.flowers_added,
    rewards: drawRewards(item.reward_snapshot_json).map((reward) => ({ ...reward, pool_type: type })),
    created_at: item.occurred_at,
  }
}

async function allDraws(): Promise<DrawHistoryItem[]> {
  if (drawsCache && Date.now() - drawsCache.at < CACHE_MS) return drawsCache.items
  const rows = await allPages<BackendDraw>('draws')
  const items = rows.map(toDrawHistory)
  drawsCache = { at: Date.now(), items }
  return items
}

async function allRings(): Promise<RingRecordItem[]> {
  if (ringsCache && Date.now() - ringsCache.at < CACHE_MS) return ringsCache.items
  const rewards = await fetchAllRewards()
  const items = rewards
    .filter(isRingReward)
    .map((reward) => ({
      id: reward.grant_id,
      item_name: reward.final_item_name ?? reward.original_item_name,
      source: reward.source_type,
      created_at: reward.created_at,
    }))
  ringsCache = { at: Date.now(), items }
  return items
}

async function allFlowers(): Promise<LitLogItem[]> {
  if (flowersCache && Date.now() - flowersCache.at < CACHE_MS) return flowersCache.items
  const rows = await allPages<BackendFlower>('flowers')
  const items = rows.map((row) => ({
    id: row.log_id,
    pool_type: poolType(row.pool_code),
    lit_count: row.flowers_added,
    created_at: row.created_at,
  }))
  flowersCache = { at: Date.now(), items }
  return items
}

function slice<T>(items: T[], page: number): T[] {
  return items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
}

export async function fetchLitLogs(page = 1): Promise<LitLogsResponse> {
  const items = await allFlowers()
  return { list: slice(items, page), total: items.length, page }
}

export async function fetchRingRecords(page = 1): Promise<RingRecordsResponse> {
  const items = await allRings()
  return { list: slice(items, page), total: items.length, page }
}

export async function fetchDrawHistory(page = 1): Promise<DrawHistoryResponse> {
  const items = await allDraws()
  return {
    total: items.length,
    page,
    page_size: PAGE_SIZE,
    list: slice(items, page),
  }
}

// Called after a draw/selection so the next popup sees fresh facts.
export function clearRecordCaches(): void {
  drawsCache = null
  ringsCache = null
  flowersCache = null
}
