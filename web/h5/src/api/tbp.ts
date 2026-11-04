import request from './request'
import { activityEditionId, activityIsActive, ensureActivity, loadState } from './activitySession'
import type { BackendPrize } from './rewards'

export interface TryBeforePaySession {
  session_id: string
  status: number
  total_cost: number
  created_at: string
}

export interface TryBeforePayStatus {
  available: boolean
  session?: TryBeforePaySession
  preview?: TryBeforePayPreview
}

export interface TryBeforePayRewardSummary {
  item_id: number
  item_name: string
  total_quantity: number
  is_choice: boolean
}

export interface TryBeforePayPreview {
  session_id: string
  total_draws: number
  total_cost: number
  reward_summary: TryBeforePayRewardSummary[]
  estimated_lit_flowers: number
  estimated_box_chances: number
}

export interface TryBeforePayConfirmResult {
  session_id: string
  total_cost: number
  draw_log_id: string
}

interface BackendStep {
  prize: BackendPrize
  flowers_before: number
  flowers_after: number
  probability_lit: number
}

interface BackendPreview {
  preview_id: string
  status: string
  result_json: BackendStep[]
  required_petals: number
}

interface BackendPreviewResponse {
  preview_id: string
  results: BackendStep[]
  required_petals: number
}

interface BackendDraw {
  draw_id: string
  consumed_petals: number
}

let lastPreviewId: string | null = null

export function resetTryBeforePayCache(): void {
  lastPreviewId = null
}

function summarize(results: BackendStep[]): TryBeforePayRewardSummary[] {
  const byItem = new Map<string, TryBeforePayRewardSummary>()
  for (const step of results) {
    const key = step.prize.item_id
    const existing = byItem.get(key)
    if (existing) {
      existing.total_quantity += step.prize.quantity
    } else {
      byItem.set(key, {
        item_id: Number(step.prize.item_id),
        item_name: step.prize.name,
        total_quantity: step.prize.quantity,
        is_choice: (step.prize.choices?.length ?? 0) > 0,
      })
    }
  }
  return [...byItem.values()]
}

function toPreview(id: string, results: BackendStep[], cost: number, initialFlowers: number): TryBeforePayPreview {
  const lit = results.reduce((sum, step) => sum + Math.max(0, step.flowers_after - step.flowers_before), 0)
  const finalFlowers = results.length > 0 ? results[results.length - 1].flowers_after : initialFlowers
  lastPreviewId = id
  return {
    session_id: id,
    total_draws: results.length,
    total_cost: cost,
    reward_summary: summarize(results),
    estimated_lit_flowers: lit,
    estimated_box_chances: Math.max(0, Math.floor(finalFlowers / 6) - Math.floor(initialFlowers / 6)),
  }
}

async function previewRows(): Promise<BackendPreview[]> {
  await ensureActivity()
  return request.get('/activity/v1/records/preview', {
    params: { edition_id: activityEditionId() },
  }).then((res) => res.data as BackendPreview[])
}

export async function fetchTryBeforePayStatus(): Promise<TryBeforePayStatus> {
  await ensureActivity()
  if (!activityIsActive()) return { available: false }
  const state = await loadState()
  const rows = await previewRows()
  const pending = rows.find((row) => row.status === 'PENDING')
  if (pending) {
    return {
      available: true,
      session: {
        session_id: pending.preview_id,
        status: 1,
        total_cost: pending.required_petals,
        created_at: new Date().toISOString(),
      },
      preview: toPreview(pending.preview_id, pending.result_json, pending.required_petals, state.flowers),
    }
  }
  return { available: state.formalDrawCount === 0, session: undefined, preview: undefined }
}

export async function previewTryBeforePay(): Promise<TryBeforePayPreview> {
  const state = await loadState()
  await ensureActivity()
  const data = await request.post('/activity/v1/preview', {
    edition_id: activityEditionId(),
  }).then((res) => res.data as BackendPreviewResponse)
  return toPreview(data.preview_id, data.results, data.required_petals, state.flowers)
}

export async function confirmTryBeforePay(): Promise<TryBeforePayConfirmResult> {
  await ensureActivity()
  const session = lastPreviewId ?? (await previewRows()).find((row) => row.status === 'PENDING')?.preview_id
  if (!session) throw new Error('先抽后付预览不存在')
  const data = await request.post('/activity/v1/preview-confirm', {
    edition_id: activityEditionId(),
  }).then((res) => res.data as BackendDraw)
  lastPreviewId = null
  return {
    session_id: session,
    total_cost: data.consumed_petals,
    draw_log_id: data.draw_id,
  }
}

export async function abandonTryBeforePay(): Promise<void> {
  await ensureActivity()
  await request.post('/activity/v1/preview-abandon', {
    edition_id: activityEditionId(),
  })
}
