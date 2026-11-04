import request from './request'

export interface FlowerProgressResponse {
  round_number: number
  lit_count: number
  cumulative_gold_spent: number
  box_chances: number
  box_opened: number
  stage_claimed: number[]
  round_status: number
  max_flowers: number
}

export function startNewRound(): Promise<FlowerProgressResponse> {
  return request.post('/flower/new-round').then((res) => res.data)
}
