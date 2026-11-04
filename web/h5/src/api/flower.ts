import request from './request'
import { activityEditionId, ensureActivity, loadState } from './activitySession'

export interface FlowerProgressResponse {
  round_number: number
  lit_count: number
  cumulative_gold_spent: number
  box_chances: number
  box_opened: number
  stage_claimed: number[]
  round_status: number
  max_flowers: number
}

const STAGES = [3, 5, 8, 11, 15, 17]

export async function startNewRound(): Promise<FlowerProgressResponse> {
  await ensureActivity()
  const state = await loadState()
  await request.post('/activity/v1/next-round', {
    edition_id: activityEditionId(),
    from_round_no: state.roundNo,
  })
  const next = await loadState()
  return {
    round_number: next.roundNo,
    lit_count: next.flowers,
    cumulative_gold_spent: 0,
    box_chances: next.unlockedChests,
    box_opened: next.completedChests,
    stage_claimed: STAGES.filter((_, index) => (next.claimedStageMask & (1 << index)) !== 0),
    round_status: next.status === 'FULL' ? 2 : next.status === 'COMPLETED' ? 3 : 1,
    max_flowers: 18,
  }
}
