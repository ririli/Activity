import request from './request'
import { activityEditionId, ensureActivity, loadState } from './activitySession'
import { choiceIDs, type BackendPrize } from './rewards'

export type BoxFlowState = 'none' | 'ready' | 'result' | 'choice'

export interface BoxReward {
  box_id: string
  grant_id?: string
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

interface BackendChest {
  chest_id: string
  round_no: number
  chest_no: number
  status: 'OPENED' | 'CONFIRMED'
  candidate_snapshot: BackendPrize
  grant_id: string
}

interface BackendOpen {
  chest_id: string
  grant_id: string
  reward: BackendPrize
}

function toBoxReward(chest: BackendChest): BoxReward {
  const prize = chest.candidate_snapshot
  const choices = prize?.choices ?? []
  return {
    box_id: chest.chest_id,
    grant_id: chest.grant_id,
    chance_index: chest.chest_no,
    item_id: Number(prize?.item_id ?? 0),
    item_name: prize?.name ?? '',
    is_choice: choices.length > 0,
    choice_ids: choices.length > 0 ? choiceIDs(prize) : undefined,
  }
}

async function chests(): Promise<BackendChest[]> {
  await ensureActivity()
  return request.get('/activity/v1/records/chests', {
    params: { edition_id: activityEditionId() },
  }).then((res) => res.data as BackendChest[])
}

export async function fetchBoxStatus(): Promise<BoxStatusResponse> {
  const [state, rows] = await Promise.all([loadState(), chests()])
  const currentRound = rows.filter((row) => row.round_no === state.roundNo)
  const opened = currentRound.filter((row) => row.status === 'OPENED')
  const confirmed = currentRound.filter((row) => row.status === 'CONFIRMED')
  const pending = opened[0]
  let flow: BoxFlowState = 'none'
  if (pending) flow = (pending.candidate_snapshot?.choices?.length ?? 0) > 0 ? 'choice' : 'result'
  else if (state.completedChests < state.unlockedChests) flow = 'ready'

  return {
    round_number: state.roundNo,
    box_chances: state.unlockedChests,
    box_opened: state.completedChests,
    flow_state: flow,
    next_chance_index: flow === 'ready' ? state.completedChests + 1 : undefined,
    pending_reward: pending ? toBoxReward(pending) : undefined,
    opened_rewards: confirmed.map((row) => ({
      chance_index: row.chest_no,
      item_id: Number(row.candidate_snapshot?.item_id ?? 0),
      item_name: row.candidate_snapshot?.name ?? '',
      is_choice: (row.candidate_snapshot?.choices?.length ?? 0) > 0,
    })),
  }
}

export async function openBox(_chanceIndex: number): Promise<BoxReward> {
  await ensureActivity()
  const data = await request.post('/activity/v1/chest-open', {
    edition_id: activityEditionId(),
  }).then((res) => res.data as BackendOpen)
  const choices = data.reward?.choices ?? []
  return {
    box_id: data.chest_id,
    grant_id: data.grant_id,
    chance_index: 0,
    item_id: Number(data.reward?.item_id ?? 0),
    item_name: data.reward?.name ?? '',
    is_choice: choices.length > 0,
    choice_ids: choices.length > 0 ? choiceIDs(data.reward) : undefined,
  }
}

export async function confirmBox(chanceIndex: number, selectedItemId = 0): Promise<ConfirmBoxResponse> {
  const status = await fetchBoxStatus()
  const pending = status.pending_reward
  if (!pending) throw new Error('没有待确认的宝箱')
  let selectedName = ''
  if (pending.is_choice && selectedItemId > 0) {
    if (!pending.grant_id) throw new Error('选择权奖励信息缺失')
    const selected = await request.post('/activity/v1/select', {
      edition_id: activityEditionId(),
      id: pending.grant_id,
      item_id: String(selectedItemId),
    }).then((res) => res.data as { item_name?: string })
    selectedName = selected.item_name ?? ''
  }
  await request.post('/activity/v1/chest-confirm', {
    edition_id: activityEditionId(),
    id: pending.box_id,
  })
  const selected = pending.choice_ids?.includes(selectedItemId) ? selectedItemId : undefined
  return {
    chance_index: chanceIndex || pending.chance_index,
    item_id: selected ?? pending.item_id,
    item_name: selectedName || pending.item_name,
    is_choice: pending.is_choice,
  }
}
