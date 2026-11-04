import { render, screen, within } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import RingSummonFlow from './RingSummonFlow.vue'
import type { BoxStatusResponse } from '@/api/treasureBox'

const { playActivityEffect } = vi.hoisted(() => ({
  playActivityEffect: vi.fn(),
}))

vi.mock('@/services/activityAudio', () => ({
  playActivityEffect,
}))

describe('RingSummonFlow', () => {
  beforeEach(() => {
    playActivityEffect.mockClear()
  })

  test('does not render an empty overlay when flow state is missing', () => {
    const malformedStatus = {
      round_number: 1,
      box_chances: 0,
      box_opened: 0,
      opened_rewards: [],
    } as unknown as BoxStatusResponse

    render(RingSummonFlow, {
      props: {
        status: malformedStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(document.querySelector('.summon-overlay')).toBeNull()
  })

  test('does not render result state without a pending reward', () => {
    const incompleteStatus: BoxStatusResponse = {
      round_number: 1,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'result',
      opened_rewards: [],
    }

    render(RingSummonFlow, {
      props: {
        status: incompleteStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(document.querySelector('.summon-overlay')).toBeNull()
  })

  test.each([
    [
      1205251,
      '月亮游记戒指',
      '/image/activity/rings/ring-1205251-moon.png',
      '/image/activity/rings/ring-1205251-moon.svg',
    ],
    [
      1205470,
      '爱意翩跹戒指',
      '/image/activity/rings/ring-1205470-love.png',
      '/image/activity/rings/ring-1205470-love.svg',
    ],
  ])('uses the animated Figma result popup for ring %s', (itemId, itemName, ringImage, animatedImage) => {
    const resultStatus: BoxStatusResponse = {
      round_number: 1,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'result',
      pending_reward: {
        box_id: 9,
        chance_index: 1,
        item_id: itemId,
        item_name: itemName,
        is_choice: false,
      },
      opened_rewards: [],
    }

    render(RingSummonFlow, {
      props: {
        status: resultStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    const dialog = screen.getByRole('dialog', { name: '戒指已回应召唤' })
    expect(dialog.querySelector('.summon-result')).toHaveClass('popup-center-expand')
    expect(dialog.querySelector('img[src="/image/activity/ring-summon/result/frame.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/ring-summon/result/reward-card.png"]')).not.toBeNull()
    expect(dialog.querySelector(`source[srcset="${animatedImage}"]`)).not.toBeNull()
    expect(within(dialog).getByAltText(itemName)).toHaveAttribute('src', ringImage)
    expect(within(dialog).getByText(itemName)).toHaveTextContent('×1')
    expect(within(dialog).getByRole('button', { name: '确认收下' }).querySelector(
      'img[src="/image/activity/ring-summon/result/confirm.png"]',
    )).not.toBeNull()
  })

  test('plays the ring-box sound once when an available chance appears', async () => {
    const readyStatus: BoxStatusResponse = {
      round_number: 1,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'ready',
      next_chance_index: 1,
      opened_rewards: [],
    }

    const view = render(RingSummonFlow, {
      props: {
        status: readyStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    expect(playActivityEffect).toHaveBeenCalledTimes(1)
    expect(playActivityEffect).toHaveBeenCalledWith('ringBoxReady')

    const dialog = screen.getByRole('dialog', { name: '六朵花已全部点亮' })
    expect(dialog.querySelector('.summon-ready')).toHaveClass('popup-center-expand')
    expect(dialog).toHaveTextContent('恭喜您已点亮6朵花')
    expect(dialog).toHaveTextContent('获得开启第1次戒指宝箱的机会')
    expect(dialog.querySelector('img[src="/image/activity/ring-summon/ready-frame.png"]')).not.toBeNull()
    expect(within(dialog).getByRole('button', { name: '召唤戒指' }).querySelector(
      'img[src="/image/activity/ring-summon/summon-button.png"]',
    )).not.toBeNull()

    await view.rerender({ status: { ...readyStatus }, loadState: 'ready' })
    expect(playActivityEffect).toHaveBeenCalledTimes(1)

    await view.rerender({
      status: { ...readyStatus, box_chances: 2, next_chance_index: 2 },
      loadState: 'ready',
    })
    expect(playActivityEffect).toHaveBeenCalledTimes(2)
  })

  test('uses the fixed names and cuts for all true-love choices', () => {
    const choiceStatus: BoxStatusResponse = {
      round_number: 1,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'choice',
      pending_reward: {
        box_id: 9,
        chance_index: 1,
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        is_choice: true,
        choice_ids: [1207751, 1207752, 1207753],
      },
      opened_rewards: [],
    }

    render(RingSummonFlow, {
      props: {
        status: choiceStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    const dialog = screen.getByRole('dialog', { name: '选择一枚戒指' })
    expect(dialog.querySelector('.true-love-choice')).toHaveClass('popup-center-expand')
    const mappings = [
      [1207751, '真爱无敌戒指经典款', '/image/activity/true-love-choice/classic.png'],
      [1207752, '真爱无敌戒指铭文款', '/image/activity/true-love-choice/true-love-inscription.png'],
      [1207753, '爱你万年戒指铭文款', '/image/activity/true-love-choice/love-forever-inscription.png'],
    ] as const

    for (const [itemId, name, image] of mappings) {
      const choice = within(dialog).getByRole('radio', { name: `${name}，道具ID ${itemId}` })
      expect(choice.querySelector('img')).toHaveAttribute('src', image)
    }
  })

  test('returns from final confirmation without losing the selected ring', async () => {
    const user = userEvent.setup()
    const choiceStatus: BoxStatusResponse = {
      round_number: 1,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'choice',
      pending_reward: {
        box_id: 9,
        chance_index: 1,
        item_id: 1207751,
        item_name: '真爱无敌戒指选择权',
        is_choice: true,
        choice_ids: [1207751, 1207752, 1207753],
      },
      opened_rewards: [],
    }

    const view = render(RingSummonFlow, {
      props: {
        status: choiceStatus,
        loadState: 'ready',
        opening: false,
        confirming: false,
      },
    })

    const selectPanel = document.querySelector('.true-love-choice')
    const selectedRing = screen.getByRole('radio', { name: /道具ID 1207753/ })
    await user.click(selectedRing)
    await user.click(screen.getByRole('button', { name: '确定所选戒指' }))

    expect(screen.getByRole('dialog', { name: '确认选择戒指' })).toBeInTheDocument()
    expect(document.querySelector('.true-love-choice')).not.toBe(selectPanel)
    expect(document.querySelector('.true-love-choice')).toHaveClass('popup-center-expand')
    expect(screen.getByText(/爱你万年戒指铭文款/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '再想想' }))

    expect(screen.getByRole('dialog', { name: '选择一枚戒指' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /道具ID 1207753/ })).toHaveAttribute('aria-checked', 'true')
    expect(view.emitted('confirm')).toBeUndefined()

    await user.click(screen.getByRole('button', { name: '确定所选戒指' }))
    await user.click(screen.getByRole('button', { name: '确认选择戒指' }))
    expect(view.emitted('confirm')).toEqual([[1207753]])
  })
})
