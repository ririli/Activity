import request from './request'

export interface RingMarqueeItem {
  id: number
  nickname: string
  item_id: number
  item_name: string
  obtained_at: string
}

export interface ActivityConfig {
  activity_key: string
  activity_period_id: number
  period_no: number
  status: number
  status_text: string
  start_time: string
  end_time: string
  ended_at?: string
  ranking_rewarded_at?: string
}

export function fetchActivityConfig(): Promise<ActivityConfig> {
  return request.get('/activity/config').then((res) => res.data)
}
