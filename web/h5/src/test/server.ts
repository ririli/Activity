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

export const server = setupServer(
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
