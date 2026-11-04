/** 进度节点状态 */
export type ProgressStatus = 'locked' | 'available' | 'claimed'

/** 阶段奖励图标类别 */
export type RewardIconKind = 'coin' | 'petal' | 'ring' | 'gift'

/** 许愿模式 */
export type WishMode = 'day' | 'night'

/** 进度奖励节点 */
export interface ProgressNode {
  id: number
  item_id: number
  name: string
  /** 点亮花朵数阈值 */
  lit_threshold: number
  /** 奖励图标类别 */
  icon: RewardIconKind
  status: ProgressStatus
}
