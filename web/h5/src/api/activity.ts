import { ensureActivity, type ActivityConfig as BackendConfig } from './activitySession'

export interface RingMarqueeItem {
  id: string
  nickname: string
  item_id?: number
  item_name: string
  obtained_at: string
}

// Legacy view model retained so existing Home/Leaderboard code needs minimal
// changes. IDs that can exceed JS safe integers stay strings.
export interface ActivityConfig {
  activity_key: string
  activity_period_id: string
  period_no: number
  status: number
  status_text: string
  start_time: string
  end_time: string
  ended_at?: string
  ranking_rewarded_at?: string
}

function statusCode(config: BackendConfig): number {
  if (config.status === 'ENDED') return 2
  const now = Date.now()
  const start = Date.parse(config.starts_at)
  const end = Date.parse(config.ends_at)
  if (!Number.isNaN(start) && now < start) return 0
  if (!Number.isNaN(end) && now >= end) return 2
  return 1
}

export async function fetchActivityConfig(): Promise<ActivityConfig> {
  const config = await ensureActivity()
  const status = statusCode(config)
  const statusText = status === 2 ? '活动已结束' : status === 0 ? '活动尚未开始' : '活动进行中'
  return {
    activity_key: config.name,
    activity_period_id: config.edition_id,
    period_no: 1,
    status,
    status_text: statusText,
    start_time: config.starts_at,
    end_time: config.ends_at,
  }
}
