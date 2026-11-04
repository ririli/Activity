import request from './request'
import { idempotencyOperations, withIdempotency } from './idempotency'
import { activityEditionId, ensureActivity, loadState } from './activitySession'

export interface AssetsResult {
  gold: number
  petals: number
}

export interface BuyPetalsParams {
  amount: number
}

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

interface PurchaseOrder {
  product_code: string
  order_id: string
}

interface PurchaseResponse {
  order_id: string
  gold?: number
  petals?: number
}

const TIERS = new Set(['5', '52', '525', '1580'])
const TIER_AMOUNTS = new Set([5, 52, 525, 1580])

export async function getAssets(): Promise<AssetsResult> {
  await ensureActivity()
  const state = await loadState()
  return { gold: state.gold, petals: state.petals }
}

async function purchaseOrders(): Promise<PurchaseOrder[]> {
  await ensureActivity()
  return request.get('/activity/v1/records/purchases', {
    params: { edition_id: activityEditionId() },
  }).then((res) => res.data as PurchaseOrder[])
}

export function buyPetals(amount: number): Promise<BuyPetalsResult> {
  const payload = { amount }
  return withIdempotency(idempotencyOperations.buyPetals, payload, async (operationId) => {
    const before = await getAssets()
    await ensureActivity()
    const order = await request.post('/activity/v1/purchase', {
      edition_id: activityEditionId(),
      operation_id: operationId,
      product_code: TIER_AMOUNTS.has(amount) ? String(amount) : 'top-up',
      petals: TIER_AMOUNTS.has(amount) ? undefined : amount,
    }).then((res) => res.data as PurchaseResponse)
    const after = order.gold !== undefined && order.petals !== undefined
      ? { gold: order.gold, petals: order.petals }
      : await getAssets()
    return {
      gold_before: before.gold,
      gold_after: after.gold,
      petals_before: before.petals,
      petals_after: after.petals,
      amount,
      cost_gold: Math.max(0, before.gold - after.gold),
    }
  })
}

export async function getPetalShopStatus(): Promise<PetalShopStatusResult> {
  const orders = await purchaseOrders()
  return {
    purchased_tiers: orders
      .map((order) => order.product_code)
      .filter((code) => TIERS.has(code))
      .map(Number),
  }
}

export function exchangePetals(amount: number): Promise<BuyPetalsResult> {
  const payload = { amount }
  return withIdempotency(idempotencyOperations.exchangePetals, payload, async (operationId) => {
    const before = await getAssets()
    await ensureActivity()
    const order = await request.post('/activity/v1/purchase', {
      edition_id: activityEditionId(),
      operation_id: operationId,
      product_code: 'top-up',
      petals: amount,
    }).then((res) => res.data as PurchaseResponse)
    const after = order.gold !== undefined && order.petals !== undefined
      ? { gold: order.gold, petals: order.petals }
      : await getAssets()
    return {
      gold_before: before.gold,
      gold_after: after.gold,
      petals_before: before.petals,
      petals_after: after.petals,
      amount,
      cost_gold: Math.max(0, before.gold - after.gold),
    }
  })
}

export async function getPetalGiftStatus(): Promise<PetalGiftStatusResult> {
  const orders = await purchaseOrders()
  const purchased = orders.some((order) => order.product_code === 'gift')
  return { available: !purchased, purchased }
}

export function purchasePetalGift(): Promise<PurchasePetalGiftResult> {
  return withIdempotency(idempotencyOperations.petalGiftPurchase, {}, (operationId) =>
    ensureActivity()
      .then(() => request.post('/activity/v1/purchase', {
        edition_id: activityEditionId(),
        operation_id: operationId,
        product_code: 'gift',
      }))
      .then(async () => {
        const after = await getAssets()
        return { amount: 100, price_yuan: 0, petals_after: after.petals }
      }),
  )
}
