const STORAGE_PREFIX = 'lottery_idempotency:'
const KEY_TTL_MS = 24 * 60 * 60 * 1000

export const idempotencyOperations = {
  lotteryDraw: 'lottery.draw',
  buyPetals: 'assets.buy_petals',
  exchangePetals: 'assets.exchange_petals',
  petalGiftPurchase: 'assets.petal_gift_purchase',
} as const

interface PendingRequest {
  key: string
  fingerprint: string
  createdAt: number
}

const memoryStorage = new Map<string, string>()
const inFlightRequests = new Map<string, Promise<unknown>>()

function normalize(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(normalize)
  }
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, item]) => [key, normalize(item)]),
    )
  }
  return value
}

function requestFingerprint(operation: string, payload: unknown): string {
  return `${operation}:${JSON.stringify(normalize(payload)) ?? 'null'}`
}

function hashFingerprint(value: string): string {
  let hash = 2166136261
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return (hash >>> 0).toString(16).padStart(8, '0')
}

function storageKey(fingerprint: string): string {
  const operation = fingerprint.slice(0, fingerprint.indexOf(':'))
  return `${STORAGE_PREFIX}${operation}:${hashFingerprint(fingerprint)}`
}

function readStorage(key: string): string | null {
  try {
    return sessionStorage.getItem(key)
  } catch {
    return memoryStorage.get(key) ?? null
  }
}

function writeStorage(key: string, value: string): void {
  try {
    sessionStorage.setItem(key, value)
  } catch {
    memoryStorage.set(key, value)
  }
}

function removeStorage(key: string): void {
  try {
    sessionStorage.removeItem(key)
  } catch {
    memoryStorage.delete(key)
  }
}

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    const bytes = crypto.getRandomValues(new Uint8Array(16))
    bytes[6] = (bytes[6] & 0x0f) | 0x40
    bytes[8] = (bytes[8] & 0x3f) | 0x80
    const hex = Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('')
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  }

  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`
}

function getOrCreatePendingRequest(operation: string, payload: unknown): { storageKey: string; request: PendingRequest } {
  const fingerprint = requestFingerprint(operation, payload)
  const key = storageKey(fingerprint)
  const stored = readStorage(key)
  if (stored) {
    try {
      const request = JSON.parse(stored) as PendingRequest
      if (request.fingerprint === fingerprint && Date.now() - request.createdAt < KEY_TTL_MS) {
        return { storageKey: key, request }
      }
    } catch {
      // 损坏或旧格式记录会在下方被覆盖。
    }
  }

  const request: PendingRequest = {
    key: createIdempotencyKey(),
    fingerprint,
    createdAt: Date.now(),
  }
  writeStorage(key, JSON.stringify(request))
  return { storageKey: key, request }
}

export function withIdempotency<T>(
  operation: string,
  payload: unknown,
  send: (idempotencyKey: string) => Promise<T>,
): Promise<T> {
  const pending = getOrCreatePendingRequest(operation, payload)
  const inFlightKey = pending.request.fingerprint
  const existing = inFlightRequests.get(inFlightKey)
  if (existing) {
    return existing as Promise<T>
  }

  const request = send(pending.request.key)
    .then((result) => {
      const current = readStorage(pending.storageKey)
      if (current) {
        try {
          const stored = JSON.parse(current) as PendingRequest
          if (stored.key === pending.request.key) {
            removeStorage(pending.storageKey)
          }
        } catch {
          removeStorage(pending.storageKey)
        }
      }
      return result
    })
    .finally(() => {
      inFlightRequests.delete(inFlightKey)
    })

  inFlightRequests.set(inFlightKey, request)
  return request
}

export function clearPendingIdempotencyKeys(): void {
  try {
    for (let index = sessionStorage.length - 1; index >= 0; index -= 1) {
      const key = sessionStorage.key(index)
      if (key?.startsWith(STORAGE_PREFIX)) {
        sessionStorage.removeItem(key)
      }
    }
  } catch {
    // sessionStorage 不可用时只清理内存降级存储。
  }
  memoryStorage.clear()
  inFlightRequests.clear()
}
