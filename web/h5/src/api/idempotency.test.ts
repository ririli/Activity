import {
  clearPendingIdempotencyKeys,
  idempotencyOperations,
  withIdempotency,
} from './idempotency'

describe('request idempotency', () => {
  beforeEach(() => {
    sessionStorage.clear()
    clearPendingIdempotencyKeys()
  })

  afterEach(() => {
    clearPendingIdempotencyKeys()
  })

  test('reuses the same key after an uncertain failure and rotates it after success', async () => {
    const keys: string[] = []
    const failedSend = vi.fn(async (key: string) => {
      keys.push(key)
      throw new Error('network timeout')
    })

    await expect(withIdempotency(
      idempotencyOperations.buyPetals,
      { amount: 52 },
      failedSend,
    )).rejects.toThrow('network timeout')

    const successfulSend = vi.fn(async (key: string) => {
      keys.push(key)
      return { amount: 52 }
    })
    await expect(withIdempotency(
      idempotencyOperations.buyPetals,
      { amount: 52 },
      successfulSend,
    )).resolves.toEqual({ amount: 52 })

    const nextSend = vi.fn(async (key: string) => {
      keys.push(key)
      return { amount: 52 }
    })
    await withIdempotency(idempotencyOperations.buyPetals, { amount: 52 }, nextSend)

    expect(keys[1]).toBe(keys[0])
    expect(keys[2]).not.toBe(keys[1])
  })

  test('deduplicates concurrent calls for the same operation and payload', async () => {
    let resolveRequest: ((value: string) => void) | undefined
    const send = vi.fn(() => new Promise<string>((resolve) => {
      resolveRequest = resolve
    }))

    const first = withIdempotency(idempotencyOperations.lotteryDraw, { pool_type: 1, count: 10 }, send)
    const second = withIdempotency(idempotencyOperations.lotteryDraw, { count: 10, pool_type: 1 }, send)

    expect(first).toBe(second)
    expect(send).toHaveBeenCalledTimes(1)

    resolveRequest?.('ok')
    await expect(first).resolves.toBe('ok')
  })

  test('uses different keys for different request parameters', async () => {
    const keys: string[] = []
    const send = vi.fn(async (key: string) => {
      keys.push(key)
      throw new Error('keep pending')
    })

    await expect(withIdempotency(idempotencyOperations.exchangePetals, { amount: 1 }, send)).rejects.toThrow()
    await expect(withIdempotency(idempotencyOperations.exchangePetals, { amount: 2 }, send)).rejects.toThrow()

    expect(keys).toHaveLength(2)
    expect(keys[0]).not.toBe(keys[1])
  })
})
