import request from './request'
import { idempotencyOperations, withIdempotency } from './idempotency'

// ---- 资产余额 ----
export interface AssetsResult {
  gold: number
  petals: number
}

// ---- 购买花瓣请求 ----
export interface BuyPetalsParams {
  amount: number
}

// ---- 购买花瓣响应 ----
export interface BuyPetalsResult {
  gold_before: number
  gold_after: number
  petals_before: number
  petals_after: number
  amount: number
  cost_gold: number
}

export interface PetalGiftStatusResult {
  available: boolean
  purchased: boolean
}

export interface PurchasePetalGiftResult {
  amount: number
  price_yuan: number
  petals_after: number
}

export interface PetalShopStatusResult {
  purchased_tiers: number[]
}

// ---- 查询资产余额 ----
export function getAssets(): Promise<AssetsResult> {
  return request.get('/assets').then((res) => res.data)
}

// ---- 购买花瓣（活动档位按活动价，其它数量按 amount × 60 金币） ----
export function buyPetals(amount: number): Promise<BuyPetalsResult> {
  const payload = { amount }
  return withIdempotency(idempotencyOperations.buyPetals, payload, (idempotencyKey) =>
    request.post('/assets/buy-petals', payload, {
      headers: { 'Idempotency-Key': idempotencyKey },
    }).then((res) => res.data),
  )
}

export function getPetalShopStatus(): Promise<PetalShopStatusResult> {
  return request.get('/assets/petal-shop/status').then((res) => res.data)
}

// 抽奖补差兑换，后端固定按 1 花瓣 = 60 金币结算。
export function exchangePetals(amount: number): Promise<BuyPetalsResult> {
  const payload = { amount }
  return withIdempotency(idempotencyOperations.exchangePetals, payload, (idempotencyKey) =>
    request.post('/assets/exchange-petals', payload, {
      headers: { 'Idempotency-Key': idempotencyKey },
    }).then((res) => res.data),
  )
}

export function getPetalGiftStatus(): Promise<PetalGiftStatusResult> {
  return request.get('/assets/petal-gift/status').then((res) => res.data)
}

export function purchasePetalGift(): Promise<PurchasePetalGiftResult> {
  const payload = {}
  return withIdempotency(idempotencyOperations.petalGiftPurchase, payload, (idempotencyKey) =>
    request.post('/assets/petal-gift/purchase', payload, {
      headers: { 'Idempotency-Key': idempotencyKey },
    }).then((res) => res.data),
  )
}
