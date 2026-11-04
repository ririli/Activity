import request from './request'

// ---- 阶段奖励项（对应后端 StageRewardVO） ----
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

// ---- 阶段奖励列表响应 ----
export interface StageRewardsResponse {
  rewards: StageRewardItem[]
  current_lit_count: number
  pending_choice?: StageRewardChoice
}

export interface StageRewardChoice {
  reward_record_id: number
  slot_index: number
  item_id: number
  item_name: string
  quantity: number
  choice_ids: number[]
}

// ---- 领取阶段奖励请求 ----
export interface ClaimStageRewardParams {
  slot_index: number
}

// ---- 领取阶段奖励响应 ----
export interface ClaimStageRewardResult {
  slot_index: number
  item_id: number
  item_name: string
  quantity: number
  is_choice: boolean
  pending_choice?: StageRewardChoice
}

export interface ConfirmStageRewardChoiceResult extends ClaimStageRewardResult {}

// ---- 获取全量阶段奖励状态 ----
export function fetchStageRewards(): Promise<StageRewardsResponse> {
  return request.get('/rewards/stage').then((res) => res.data)
}

// ---- 领取指定档位阶段奖励 ----
export function claimStageReward(slotIndex: number): Promise<ClaimStageRewardResult> {
  return request.post('/rewards/stage/claim', { slot_index: slotIndex }).then((res) => res.data)
}

export function confirmStageRewardChoice(
  rewardRecordId: number,
  selectedItemId: number,
): Promise<ConfirmStageRewardChoiceResult> {
  return request.post('/rewards/stage/confirm', {
    reward_record_id: rewardRecordId,
    selected_item_id: selectedItemId,
  }).then((res) => res.data)
}
