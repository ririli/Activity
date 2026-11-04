import request from './request'
import { activityEditionId, activityRules, ensureActivity, loadState, type Prize } from './activitySession'
import { choiceIDs, fetchAllRewards, type BackendReward } from './rewards'

export interface StageRewardItem {
  slot_index: number
  lit_threshold: number
  item_id: number
  item_name: string
  quantity: number
  is_choice: boolean
  choice_ids: string
  claimed: boolean
  reachable: boolean
}

export interface StageRewardsResponse {
  rewards: StageRewardItem[]
  current_lit_count: number
  pending_choice?: StageRewardChoice
}

export interface StageRewardChoice {
  reward_record_id: string
  slot_index: number
  item_id: number
  item_name: string
  quantity: number
  choice_ids: number[]
}

export interface ClaimStageRewardParams {
  slot_index: number
}

export interface ClaimStageRewardResult {
  slot_index: number
  item_id: number
  item_name: string
  quantity: number
  is_choice: boolean
  pending_choice?: StageRewardChoice
}

export interface ConfirmStageRewardChoiceResult extends ClaimStageRewardResult {}

const STAGE_CODES = [3, 5, 8, 11, 15, 17] as const
const choiceSlots = new Map<string, number>()

export function resetStageChoiceCache(): void {
  choiceSlots.clear()
}

function stagePrize(stage: number): Prize | null {
  const rules = activityRules()
  return rules?.stages?.[String(stage)] ?? null
}

function stageIndexForReward(reward: BackendReward): number {
  const index = STAGE_CODES.findIndex((stage) => stagePrize(stage)?.code === reward.original_reward_code)
  if (index >= 0) return index + 1
  const numeric = Number(reward.original_reward_code)
  const byThreshold = STAGE_CODES.indexOf(numeric as typeof STAGE_CODES[number])
  return byThreshold >= 0 ? byThreshold + 1 : 0
}

function toChoice(reward: BackendReward): StageRewardChoice | null {
  const prize = reward.candidate_snapshot?.[0]
  if (!prize) return null
  const slot = stageIndexForReward(reward)
  choiceSlots.set(reward.grant_id, slot)
  return {
    reward_record_id: reward.grant_id,
    slot_index: slot,
    item_id: Number(prize.item_id),
    item_name: prize.name,
    quantity: prize.quantity,
    choice_ids: choiceIDs(prize),
  }
}

export async function fetchStageRewards(): Promise<StageRewardsResponse> {
  await ensureActivity()
  const [state, rewards] = await Promise.all([
    loadState(),
    fetchAllRewards('PENDING_CHOICE'),
  ])
  const stageRewards = rewards
    .filter((reward) => reward.source_type === 'STAGE')
    .map(toChoice)
    .filter((choice): choice is StageRewardChoice => choice !== null)
  const pending = stageRewards[0]
  const rewardsList: StageRewardItem[] = STAGE_CODES.map((stage, index) => {
    const prize = stagePrize(stage)
    const choices = prize?.choices ?? []
    return {
      slot_index: index + 1,
      lit_threshold: stage,
      item_id: Number(prize?.item_id ?? 0),
      item_name: prize?.name ?? '',
      quantity: prize?.quantity ?? 1,
      is_choice: choices.length > 0,
      choice_ids: choices.map((choice) => choice.item_id).join(','),
      claimed: (state.claimedStageMask & (1 << index)) !== 0,
      reachable: state.flowers >= stage,
    }
  })
  return {
    rewards: rewardsList,
    current_lit_count: state.flowers,
    pending_choice: pending,
  }
}

export async function claimStageReward(slotIndex: number): Promise<ClaimStageRewardResult> {
  await ensureActivity()
  const stage = STAGE_CODES[slotIndex - 1]
  if (!stage) throw new Error('阶段奖励档位无效')
  const response = await request.post('/activity/v1/stage', {
    edition_id: activityEditionId(),
    stage,
  }).then((res) => res.data as { grant_id: string })
  const prize = stagePrize(stage)
  const choices = prize?.choices ?? []
  let pending: StageRewardChoice | undefined
  if (choices.length > 0) {
    pending = {
      reward_record_id: response.grant_id,
      slot_index: slotIndex,
      item_id: Number(choices[0].item_id),
      item_name: choices[0].name,
      quantity: choices[0].quantity,
      choice_ids: choices.map((choice) => Number(choice.item_id)),
    }
    choiceSlots.set(response.grant_id, slotIndex)
  }
  return {
    slot_index: slotIndex,
    item_id: Number(prize?.item_id ?? 0),
    item_name: prize?.name ?? '',
    quantity: prize?.quantity ?? 1,
    is_choice: choices.length > 0,
    pending_choice: pending,
  }
}

export async function confirmStageRewardChoice(
  rewardRecordId: string,
  selectedItemId: number,
): Promise<ConfirmStageRewardChoiceResult> {
  await ensureActivity()
  const data = await request.post('/activity/v1/select', {
    edition_id: activityEditionId(),
    id: rewardRecordId,
    item_id: String(selectedItemId),
  }).then((res) => res.data as { grant_id: string; item_id: string; item_name?: string; quantity?: number })
  const slot = choiceSlots.get(rewardRecordId) ?? 0
  return {
    slot_index: slot,
    item_id: Number(data.item_id),
    item_name: data.item_name ?? '',
    quantity: data.quantity ?? 1,
    is_choice: true,
  }
}
