import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import type { StageRewardChoice, StageRewardItem } from '@/api/stage'
import type { BoxFlowState, BoxReward, OpenedBoxReward } from '@/api/treasureBox'
import type { DrawPendingChoice } from '@/api/lottery'
import type { RingMarqueeItem } from '@/api/activity'

interface ApiState {
  activityStatus: number
  activityPeriodId: number
  roundNumber: number
  gold: number
  petals: number
  petalGiftPurchased: boolean
  petalGiftPurchaseCount: number
  purchasedPetalShopTiers: number[]
  currentLitCount: number
  rewards: StageRewardItem[]
  claimedSlot: number | null
  stagePendingChoice: StageRewardChoice | null
  stageConfirmedItemId: number | null
  drawPoolType: number | null
  drawCount: number | null
  drawRequestCount: number
  drawActualCount: number | null
  drawRefundPetals: number
  drawLitFlowers: number | null
  drawLitCountAfter: number | null
  drawChoiceOnNextDraw: boolean
  drawPendingChoices: DrawPendingChoice[]
  drawConfirmedItemIds: number[]
  tbpAvailable: boolean
  tbpActiveSession: boolean
  tbpPreviewCount: number
  tbpEstimatedLitFlowers: number
  tbpConfirmed: boolean
  tbpAbandoned: boolean
  tbpChoiceOnConfirm: boolean
  boxChances: number
  boxOpened: number
  boxFlowState: BoxFlowState
  boxPendingReward: BoxReward | null
  boxOpenedRewards: OpenedBoxReward[]
  boxNextRewardIsChoice: boolean
  boxOpenCount: number
  boxConfirmCount: number
  boxConfirmedItemId: number | null
  newRoundRequestCount: number
  newRoundStarted: boolean
  rankingRequestCount: number
  marqueeRequestCount: number
  marqueeItems: RingMarqueeItem[]
}

function createStageRewards(): StageRewardItem[] {
  return [
    {
      slot_index: 0,
      lit_threshold: 3,
      item_id: 1001,
      item_name: '金币',
      quantity: 500,
      is_choice: false,
      choice_ids: '',
      claimed: false,
      reachable: true,
    },
    {
      slot_index: 1,
      lit_threshold: 5,
      item_id: 1002,
      item_name: '花瓣',
      quantity: 10,
      is_choice: false,
      choice_ids: '',
      claimed: false,
      reachable: false,
    },
    {
      slot_index: 2,
      lit_threshold: 8,
      item_id: 1207244,
      item_name: '烟花之恋戒指',
      quantity: 1,
      is_choice: false,
      choice_ids: '',
      claimed: false,
      reachable: false,
    },
    {
      slot_index: 3,
      lit_threshold: 11,
      item_id: 1001,
      item_name: '金币',
      quantity: 2000,
      is_choice: false,
      choice_ids: '',
      claimed: false,
      reachable: false,
    },
    {
      slot_index: 4,
      lit_threshold: 15,
      item_id: 1205251,
      item_name: '月亮游记戒指',
      quantity: 1,
      is_choice: false,
      choice_ids: '',
      claimed: false,
      reachable: false,
    },
    {
      slot_index: 5,
      lit_threshold: 17,
      item_id: 1207751,
      item_name: '真爱无敌戒指选择权',
      quantity: 1,
      is_choice: true,
      choice_ids: '1207751,1207752,1207753',
      claimed: false,
      reachable: false,
    },
  ]
}

export const apiState: ApiState = {
  activityStatus: 1,
  activityPeriodId: 1,
  roundNumber: 1,
  gold: 12_000,
  petals: 300,
  petalGiftPurchased: false,
  petalGiftPurchaseCount: 0,
  purchasedPetalShopTiers: [],
  currentLitCount: 4,
  rewards: createStageRewards(),
  claimedSlot: null,
  stagePendingChoice: null,
  stageConfirmedItemId: null,
  drawPoolType: null,
  drawCount: null,
  drawRequestCount: 0,
  drawActualCount: null,
  drawRefundPetals: 0,
  drawLitFlowers: null,
  drawLitCountAfter: null,
  drawChoiceOnNextDraw: false,
  drawPendingChoices: [],
  drawConfirmedItemIds: [],
  tbpAvailable: true,
  tbpActiveSession: false,
  tbpPreviewCount: 0,
  tbpEstimatedLitFlowers: 6,
  tbpConfirmed: false,
  tbpAbandoned: false,
  tbpChoiceOnConfirm: false,
  boxChances: 0,
  boxOpened: 0,
  boxFlowState: 'none',
  boxPendingReward: null,
  boxOpenedRewards: [],
  boxNextRewardIsChoice: false,
  boxOpenCount: 0,
  boxConfirmCount: 0,
  boxConfirmedItemId: null,
  newRoundRequestCount: 0,
  newRoundStarted: false,
  rankingRequestCount: 0,
  marqueeRequestCount: 0,
  marqueeItems: [],
}

export function resetApiState() {
  apiState.activityStatus = 1
  apiState.activityPeriodId = 1
  apiState.roundNumber = 1
  apiState.gold = 12_000
  apiState.petals = 300
  apiState.petalGiftPurchased = false
  apiState.petalGiftPurchaseCount = 0
  apiState.purchasedPetalShopTiers = []
  apiState.currentLitCount = 4
  apiState.rewards = createStageRewards()
  apiState.claimedSlot = null
  apiState.stagePendingChoice = null
  apiState.stageConfirmedItemId = null
  apiState.drawPoolType = null
  apiState.drawCount = null
  apiState.drawRequestCount = 0
  apiState.drawActualCount = null
  apiState.drawRefundPetals = 0
  apiState.drawLitFlowers = null
  apiState.drawLitCountAfter = null
  apiState.drawChoiceOnNextDraw = false
  apiState.drawPendingChoices = []
  apiState.drawConfirmedItemIds = []
  apiState.tbpAvailable = true
  apiState.tbpActiveSession = false
  apiState.tbpPreviewCount = 0
  apiState.tbpEstimatedLitFlowers = 6
  apiState.tbpConfirmed = false
  apiState.tbpAbandoned = false
  apiState.tbpChoiceOnConfirm = false
  apiState.boxChances = 0
  apiState.boxOpened = 0
  apiState.boxFlowState = 'none'
  apiState.boxPendingReward = null
  apiState.boxOpenedRewards = []
  apiState.boxNextRewardIsChoice = false
  apiState.boxOpenCount = 0
  apiState.boxConfirmCount = 0
  apiState.boxConfirmedItemId = null
  apiState.newRoundRequestCount = 0
  apiState.newRoundStarted = false
  apiState.rankingRequestCount = 0
  apiState.marqueeRequestCount = 0
  apiState.marqueeItems = []
}

function success(data: unknown) {
  return HttpResponse.json({ code: 0, message: 'ok', data })
}

function failure(code: number, message: string, status = 400) {
  return HttpResponse.json({ code, message, data: null }, { status })
}

const petalTierCosts: Record<number, number> = {
  5: 300,
  52: 3000,
  525: 30000,
  1580: 90000,
}
const TIER_AMOUNTS_FOR_TEST = [52, 525, 1580]

function getPetalCostGold(amount: number) {
  return petalTierCosts[amount] ?? amount * 60
}

function createTryBeforePayPreview() {
  return {
    session_id: 77,
    total_draws: 180,
    total_cost: 900,
    reward_summary: [
      { item_id: 1002, item_name: '花瓣', total_quantity: 80, is_choice: false },
      { item_id: 2001, item_name: '月亮游记戒指', total_quantity: 1, is_choice: true },
    ],
    estimated_lit_flowers: apiState.tbpEstimatedLitFlowers,
    estimated_box_chances: 1,
  }
}

const drawRewardCatalog = [
  { item_id: 1002, item_name: '花瓣', quantity: 1, is_choice: false, choice_ids: null },
  { item_id: 1001, item_name: '金币', quantity: 100, is_choice: false, choice_ids: null },
  { item_id: 1207244, item_name: '烟花之恋戒指', quantity: 1, is_choice: false, choice_ids: null },
  { item_id: 1203743, item_name: '闪闪心蝶戒指', quantity: 1, is_choice: false, choice_ids: null },
  { item_id: 1205251, item_name: '月亮游记戒指', quantity: 1, is_choice: false, choice_ids: null },
  { item_id: 3001, item_name: '88元暖心欢乐礼包', quantity: 1, is_choice: false, choice_ids: null },
]

const dayPoolPrizes = [
  { id: 1, pool_id: 1, item_id: 1001, item_name: '金币', quantity: 100, probability: 35, is_choice: 0, choice_ids: null, sort_order: 0 },
  { id: 2, pool_id: 1, item_id: 1001, item_name: '金币', quantity: 300, probability: 25, is_choice: 0, choice_ids: null, sort_order: 1 },
  { id: 3, pool_id: 1, item_id: 1002, item_name: '花瓣', quantity: 1, probability: 20, is_choice: 0, choice_ids: null, sort_order: 2 },
  { id: 4, pool_id: 1, item_id: 1207244, item_name: '烟花之恋戒指', quantity: 1, probability: 10, is_choice: 0, choice_ids: null, sort_order: 3 },
  { id: 5, pool_id: 1, item_id: 1203743, item_name: '闪闪心蝶戒指', quantity: 1, probability: 7, is_choice: 0, choice_ids: null, sort_order: 4 },
  { id: 6, pool_id: 1, item_id: 1205251, item_name: '月亮游记戒指', quantity: 1, probability: 2.5, is_choice: 0, choice_ids: null, sort_order: 5 },
  { id: 7, pool_id: 1, item_id: 1207751, item_name: '真爱无敌戒指选择权', quantity: 1, probability: 0.5, is_choice: 1, choice_ids: '1207751,1207752,1207753', sort_order: 6 },
]

const nightPoolPrizes = [
  { id: 8, pool_id: 2, item_id: 1001, item_name: '金币', quantity: 1000, probability: 25, is_choice: 0, choice_ids: null, sort_order: 0 },
  { id: 9, pool_id: 2, item_id: 1002, item_name: '花瓣', quantity: 10, probability: 20, is_choice: 0, choice_ids: null, sort_order: 1 },
  { id: 10, pool_id: 2, item_id: 1207244, item_name: '烟花之恋戒指', quantity: 1, probability: 20, is_choice: 0, choice_ids: null, sort_order: 2 },
  { id: 11, pool_id: 2, item_id: 1203743, item_name: '闪闪心蝶戒指', quantity: 1, probability: 15, is_choice: 0, choice_ids: null, sort_order: 3 },
  { id: 12, pool_id: 2, item_id: 1205251, item_name: '月亮游记戒指', quantity: 1, probability: 10, is_choice: 0, choice_ids: null, sort_order: 4 },
  { id: 13, pool_id: 2, item_id: 1205470, item_name: '爱意翩跹戒指', quantity: 1, probability: 7, is_choice: 0, choice_ids: null, sort_order: 5 },
  { id: 14, pool_id: 2, item_id: 1207751, item_name: '真爱无敌戒指选择权', quantity: 1, probability: 3, is_choice: 1, choice_ids: '1207751,1207752,1207753', sort_order: 6 },
]

const myRewardItems = [
  { item_id: 1207751, item_name: '真爱无敌戒指经典款', quantity: 2 },
  { item_id: 1207752, item_name: '真爱无敌戒指铭文款', quantity: 1 },
  { item_id: 1207753, item_name: '爱你万年戒指铭文款', quantity: 1 },
  { item_id: 1205251, item_name: '月亮游记戒指', quantity: 3 },
  { item_id: 1323, item_name: '定制称号/修改V8称号', quantity: 1 },
  { item_id: 3633, item_name: '春日公主称号', quantity: 1 },
  { item_id: 1207809, item_name: '春日公主头像框', quantity: 1 },
  { item_id: 1207811, item_name: '春日公主冠名头像框', quantity: 1 },
  { item_id: 1001, item_name: '金币', quantity: 500 },
]

function createDrawRewards(count: number, poolType: number) {
  return Array.from({ length: count }, (_, index) => ({
    ...drawRewardCatalog[index % drawRewardCatalog.length],
    pool_type: poolType,
  }))
}

const drawHistoryItems = [
  {
    draw_log_id: 102,
    pool_type: 1,
    pool_name: '白昼花园',
    draw_mode: 2,
    requested_count: 10,
    actual_count: 2,
    cost_petals: 10,
    refund_petals: 0,
    lit_flowers: 2,
    rewards: [
      { item_id: 3001, item_name: '88元暖心欢乐礼包', quantity: 1, is_choice: false, choice_ids: null, pool_type: 1 },
      { item_id: 3002, item_name: '告白气球语音派对背景1天', quantity: 1, is_choice: false, choice_ids: null, pool_type: 1 },
    ],
    created_at: '2026-12-24T20:00:00+08:00',
  },
  {
    draw_log_id: 101,
    pool_type: 2,
    pool_name: '星夜花园',
    draw_mode: 1,
    requested_count: 1,
    actual_count: 1,
    cost_petals: 100,
    refund_petals: 0,
    lit_flowers: 0,
    rewards: [
      { item_id: 4001, item_name: '月亮游记戒指', quantity: 1, is_choice: false, choice_ids: null, pool_type: 2 },
    ],
    created_at: '2026-12-23T19:30:00+08:00',
  },
  {
    draw_log_id: 100,
    pool_type: 1,
    pool_name: '白昼花园',
    draw_mode: 1,
    requested_count: 1,
    actual_count: 1,
    cost_petals: 5,
    refund_petals: 0,
    lit_flowers: 0,
    rewards: [
      { item_id: 1002, item_name: '花瓣', quantity: 1, is_choice: false, choice_ids: null, pool_type: 1 },
    ],
    created_at: '2026-12-22T18:20:00+08:00',
  },
]

const ringRecordItems = [
  {
    id: 202,
    item_name: '月亮游记戒指',
    source: 'treasure_box',
    created_at: '2026-12-24T20:00:00+08:00',
  },
  {
    id: 201,
    item_name: '真爱无敌戒指选择权',
    source: 'treasure_box',
    created_at: '2026-12-23T19:30:00+08:00',
  },
]

const rankingNicknames = [
  '月下花语',
  '春风十里',
  '星河入梦',
  '云端漫步',
  '花间听雨',
  '晚风来信',
  '银河旅人',
  '长街灯火',
  '山海有期',
  '晨雾微光',
  '蔷薇心事',
  '月桂来客',
  '夜航星',
  '花园访客',
  '春日邮差',
  '星屿漫游',
  '玫瑰信笺',
  '白昼梦境',
  '月色回声',
  '许愿旅人',
  '超额用户21',
  '超额用户22',
]

const rankingItems = rankingNicknames.map((nickname, index) => ({
  rank: index + 1,
  user_id: index + 101,
  nickname,
  score: Math.max(99, 999_999 - index * 37_421),
  achieved_at: '2026-07-10T12:00:00Z',
}))

// ---------------------------------------------------------------------------
// New-contract fixtures. The H5 adapters consume the real backend shapes, so
// MSW returns those shapes directly (no {code,message,data} envelope).
// ---------------------------------------------------------------------------

const STAGE_CODES = [3, 5, 8, 11, 15, 17]
const RING_CHOICES = [
  { item_id: '1207751', name: '真爱无敌戒指经典款' },
  { item_id: '1207752', name: '真爱无敌戒指铭文款' },
  { item_id: '1207753', name: '爱你万年戒指铭文款' },
]

function bPrize(itemId: number, name: string, quantity = 1, choices?: Array<{ item_id: string; name: string }>) {
  return {
    code: String(itemId),
    item_id: String(itemId),
    name,
    quantity,
    weight: 1,
    ...(choices ? { choices } : {}),
  }
}

function choicePrize() {
  return bPrize(1207751, '真爱无敌戒指选择权', 1, RING_CHOICES)
}

function stageRule(index: number) {
  const reward = apiState.rewards[index]
  const prize = bPrize(
    reward?.item_id ?? 0,
    reward?.item_name ?? '',
    reward?.quantity ?? 1,
    reward?.is_choice ? RING_CHOICES : undefined,
  )
  if (reward?.is_choice) prize.code = 'TRUE_LOVE_CHOICE'
  return prize
}

function rulesPayload() {
  return {
    pools: {
      DAY: {
        cost: 5,
        prizes: dayPoolPrizes.map((prize) => bPrize(
          prize.item_id,
          prize.item_name,
          prize.quantity,
          prize.is_choice ? RING_CHOICES : undefined,
        )),
      },
      NIGHT: {
        cost: 8,
        prizes: nightPoolPrizes.map((prize) => bPrize(
          prize.item_id,
          prize.item_name,
          prize.quantity,
          prize.is_choice ? RING_CHOICES : undefined,
        )),
      },
    },
    stages: Object.fromEntries(STAGE_CODES.map((stage, index) => [String(stage), stageRule(index)])),
    chests: [bPrize(1205251, '月亮游记戒指'), bPrize(1205470, '爱意翩跹戒指'), choicePrize()],
  }
}

function claimedMask() {
  return apiState.rewards.reduce((mask, reward) => (
    reward.claimed ? mask | (1 << reward.slot_index) : mask
  ), 0)
}

function statePayload() {
  return {
    gold_balance: apiState.gold,
    petal_balance: apiState.petals,
    current_round_no: apiState.roundNumber,
    formal_draw_count: apiState.tbpAvailable ? 0 : 1,
    flower_count: apiState.currentLitCount,
    // Tests use boxFlowState as an explicit "chance is exposed" switch; the
    // production backend derives readiness from these counters.
    unlocked_chests: apiState.boxFlowState === 'none' ? apiState.boxOpened : apiState.boxChances,
    completed_chests: apiState.boxOpened,
    claimed_stage_mask: claimedMask(),
    status: apiState.currentLitCount >= 18 ? 'FULL' : 'ACTIVE',
    rule_json: rulesPayload(),
  }
}

function joinPayload() {
  return {
    edition_id: String(apiState.activityPeriodId),
    gold: apiState.gold,
    petals: apiState.petals,
    round_no: apiState.roundNumber,
    flowers: apiState.currentLitCount,
    unlocked_chests: apiState.boxChances,
    completed_chests: apiState.boxOpened,
    claimed_stage_mask: claimedMask(),
    score: 0,
    rules: rulesPayload(),
  }
}

function purchasesPayload() {
  const orders: Array<{ product_code: string; order_id: string }> = []
  apiState.purchasedPetalShopTiers.forEach((tier, index) => {
    orders.push({ product_code: String(tier), order_id: `tier-${index}` })
  })
  if (apiState.petalGiftPurchased) orders.push({ product_code: 'gift', order_id: 'gift-1' })
  return orders
}

function drawChoiceReward(item: DrawPendingChoice) {
  const choices = RING_CHOICES
  return {
    grant_id: String(item.reward_record_id),
    source_type: apiState.drawPoolType === 2 ? 'NIGHT_DRAW' : 'DAY_DRAW',
    source_id: '1',
    source_entry_no: 1,
    round_no: apiState.roundNumber,
    draw_id: '10',
    original_reward_code: 'TRUE_LOVE_CHOICE',
    original_item_name: item.item_name,
    quantity: item.quantity,
    status: 'PENDING_CHOICE',
    candidate_snapshot: [bPrize(item.item_id, item.item_name, item.quantity, choices)],
    created_at: '2026-07-10T12:00:00Z',
  }
}

function stageChoiceReward() {
  const pending = apiState.stagePendingChoice
  if (!pending) return null
  return {
    grant_id: String(pending.reward_record_id),
    source_type: 'STAGE',
    source_id: '1',
    source_entry_no: 1,
    round_no: apiState.roundNumber,
    original_reward_code: 'TRUE_LOVE_CHOICE',
    original_item_name: pending.item_name,
    quantity: pending.quantity,
    status: 'PENDING_CHOICE',
    candidate_snapshot: [bPrize(pending.item_id, pending.item_name, pending.quantity, RING_CHOICES)],
    created_at: '2026-07-10T12:00:00Z',
  }
}

function rewardsPayload(status?: string | null) {
  const rows: any[] = []
  const stage = stageChoiceReward()
  if (stage) rows.push(stage)
  apiState.drawPendingChoices.forEach((choice) => rows.push(drawChoiceReward(choice)))
  ringRecordItems.forEach((item) => {
    rows.push({
      grant_id: String(item.id),
      source_type: item.source === 'treasure_box' ? 'CHEST' : 'DAY_DRAW',
      source_id: '1',
      source_entry_no: 1,
      round_no: apiState.roundNumber,
      original_reward_code: 'RING',
      original_item_name: item.item_name,
      quantity: 1,
      status: 'OBTAINED',
      final_item_id: '1205251',
      final_item_name: item.item_name,
      candidate_snapshot: null,
      created_at: item.created_at,
    })
  })
  return status ? rows.filter((row) => row.status === status) : rows
}

function chestRowsPayload() {
  const rows: any[] = []
  if (apiState.boxPendingReward) {
    const pending = apiState.boxPendingReward
    rows.push({
      chest_id: String(pending.box_id),
      round_no: apiState.roundNumber,
      chest_no: pending.chance_index,
      status: 'OPENED',
      candidate_snapshot: bPrize(
        pending.item_id,
        pending.item_name,
        1,
        pending.is_choice ? RING_CHOICES : undefined,
      ),
      grant_id: String(pending.grant_id ?? 901),
      reward_status: pending.is_choice ? 'PENDING_CHOICE' : 'OBTAINED',
    })
  }
  apiState.boxOpenedRewards.forEach((reward, index) => {
    rows.push({
      chest_id: `confirmed-${index + 1}`,
      round_no: apiState.roundNumber,
      chest_no: reward.chance_index,
      status: 'CONFIRMED',
      candidate_snapshot: bPrize(reward.item_id, reward.item_name, 1),
      grant_id: `opened-${index + 1}`,
      reward_status: 'OBTAINED',
    })
  })
  return rows
}

function tbpSteps() {
  const steps: any[] = []
  const lit = Math.max(0, Math.min(180, apiState.tbpEstimatedLitFlowers))
  for (let index = 0; index < 180; index += 1) {
    const prize = index === 80
      ? bPrize(1205251, '月亮游记戒指', 1)
      : bPrize(1002, '花瓣', 1)
    const before = Math.min(index, lit)
    const after = index < lit ? before + 1 : before
    steps.push({ prize, flowers_before: before, flowers_after: after, probability_lit: index < lit ? 1 : 0, gold_value: 0 })
  }
  return steps
}

function previewPayload() {
  return {
    preview_id: '77',
    status: 'PENDING',
    result_json: tbpSteps(),
    required_petals: 900,
  }
}

function rankingPayload() {
  return {
    top: rankingItems.map((item) => ({
      user_id: String(item.user_id),
      nickname: item.nickname,
      score: item.score,
      score_reached_at_sec: Math.floor(Date.parse(item.achieved_at) / 1000),
      rank: item.rank,
    })),
    me: {
      user_id: '1',
      nickname: '测试用户',
      score: 1_280,
      score_reached_at_sec: Math.floor(Date.parse('2026-07-10T13:00:00Z') / 1000),
      rank: 23,
    },
    source: 'mysql',
  }
}

export const server = setupServer(
  // ---- New contract: identity ----
  http.post('/api/identity/v1/auth/refresh', () => HttpResponse.json({
    access_token: 'test-token', user_id: '1', nickname: '测试用户',
  })),
  http.post('/api/identity/v1/auth/login', () => HttpResponse.json({
    access_token: 'test-token', user_id: '1', nickname: '测试用户',
  })),
  http.post('/api/identity/v1/auth/register', () => HttpResponse.json({
    access_token: 'test-token', user_id: '1', nickname: '测试用户',
  })),
  http.post('/api/identity/v1/auth/logout', () => HttpResponse.json({ status: 'ok' })),
  http.get('/api/identity/v1/me', () => HttpResponse.json({
    user_id: '1', session_id: 's1', nickname: '测试用户',
  })),

  // ---- New contract: activity session ----
  http.get('/api/activity/v1/records/config', () => {
    const ended = apiState.activityStatus === 2
    const notStarted = apiState.activityStatus === 0
    return HttpResponse.json([{
      edition_id: String(apiState.activityPeriodId),
      name: 'flower_2026',
      status: ended ? 'ENDED' : 'ACTIVE',
      starts_at: notStarted ? '2099-01-01T00:00:00Z' : '2026-01-01T00:00:00Z',
      ends_at: ended ? '2026-02-01T00:00:00Z' : '2099-12-31T23:59:59Z',
      settlement_status: 'PENDING',
    }])
  }),
  http.post('/api/activity/v1/join', () => HttpResponse.json(joinPayload())),
  http.get('/api/activity/v1/records/state', () => HttpResponse.json([statePayload()])),
  http.get('/api/activity/v1/records/purchases', () => HttpResponse.json(purchasesPayload())),
  http.get('/api/activity/v1/records/rewards', ({ request }) => {
    const status = new URL(request.url).searchParams.get('status')
    return HttpResponse.json(rewardsPayload(status))
  }),
  http.get('/api/activity/v1/records/summary', () => HttpResponse.json(
    myRewardItems.map((item) => ({
      item_id: String(item.item_id),
      final_item_name: item.item_name,
      quantity: item.quantity,
    })),
  )),
  http.get('/api/activity/v1/records/draws', () => HttpResponse.json(
    drawHistoryItems.map((item) => ({
      draw_id: String(item.draw_log_id),
      pool_code: item.pool_type === 2 ? 'NIGHT' : 'DAY',
      mode: item.draw_mode === 2 ? 'PREVIEW' : 'NORMAL',
      requested_count: item.requested_count,
      actual_count: item.actual_count,
      consumed_petals: item.cost_petals,
      refunded_petals: item.refund_petals,
      flowers_added: item.lit_flowers,
      reward_snapshot_json: item.rewards.map((reward) => ({
        prize: bPrize(reward.item_id, reward.item_name, reward.quantity),
        flowers_before: 0,
        flowers_after: 0,
        probability_lit: 0,
        gold_value: 0,
      })),
      occurred_at: item.created_at,
    })),
  )),
  http.get('/api/activity/v1/records/flowers', () => HttpResponse.json([])),
  http.get('/api/activity/v1/records/chests', () => HttpResponse.json(chestRowsPayload())),
  http.get('/api/activity/v1/records/preview', () => HttpResponse.json(
    apiState.tbpActiveSession ? [previewPayload()] : [],
  )),

  // ---- New contract: ranking ----
  http.get('/api/ranking/v1/board', () => {
    apiState.rankingRequestCount += 1
    return HttpResponse.json(rankingPayload())
  }),

  // ---- New contract: lottery ----
  http.post('/api/activity/v1/draw', async ({ request }) => {
    const body = await request.json() as { pool_code: string; count: number; operation_id?: string }
    const poolType = body.pool_code === 'NIGHT' ? 2 : 1
    apiState.drawRequestCount += 1
    const actualCount = apiState.drawActualCount ?? body.count
    const litFlowers = apiState.drawLitFlowers ?? (body.count === 1 ? 1 : 3)
    const beforeCount = apiState.currentLitCount
    const litCountAfter = apiState.drawLitCountAfter ?? beforeCount + litFlowers
    apiState.drawPoolType = poolType
    apiState.drawCount = body.count
    apiState.tbpAvailable = false
    apiState.currentLitCount = litCountAfter
    if (litCountAfter >= 18) {
      apiState.boxChances = Math.max(apiState.boxChances, 3)
      if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    }
    if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    const rewards = createDrawRewards(actualCount, poolType)
    if (apiState.drawChoiceOnNextDraw) {
      const pendingChoice: DrawPendingChoice = {
        reward_record_id: '901',
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        quantity: 1,
        choice_ids: [1207751, 1207752, 1207753],
      }
      rewards[rewards.length - 1] = {
        reward_record_id: '901',
        item_id: 1207751,
        item_name: pendingChoice.item_name,
        quantity: 1,
        is_choice: true,
        choice_ids: pendingChoice.choice_ids,
        pool_type: poolType,
      }
      apiState.drawPendingChoices = [pendingChoice]
    }
    const steps: any[] = []
    let flowers = beforeCount
    const increments = Math.max(0, litCountAfter - beforeCount)
    rewards.forEach((reward, index) => {
      const before = flowers
      if (index < increments && flowers < 18) flowers += 1
      steps.push({
        prize: bPrize(
          reward.item_id,
          reward.item_name,
          reward.quantity,
          reward.is_choice ? RING_CHOICES : undefined,
        ),
        flowers_before: before,
        flowers_after: flowers,
        probability_lit: index < increments ? 1 : 0,
        gold_value: 0,
        grant_id: reward.reward_record_id ? String(reward.reward_record_id) : String(1000 + index),
      })
    })
    return HttpResponse.json({
      draw_id: '10',
      results: steps,
      consumed_petals: actualCount * (poolType === 2 ? 8 : 5),
      refunded_petals: apiState.drawRefundPetals,
      requested_count: body.count,
      actual_count: actualCount,
      flowers: litCountAfter,
    })
  }),

  // ---- New contract: reward selection ----
  http.post('/api/activity/v1/select', async ({ request }) => {
    const body = await request.json() as { id: string; item_id: string }
    const itemId = Number(body.item_id)
    const names: Record<number, string> = {
      1207751: '真爱无敌戒指经典款',
      1207752: '真爱无敌戒指铭文款',
      1207753: '爱你万年戒指铭文款',
    }
    const stage = apiState.stagePendingChoice
    if (stage && String(stage.reward_record_id) === body.id) {
      apiState.stageConfirmedItemId = itemId
      apiState.stagePendingChoice = null
      return HttpResponse.json({
        grant_id: body.id, item_id: body.item_id, item_name: names[itemId] ?? stage.item_name, quantity: 1,
      })
    }
    const draw = apiState.drawPendingChoices.find((item) => String(item.reward_record_id) === body.id)
    if (draw && draw.choice_ids.includes(itemId)) {
      apiState.drawConfirmedItemIds.push(itemId)
      apiState.drawPendingChoices = apiState.drawPendingChoices.filter((item) => item.reward_record_id !== body.id)
      return HttpResponse.json({
        grant_id: body.id, item_id: body.item_id, item_name: names[itemId] ?? draw.item_name, quantity: draw.quantity,
      })
    }
    const box = apiState.boxPendingReward
    if (box && String(box.grant_id ?? 901) === body.id) {
      apiState.boxConfirmedItemId = itemId
      return HttpResponse.json({
        grant_id: body.id, item_id: body.item_id, item_name: names[itemId] ?? box.item_name, quantity: 1,
      })
    }
    return HttpResponse.json({ error: 'NOT_FOUND' }, { status: 404 })
  }),

  // ---- New contract: stage ----
  http.post('/api/activity/v1/stage', async ({ request }) => {
    const body = await request.json() as { stage: number }
    const index = STAGE_CODES.indexOf(body.stage)
    if (index < 0) return HttpResponse.json({ error: 'INVALID_STAGE' }, { status: 400 })
    const claimed = apiState.rewards[index]
    apiState.claimedSlot = index
    apiState.rewards = apiState.rewards.map((reward, i) => i === index
      ? { ...reward, claimed: true, reachable: false }
      : reward)
    if (claimed?.is_choice) {
      apiState.stagePendingChoice = {
        reward_record_id: `50${index + 1}`,
        slot_index: index,
        item_id: claimed.item_id,
        item_name: claimed.item_name,
        quantity: claimed.quantity,
        choice_ids: [1207751, 1207752, 1207753],
      }
    }
    return HttpResponse.json({ grant_id: `50${index + 1}` })
  }),

  // ---- New contract: treasure box ----
  http.post('/api/activity/v1/chest-open', () => {
    apiState.boxOpenCount += 1
    apiState.boxPendingReward = apiState.boxNextRewardIsChoice
      ? {
          box_id: '9',
          grant_id: '901',
          chance_index: apiState.boxOpened + 1,
          item_id: 1207751,
          item_name: '真爱无敌戒指选择权',
          is_choice: true,
          choice_ids: [1207751, 1207752, 1207753],
        }
      : {
          box_id: '9',
          grant_id: '902',
          chance_index: apiState.boxOpened + 1,
          item_id: 1205251,
          item_name: '月亮游记戒指',
          is_choice: false,
          choice_ids: [],
        }
    apiState.boxFlowState = apiState.boxPendingReward.is_choice ? 'choice' : 'result'
    return HttpResponse.json({
      chest_id: apiState.boxPendingReward.box_id,
      grant_id: apiState.boxPendingReward.grant_id,
      reward: bPrize(
        apiState.boxPendingReward.item_id,
        apiState.boxPendingReward.item_name,
        1,
        apiState.boxPendingReward.is_choice ? RING_CHOICES : undefined,
      ),
    })
  }),
  http.post('/api/activity/v1/chest-confirm', async ({ request }) => {
    const body = await request.json() as { id: string }
    const pending = apiState.boxPendingReward
    if (!pending || String(pending.box_id) !== body.id) {
      return HttpResponse.json({ error: 'CHEST_NOT_FOUND' }, { status: 404 })
    }
    const selectedItemId = apiState.boxConfirmedItemId ?? pending.item_id
    apiState.boxConfirmCount += 1
    apiState.boxOpened = Math.max(apiState.boxOpened, pending.chance_index)
    apiState.boxOpenedRewards.push({
      chance_index: pending.chance_index,
      item_id: selectedItemId,
      item_name: selectedItemId === 1205470 ? '爱意翩跹戒指' : pending.item_name,
      is_choice: Boolean(pending.is_choice),
    })
    apiState.boxPendingReward = null
    apiState.boxFlowState = apiState.boxOpened < apiState.boxChances ? 'ready' : 'none'
    return HttpResponse.json({ status: 'CONFIRMED' })
  }),

  // ---- New contract: try-before-pay ----
  http.post('/api/activity/v1/preview', () => {
    apiState.tbpPreviewCount += 1
    apiState.tbpActiveSession = true
    return HttpResponse.json(previewPayload())
  }),
  http.post('/api/activity/v1/preview-confirm', () => {
    const shortage = Math.max(0, 900 - apiState.petals)
    const goldCost = shortage * 60
    if (apiState.gold < goldCost) {
      return HttpResponse.json({ error: 'INSUFFICIENT_GOLD' }, { status: 409 })
    }
    apiState.gold -= goldCost
    apiState.petals -= Math.min(900, apiState.petals)
    apiState.tbpConfirmed = true
    apiState.tbpActiveSession = false
    apiState.tbpAvailable = false
    if (apiState.tbpChoiceOnConfirm) {
      apiState.drawPendingChoices = [{
        reward_record_id: '903',
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        quantity: 1,
        choice_ids: [1207751, 1207752, 1207753],
      }]
    }
    if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    return HttpResponse.json({
      draw_id: '30',
      results: [],
      consumed_petals: 900,
      refunded_petals: 0,
      requested_count: 180,
      actual_count: 180,
      flowers: apiState.currentLitCount + apiState.tbpEstimatedLitFlowers,
    })
  }),
  http.post('/api/activity/v1/preview-abandon', () => {
    apiState.tbpAbandoned = true
    apiState.tbpActiveSession = false
    apiState.tbpAvailable = false
    return HttpResponse.json({ status: 'ABANDONED' })
  }),

  // ---- New contract: next round ----
  http.post('/api/activity/v1/next-round', () => {
    apiState.newRoundRequestCount += 1
    if (apiState.currentLitCount < 18) {
      return HttpResponse.json({ error: 'ROUND_NOT_COMPLETE' }, { status: 409 })
    }
    if (apiState.boxOpened < apiState.boxChances) {
      return HttpResponse.json({ error: 'ROUND_NOT_COMPLETE' }, { status: 409 })
    }
    const hasUnclaimedReward = apiState.rewards.some((reward) => (
      reward.lit_threshold <= apiState.currentLitCount && !reward.claimed
    ))
    if (hasUnclaimedReward) {
      return HttpResponse.json({ error: 'ROUND_NOT_COMPLETE' }, { status: 409 })
    }
    apiState.roundNumber += 1
    apiState.currentLitCount = 0
    apiState.rewards = createStageRewards().map((reward) => ({ ...reward, claimed: false, reachable: false }))
    apiState.boxChances = 0
    apiState.boxOpened = 0
    apiState.boxFlowState = 'none'
    apiState.boxPendingReward = null
    apiState.boxOpenedRewards = []
    apiState.newRoundStarted = true
    return HttpResponse.json({ round_no: apiState.roundNumber })
  }),

  // ---- New contract: purchase ----
  http.post('/api/activity/v1/purchase', async ({ request }) => {
    const body = await request.json() as { product_code: string; petals?: number }
    if (body.product_code === 'gift') {
      if (apiState.petalGiftPurchased) {
        return HttpResponse.json({ error: 'ALREADY_PURCHASED' }, { status: 409 })
      }
      apiState.petalGiftPurchased = true
      apiState.petalGiftPurchaseCount += 1
      apiState.petals += 100
      return HttpResponse.json({ order_id: 'gift-1', gold: apiState.gold, petals: apiState.petals })
    }
    const tier = Number(body.product_code)
    const amount = body.product_code === 'top-up' ? Number(body.petals ?? 0) : tier
    if (!Number.isFinite(amount) || amount <= 0) {
      return HttpResponse.json({ error: 'INVALID_REQUEST' }, { status: 400 })
    }
    const isOneTimeTier = TIER_AMOUNTS_FOR_TEST.includes(amount)
    if (isOneTimeTier && apiState.purchasedPetalShopTiers.includes(amount)) {
      return HttpResponse.json({ error: 'ALREADY_PURCHASED' }, { status: 409 })
    }
    const costGold = body.product_code === 'top-up' ? amount * 60 : getPetalCostGold(amount)
    if (apiState.gold < costGold) {
      return HttpResponse.json({ error: 'INSUFFICIENT_OR_OVERFLOWING_ASSET' }, { status: 409 })
    }
    apiState.gold -= costGold
    apiState.petals += amount
    if (isOneTimeTier) apiState.purchasedPetalShopTiers.push(amount)
    return HttpResponse.json({
      order_id: `order-${Date.now()}`,
      gold: apiState.gold,
      petals: apiState.petals,
    })
  }),

  // ---- New contract: SSE ----
  http.get('/api/notification/v1/barrage/stream', () => {
    apiState.marqueeRequestCount += 1
    const ready = 'event: ready\ndata: {"status":"connected"}\n\n'
    const events = apiState.marqueeItems.map((item) => {
      const id = String(item.id)
      return `event: barrage\nid: ${id}\ndata: ${JSON.stringify({
        message_id: id,
        display_name: item.nickname,
        item_name: item.item_name,
        quantity: 1,
        expires_at: '2099-01-01T00:00:00Z',
      })}\n\n`
    }).join('')
    return new HttpResponse(ready + events, {
      headers: { 'Content-Type': 'text/event-stream' },
    })
  }),

  // ---- Legacy handlers below are no longer reached by the migrated adapters.
  http.get('/api/v1/auth/profile', () => success({ user_id: 1, username: 'tester', nickname: '测试用户' })),
  http.post('/api/v1/auth/logout', () => success(null)),
  http.get('/api/v1/activity/config', () => success({
    activity_key: 'flower_2026',
    activity_period_id: apiState.activityPeriodId,
    period_no: apiState.activityPeriodId,
    status: apiState.activityStatus,
    status_text: apiState.activityStatus === 1 ? '进行中' : apiState.activityStatus === 2 ? '已结束' : '未开始',
    start_time: '2026-07-01T00:00:00+08:00',
    end_time: '2026-07-31T23:59:59+08:00',
  })),
  // SSE mock：连接建立后不补发历史，仅发 connected 注释后关闭连接。
  // jsdom 环境无法模拟真实长连接流，测试通过「注入 marqueeItems + 触发重连」
  // 模拟连接后到达的实时弹幕；marqueeItems 为空时只收到 connected。
  http.get('/api/v1/activity/marquee/stream', () => {
    apiState.marqueeRequestCount += 1
    const events = apiState.marqueeItems
      .map((item) => `data: ${JSON.stringify(item)}\n\n`)
      .join('')
    return new HttpResponse(`${events}: connected\n\n`, {
      headers: { 'Content-Type': 'text/event-stream' },
    })
  }),
  http.get('/api/v1/ranking', () => {
    apiState.rankingRequestCount += 1
    return success({
      top_20: rankingItems,
      current_user: {
        rank: 23,
        score: 1_280,
        achieved_at: '2026-07-10T13:00:00Z',
      },
    })
  }),
  http.get('/api/v1/assets', () => success({ gold: apiState.gold, petals: apiState.petals })),
  http.get('/api/v1/rewards/mine', () => success({ list: myRewardItems })),
  http.get('/api/v1/assets/petal-shop/status', () => success({
    purchased_tiers: apiState.purchasedPetalShopTiers,
  })),
  http.get('/api/v1/assets/petal-gift/status', () => success({
    available: !apiState.petalGiftPurchased,
    purchased: apiState.petalGiftPurchased,
  })),
  http.post('/api/v1/assets/petal-gift/purchase', () => {
    if (apiState.petalGiftPurchased) return failure(20005, '该花瓣礼包每位用户限购一次')
    apiState.petalGiftPurchased = true
    apiState.petalGiftPurchaseCount += 1
    apiState.petals += 100
    return success({ amount: 100, price_yuan: 30, petals_after: apiState.petals })
  }),
  http.post('/api/v1/assets/buy-petals', async ({ request }) => {
    const body = await request.json() as { amount: number }
    const isOneTimeTier = [52, 525, 1580].includes(body.amount)
    if (isOneTimeTier && apiState.purchasedPetalShopTiers.includes(body.amount)) {
      return failure(20006, '该赠送档位每位用户限购一次')
    }
    const goldBefore = apiState.gold
    const petalsBefore = apiState.petals
    const costGold = getPetalCostGold(body.amount)
    apiState.gold -= costGold
    apiState.petals += body.amount
    if (isOneTimeTier) apiState.purchasedPetalShopTiers.push(body.amount)
    return success({
      gold_before: goldBefore,
      gold_after: apiState.gold,
      petals_before: petalsBefore,
      petals_after: apiState.petals,
      amount: body.amount,
      cost_gold: costGold,
    })
  }),
  http.post('/api/v1/assets/exchange-petals', async ({ request }) => {
    const body = await request.json() as { amount: number }
    const goldBefore = apiState.gold
    const petalsBefore = apiState.petals
    const costGold = body.amount * 60
    apiState.gold -= costGold
    apiState.petals += body.amount
    return success({
      gold_before: goldBefore,
      gold_after: apiState.gold,
      petals_before: petalsBefore,
      petals_after: apiState.petals,
      amount: body.amount,
      cost_gold: costGold,
    })
  }),
  http.get('/api/v1/lottery/pools', () => success([
    { pool_type: 1, pool_name: '白昼花园', cost_per_draw: 5, status: 1, prizes: dayPoolPrizes },
    { pool_type: 2, pool_name: '星夜花园', cost_per_draw: 8, status: 1, prizes: nightPoolPrizes },
  ])),
  http.get('/api/v1/lottery/draw/history', ({ request }) => {
    const page = Number(new URL(request.url).searchParams.get('page') ?? '1')
    return success({
      total: drawHistoryItems.length,
      page,
      page_size: 5,
      list: page === 1 ? drawHistoryItems : [],
    })
  }),
  http.get('/api/v1/rewards/rings', ({ request }) => {
    const page = Number(new URL(request.url).searchParams.get('page') ?? '1')
    return success({
      total: ringRecordItems.length,
      page,
      list: page === 1 ? ringRecordItems : [],
    })
  }),
  http.post('/api/v1/lottery/draw', async ({ request }) => {
    const body = await request.json() as { pool_type: number; count: number }
    apiState.drawRequestCount += 1
    const actualCount = apiState.drawActualCount ?? body.count
    const litFlowers = apiState.drawLitFlowers ?? (body.count === 1 ? 1 : 3)
    const litCountAfter = apiState.drawLitCountAfter ?? apiState.currentLitCount + litFlowers
    apiState.drawPoolType = body.pool_type
    apiState.drawCount = body.count
    apiState.tbpAvailable = false
    apiState.currentLitCount = litCountAfter
    if (litCountAfter >= 18) {
      apiState.boxChances = Math.max(apiState.boxChances, 3)
      if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    }
    if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    const rewards = createDrawRewards(actualCount, body.pool_type)
    if (apiState.drawChoiceOnNextDraw) {
      const pendingChoice: DrawPendingChoice = {
        reward_record_id: 901,
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        quantity: 1,
        choice_ids: [1207751, 1207752, 1207753],
      }
      rewards[0] = { ...pendingChoice, is_choice: true, pool_type: body.pool_type }
      apiState.drawPendingChoices = [pendingChoice]
    }
    return success({
      draw_log_id: 10,
      rewards,
      lit_flowers: litFlowers,
      actual_count: actualCount,
      requested_count: body.count,
      refund_petals: apiState.drawRefundPetals,
      round_status: litCountAfter >= 18 ? 2 : 1,
      lit_count_after: litCountAfter,
    })
  }),
  http.get('/api/v1/lottery/draw/choices/pending', () => success(apiState.drawPendingChoices)),
  http.post('/api/v1/lottery/draw/choices/confirm', async ({ request }) => {
    const body = await request.json() as { reward_record_id: number; selected_item_id: number }
    const pending = apiState.drawPendingChoices.find((item) => item.reward_record_id === body.reward_record_id)
    if (!pending || !pending.choice_ids.includes(body.selected_item_id)) {
      return failure(30006, '请选择有效的抽奖奖励戒指')
    }
    apiState.drawConfirmedItemIds.push(body.selected_item_id)
    apiState.drawPendingChoices = apiState.drawPendingChoices.filter((item) => item.reward_record_id !== body.reward_record_id)
    const names: Record<number, string> = {
      1207751: '真爱无敌戒指经典款',
      1207752: '真爱无敌戒指铭文款',
      1207753: '爱你万年戒指铭文款',
    }
    return success({
      reward_record_id: body.reward_record_id,
      item_id: body.selected_item_id,
      item_name: names[body.selected_item_id],
      quantity: pending.quantity,
    })
  }),
  http.get('/api/v1/tbp/status', () => success({
    available: apiState.tbpAvailable,
    session: apiState.tbpActiveSession
      ? { session_id: 77, status: 1, total_cost: 900, created_at: '2026-07-10T12:00:00Z' }
      : undefined,
    preview: apiState.tbpActiveSession ? createTryBeforePayPreview() : undefined,
  })),
  http.post('/api/v1/tbp/preview', () => {
    apiState.tbpPreviewCount += 1
    apiState.tbpActiveSession = true
    return success(createTryBeforePayPreview())
  }),
  http.post('/api/v1/tbp/confirm', () => {
    const shortage = Math.max(0, 900 - apiState.petals)
    const goldCost = shortage * 60
    if (apiState.gold < goldCost) {
      return failure(20001, `金币不足，需要 ${goldCost}，当前 ${apiState.gold}`)
    }
    apiState.tbpConfirmed = true
    apiState.tbpActiveSession = false
    apiState.tbpAvailable = false
    apiState.petals -= 900
    if (apiState.tbpChoiceOnConfirm) {
      apiState.drawPendingChoices = [{
        reward_record_id: 903,
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        quantity: 1,
        choice_ids: [1207751, 1207752, 1207753],
      }]
    }
    if (apiState.boxChances > apiState.boxOpened) apiState.boxFlowState = 'ready'
    return success({ session_id: 77, total_cost: 900, draw_log_id: 30 })
  }),
  http.post('/api/v1/tbp/abandon', () => {
    apiState.tbpAbandoned = true
    apiState.tbpActiveSession = false
    apiState.tbpAvailable = false
    return success({ message: '已放弃先抽后付预览' })
  }),
  http.post('/api/v1/flower/new-round', () => {
    apiState.newRoundRequestCount += 1
    if (apiState.currentLitCount < 18) return failure(40006, '需要先点亮18朵花')
    if (apiState.boxOpened < apiState.boxChances) return failure(40006, '请先完成本轮戒指召唤')

    const hasUnclaimedReward = apiState.rewards.some((reward) => (
      reward.lit_threshold <= apiState.currentLitCount && !reward.claimed
    ))
    if (hasUnclaimedReward) return failure(40006, '请先领取全部阶段奖励')

    apiState.roundNumber += 1
    apiState.currentLitCount = 0
    apiState.rewards = createStageRewards().map((reward) => ({
      ...reward,
      claimed: false,
      reachable: false,
    }))
    apiState.boxChances = 0
    apiState.boxOpened = 0
    apiState.boxFlowState = 'none'
    apiState.boxPendingReward = null
    apiState.boxOpenedRewards = []
    apiState.newRoundStarted = true
    return success({
      round_number: apiState.roundNumber,
      lit_count: 0,
      cumulative_gold_spent: 0,
      box_chances: 0,
      box_opened: 0,
      stage_claimed: [],
      round_status: 1,
      max_flowers: 18,
    })
  }),
  http.get('/api/v1/box/status', () => success({
    round_number: apiState.roundNumber,
    box_chances: apiState.boxChances,
    box_opened: apiState.boxOpened,
    flow_state: apiState.boxFlowState,
    next_chance_index: apiState.boxFlowState === 'none' ? undefined : apiState.boxOpened + 1,
    pending_reward: apiState.boxPendingReward ?? undefined,
    opened_rewards: apiState.boxOpenedRewards,
  })),
  http.post('/api/v1/box/open', () => {
    apiState.boxOpenCount += 1
    if (!apiState.boxPendingReward) {
      apiState.boxPendingReward = apiState.boxNextRewardIsChoice
        ? {
            box_id: 9,
            chance_index: apiState.boxOpened + 1,
            item_id: 1207751,
            item_name: '真爱无敌戒指选择权',
            is_choice: true,
            choice_ids: [1207751, 1207752, 1207753],
          }
        : {
            box_id: 9,
            chance_index: apiState.boxOpened + 1,
            item_id: 1205251,
            item_name: '月亮游记戒指',
            is_choice: false,
            choice_ids: [],
          }
    }
    apiState.boxFlowState = apiState.boxPendingReward.is_choice ? 'choice' : 'result'
    return success(apiState.boxPendingReward)
  }),
  http.post('/api/v1/box/confirm', async ({ request }) => {
    const body = await request.json() as { chance_index: number; selected_item_id: number }
    const pending = apiState.boxPendingReward
    const selectedItemId = pending?.is_choice ? body.selected_item_id : pending?.item_id ?? 0
    apiState.boxConfirmCount += 1
    apiState.boxConfirmedItemId = selectedItemId
    apiState.boxOpened = Math.max(apiState.boxOpened, body.chance_index)
    apiState.boxOpenedRewards.push({
      chance_index: body.chance_index,
      item_id: selectedItemId,
      item_name: selectedItemId === 1205470 ? '爱意翩跹戒指' : pending?.item_name ?? '月亮游记戒指',
      is_choice: Boolean(pending?.is_choice),
    })
    apiState.boxPendingReward = null
    apiState.boxFlowState = apiState.boxOpened < apiState.boxChances ? 'ready' : 'none'
    return success({
      chance_index: body.chance_index,
      item_id: selectedItemId,
      item_name: selectedItemId === 1207751
        ? '真爱无敌戒指经典款'
        : selectedItemId === 1207752
          ? '真爱无敌戒指铭文款'
          : selectedItemId === 1207753
            ? '爱你万年戒指铭文款'
            : '月亮游记戒指',
      is_choice: body.selected_item_id > 0,
    })
  }),
  http.get('/api/v1/rewards/stage', () => success({
    rewards: apiState.rewards,
    current_lit_count: apiState.currentLitCount,
    pending_choice: apiState.stagePendingChoice ?? undefined,
  })),
  http.post('/api/v1/rewards/stage/claim', async ({ request }) => {
    const body = await request.json() as { slot_index: number }
    const claimedReward = apiState.rewards.find((reward) => reward.slot_index === body.slot_index)
    apiState.claimedSlot = body.slot_index
    apiState.rewards = apiState.rewards.map((reward) => reward.slot_index === body.slot_index
      ? { ...reward, claimed: true, reachable: false }
      : reward)
    if (claimedReward?.is_choice) {
      apiState.stagePendingChoice = {
        reward_record_id: 501,
        slot_index: body.slot_index,
        item_id: claimedReward.item_id,
        item_name: claimedReward.item_name,
        quantity: claimedReward.quantity,
        choice_ids: [1207751, 1207752, 1207753],
      }
    }
    return success({
      slot_index: body.slot_index,
      item_id: claimedReward?.item_id ?? 0,
      item_name: claimedReward?.item_name ?? '',
      quantity: claimedReward?.quantity ?? 0,
      is_choice: claimedReward?.is_choice ?? false,
      pending_choice: apiState.stagePendingChoice ?? undefined,
    })
  }),
  http.post('/api/v1/rewards/stage/confirm', async ({ request }) => {
    const body = await request.json() as { reward_record_id: number; selected_item_id: number }
    apiState.stageConfirmedItemId = body.selected_item_id
    apiState.stagePendingChoice = null
    const names: Record<number, string> = {
      1207751: '真爱无敌戒指经典款',
      1207752: '真爱无敌戒指铭文款',
      1207753: '爱你万年戒指铭文款',
    }
    return success({
      slot_index: 5,
      item_id: body.selected_item_id,
      item_name: names[body.selected_item_id],
      quantity: 1,
      is_choice: true,
    })
  }),
)
