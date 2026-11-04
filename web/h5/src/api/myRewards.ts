import request from './request'
import { activityEditionId, ensureActivity } from './activitySession'

export interface MyRewardItem {
  item_id: number
  item_name: string
  quantity: number
}

export interface MyRewardsResponse {
  list: MyRewardItem[]
}

interface BackendSummary {
  item_id: string
  final_item_name: string
  quantity: number
}

export async function fetchMyRewards(): Promise<MyRewardsResponse> {
  await ensureActivity()
  const rows = await request.get('/activity/v1/records/summary', {
    params: { edition_id: activityEditionId() },
  }).then((res) => res.data as BackendSummary[])
  return {
    list: rows.map((row) => ({
      item_id: Number(row.item_id),
      item_name: row.final_item_name,
      quantity: row.quantity,
    })),
  }
}
