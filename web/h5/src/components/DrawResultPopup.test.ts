import { fireEvent, render, screen, within } from '@testing-library/vue'
import DrawResultPopup from './DrawResultPopup.vue'
import type { DrawResponse } from '@/api/lottery'

const redrawCases = [
  { poolType: 1, count: 1, asset: '/image/activity/results/redraw-day-1.png' },
  { poolType: 1, count: 10, asset: '/image/activity/results/redraw-day-10.png' },
  { poolType: 1, count: 30, asset: '/image/activity/results/redraw-day-30.png' },
  { poolType: 2, count: 1, asset: '/image/activity/results/redraw-night-1.png' },
  { poolType: 2, count: 10, asset: '/image/activity/results/redraw-night-10.png' },
  { poolType: 2, count: 30, asset: '/image/activity/results/redraw-night-30.png' },
] as const

function createResult(count: number, poolType: number, litFlowers = 0): DrawResponse {
  return {
    draw_log_id: 1,
    rewards: [{
      item_id: 1002,
      item_name: '花瓣',
      quantity: 1,
      is_choice: false,
      choice_ids: null,
      pool_type: poolType,
    }],
    lit_flowers: litFlowers,
    actual_count: count,
    requested_count: count,
    refund_petals: 0,
    round_status: 1,
    lit_count_after: litFlowers,
  }
}

describe('DrawResultPopup', () => {
  test.each(redrawCases)('shows the matching redraw asset for pool $poolType and count $count', ({
    poolType,
    count,
    asset,
  }) => {
    render(DrawResultPopup, {
      props: {
        result: createResult(count, poolType),
        confirming: false,
        poolType,
      },
    })

    const dialog = screen.getByRole('dialog', { name: '恭喜获得' })
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm-pair.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByRole('button', { name: `再抽${count}次` }).querySelector(
      `img[src="${asset}"]`,
    )).not.toBeNull()
  })

  test('keeps the original single confirm button when flowers are lit', () => {
    render(DrawResultPopup, {
      props: {
        result: createResult(1, 1, 1),
        confirming: false,
        poolType: 1,
      },
    })

    const dialog = screen.getByRole('dialog', { name: '恭喜获得' })
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm.png"]',
    )).not.toBeNull()
    expect(within(dialog).queryByRole('button', { name: '再抽1次' })).not.toBeInTheDocument()
  })

  test('keeps the original single confirm button for a true-love reward without flowers', () => {
    const result = createResult(1, 1)
    result.rewards[0] = {
      reward_record_id: 9,
      item_id: 1207751,
      item_name: '真爱无敌戒指选择权',
      quantity: 1,
      is_choice: true,
      choice_ids: [1207751, 1207752, 1207753],
      pool_type: 1,
    }
    render(DrawResultPopup, {
      props: {
        result,
        confirming: false,
        poolType: 1,
      },
    })

    const dialog = screen.getByRole('dialog', { name: '恭喜获得' })
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm.png"]',
    )).not.toBeNull()
    expect(within(dialog).queryByRole('button', { name: '再抽1次' })).not.toBeInTheDocument()
  })

  test('emits the requested draw count from the redraw button', async () => {
    const result = render(DrawResultPopup, {
      props: {
        result: createResult(30, 2),
        confirming: false,
        poolType: 2,
      },
    })

    await fireEvent.click(screen.getByRole('button', { name: '再抽30次' }))

    expect(result.emitted().redraw).toEqual([[30]])
  })
})
