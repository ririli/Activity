import request from './request'

export interface MyRewardItem {
  item_id: number
  item_name: string
  quantity: number
}

export interface MyRewardsResponse {
  list: MyRewardItem[]
}

export function fetchMyRewards(): Promise<MyRewardsResponse> {
  return request.get('/rewards/mine').then((res) => res.data)
}
