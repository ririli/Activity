import request from './request'
import { activityEditionId, ensureActivity } from './activitySession'

export interface RankingItem {
  rank: number
  user_id: string
  nickname: string
  score: number
  achieved_at: string
}

export interface UserRank {
  rank: number
  score: number
  achieved_at: string
}

export interface RankingResponse {
  top_20: RankingItem[]
  current_user: UserRank | null
}

interface BackendEntry {
  user_id: string
  nickname: string
  score: number
  score_reached_at_sec: number
  rank: number
}

interface BackendBoard {
  top: BackendEntry[]
  me: BackendEntry | null
  source: string
}

function achievedAt(seconds: number): string {
  if (!seconds) return ''
  return new Date(seconds * 1000).toISOString()
}

export async function fetchRanking(): Promise<RankingResponse> {
  await ensureActivity()
  const board = await request.get('/ranking/v1/board', {
    params: { edition_id: activityEditionId() },
  }).then((res) => res.data as BackendBoard)
  const current = board.me
  return {
    top_20: (board.top ?? []).map((entry) => ({
      rank: entry.rank,
      user_id: entry.user_id,
      nickname: entry.nickname,
      score: entry.score,
      achieved_at: achievedAt(entry.score_reached_at_sec),
    })),
    current_user: current
      ? { rank: current.rank, score: current.score, achieved_at: achievedAt(current.score_reached_at_sec) }
      : null,
  }
}
