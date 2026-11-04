import request from './request'

// ---- 排行榜单项（后台 JSON tag: top_20） ----
export interface RankingItem {
  rank: number
  user_id: number
  nickname: string
  score: number
  achieved_at: string
}

// ---- 当前用户排名信息（后台 JSON tag: current_user） ----
export interface UserRank {
  rank: number
  score: number
  achieved_at: string
}

// ---- 排行榜接口响应 ----
export interface RankingResponse {
  top_20: RankingItem[]
  current_user: UserRank | null
}

// ---- 获取排行榜（Top20 + 当前用户排名） ----
export function fetchRanking(): Promise<RankingResponse> {
  return request.get('/ranking').then((res) => res.data)
}
