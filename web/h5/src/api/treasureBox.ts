import request from './request'

export type BoxFlowState = 'none' | 'ready' | 'result' | 'choice'

export interface BoxReward {
  box_id: number
  chance_index: number
  item_id: number
  item_name: string
  is_choice: boolean
  choice_ids?: number[]
}

export interface OpenedBoxReward {
  chance_index: number
  item_id: number
  item_name: string
  is_choice: boolean
}

export interface BoxStatusResponse {
  round_number: number
  box_chances: number
  box_opened: number
  flow_state: BoxFlowState
  next_chance_index?: number
  pending_reward?: BoxReward
  opened_rewards: OpenedBoxReward[]
}

export interface ConfirmBoxResponse {
  chance_index: number
  item_id: number
  item_name: string
  is_choice: boolean
}

export function fetchBoxStatus(): Promise<BoxStatusResponse> {
  return request.get('/box/status').then((res) => res.data)
}

export function openBox(chanceIndex: number): Promise<BoxReward> {
  return request.post('/box/open', { chance_index: chanceIndex }).then((res) => res.data)
}

export function confirmBox(chanceIndex: number, selectedItemId = 0): Promise<ConfirmBoxResponse> {
  return request.post('/box/confirm', {
    chance_index: chanceIndex,
    selected_item_id: selectedItemId,
  }).then((res) => res.data)
}
