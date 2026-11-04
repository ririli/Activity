import request from './request'
import { activityEditionId, ensureActivity } from './activitySession'

export interface BackendPrize {
  code: string
  item_id: string
  name: string
  quantity: number
  weight?: number
  choices?: BackendPrize[]
}

export interface BackendReward {
  grant_id: string
  source_type: string
  source_id?: string
  source_entry_no?: number
  round_no?: number
  draw_id?: string
  original_reward_code: string
  original_item_name: string
  quantity: number
  status: 'OBTAINED' | 'PENDING_CHOICE' | 'SELECTED'
  final_item_id?: string
  final_item_name?: string
  candidate_snapshot?: BackendPrize[] | null
  created_at: string
}

// The records endpoint is cursor-based. Adapters that need "all pending
// choices" must page until the server returns fewer than 20 rows; otherwise an
// older pending choice can be hidden behind newer rewards.
export async function fetchAllRewards(status?: string): Promise<BackendReward[]> {
  await ensureActivity()
  const edition = activityEditionId()
  const out: BackendReward[] = []
  let before = ''
  for (let page = 0; page < 100; page += 1) {
    const rows = await request.get('/activity/v1/records/rewards', {
      params: { edition_id: edition, before: before || undefined, status },
    }).then((res) => res.data as BackendReward[])
    out.push(...rows)
    if (rows.length < 20) break
    before = rows[rows.length - 1]?.grant_id ?? ''
    if (!before) break
  }
  return out
}

export function choiceIDs(prize: BackendPrize | undefined): number[] {
  return (prize?.choices ?? [])
    .map((choice) => Number(choice.item_id))
    .filter((id) => Number.isSafeInteger(id) && id > 0)
}

export function isRingReward(reward: BackendReward): boolean {
  return (reward.final_item_name ?? reward.original_item_name).includes('戒指')
}
