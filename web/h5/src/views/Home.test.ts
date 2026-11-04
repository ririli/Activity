import { fireEvent, render, screen, waitFor, within } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { vi } from 'vitest'
import Home from './Home.vue'
import { apiState } from '@/test/server'
import { createDrawAnimationTransform, toCssMatrix } from '@/utils/drawAnimationTransform'

function createDomRect(left: number, top: number, width: number, height: number): DOMRect {
  return {
    x: left,
    y: top,
    top,
    right: left + width,
    bottom: top + height,
    left,
    width,
    height,
    toJSON: () => ({}),
  } as DOMRect
}

async function renderHome({ firstVisit = false, keepTryBeforePayOffer = false } = {}) {
  localStorage.setItem('lottery_access_token', 'test-token')
  if (!firstVisit) localStorage.setItem('lottery_new_ring_benefit_seen:1', 'true')
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'Home', component: Home },
      { path: '/login', name: 'Login', component: { template: '<div>登录页</div>' } },
      { path: '/leaderboard', name: 'Leaderboard', component: { template: '<div>排行榜页面</div>' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const result = render(Home, {
    global: {
      plugins: [createPinia(), router],
    },
  })
  if (apiState.activityStatus === 1 && apiState.tbpAvailable && !apiState.tbpActiveSession && !keepTryBeforePayOffer) {
    const offer = await screen.findByRole('dialog', { name: '先抽后付机会' })
    await fireEvent.click(within(offer).getByRole('button', { name: '再想想' }))
  }
  return { ...result, router }
}

describe('Home activity page', () => {
  test('shows the ended prompt and disables all draw buttons after the activity ends', async () => {
    const user = userEvent.setup()
    apiState.activityStatus = 2
    apiState.boxChances = 1
    apiState.boxFlowState = 'ready'
    await renderHome()

    await waitFor(() => expect(document.querySelector('.van-toast')).toHaveTextContent('活动已结束'))
    expect(document.querySelector('.van-dialog')).not.toBeInTheDocument()
    expect(document.querySelector('.draw-dock__ended')).toHaveTextContent('活动已结束')
    expect(screen.getByRole('button', { name: /许愿1次/ })).toBeDisabled()
    expect(screen.getByRole('button', { name: /许愿10次/ })).toBeDisabled()
    expect(screen.getByRole('button', { name: /许愿30次/ })).toBeDisabled()
    expect(screen.getByRole('button', { name: '购买花瓣' })).toBeDisabled()

    await waitFor(() => expect(document.querySelector('main.activity-page')).not.toHaveAttribute('inert'))

    await user.click(screen.getByRole('button', { name: '我的奖励' }))
    let recordsDialog = await screen.findByRole('dialog', { name: '我的奖励' })
    expect(within(recordsDialog).queryByRole('button', { name: '关闭我的奖励' })).not.toBeInTheDocument()
    await fireEvent.click(recordsDialog.parentElement as HTMLElement)
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '我的奖励' })).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: '抽奖记录' }))
    recordsDialog = await screen.findByRole('dialog', { name: '抽奖记录' })
    await user.click(within(recordsDialog).getByRole('button', { name: '关闭抽奖记录' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '抽奖记录' })).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: '戒指召唤记录' }))
    recordsDialog = await screen.findByRole('dialog', { name: '戒指召唤记录' })
    await user.click(within(recordsDialog).getByRole('button', { name: '关闭戒指召唤记录' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '戒指召唤记录' })).not.toBeInTheDocument())
  })

  test('opens my rewards with the supplied cuts and keeps three true-love variants separate', async () => {
    const user = userEvent.setup()
    await renderHome()

    const button = screen.getByRole('button', { name: '我的奖励' })
    expect(button.querySelector('img')).toHaveAttribute(
      'src',
      '/image/activity/my-rewards/button-bg.png',
    )

    await user.click(button)

    const dialog = await screen.findByRole('dialog', { name: '我的奖励' })
    expect(dialog).toHaveClass('popup-center-expand')
    expect(document.querySelector('main.activity-page')).toHaveAttribute('inert')
    expect(dialog.querySelector('img[src="/image/activity/my-rewards/popup-frame.png"]')).not.toBeNull()
    const rewardItems = within(dialog).getAllByRole('listitem')
    expect(rewardItems[0]).toHaveTextContent('金币')
    expect(rewardItems.at(-1)).toHaveTextContent('真爱无敌戒指经典款')

    const expectedTrueLoveRewards = [
      ['真爱无敌戒指经典款', '/image/activity/rings/ring-1207751-true-love-classic.png'],
      ['真爱无敌戒指铭文款', '/image/activity/rings/ring-1207752-true-love-inscription.png'],
      ['爱你万年戒指铭文款', '/image/activity/rings/ring-1207753-love-forever-inscription.png'],
    ] as const
    for (const [name, image] of expectedTrueLoveRewards) {
      const tile = within(dialog).getByText(name).closest('.reward-preview-tile')
      expect(tile).not.toBeNull()
      expect(tile?.querySelector(`img[src="${image}"]`)).not.toBeNull()
    }

    const expectedPrincessRewards = [
      ['春日公主称号', '/image/leaderboard/rewards/spring-princess-title.png', 'reward-preview-tile__image--title'],
      ['春日公主头像框', '/image/leaderboard/rewards/spring-princess-frame.png', 'reward-preview-tile__image--avatar-frame'],
      ['春日公主冠名头像框', '/image/leaderboard/rewards/spring-princess-naming.png', 'reward-preview-tile__image--avatar-frame'],
    ] as const
    for (const [name, image, className] of expectedPrincessRewards) {
      const tile = within(dialog).getByText(name).closest('.reward-preview-tile')
      const rewardImage = tile?.querySelector(`img[src="${image}"]`)
      expect(rewardImage).not.toBeNull()
      expect(rewardImage).toHaveClass(className)
    }
    const customTitleTile = within(dialog).getByText('定制称号/修改V8称号').closest('.reward-preview-tile')
    const customTitleImage = customTitleTile?.querySelector(
      'img[src="/image/leaderboard/rewards/custom-title.png"]',
    )
    expect(customTitleImage).not.toBeNull()
    expect(customTitleImage).toHaveClass('reward-preview-tile__image--custom-title')
    expect(within(dialog).queryByText(/春日王子/)).not.toBeInTheDocument()
    expect(within(dialog).getByText('真爱无敌戒指经典款').closest('.reward-preview-tile')).toHaveTextContent('x2')

    expect(within(dialog).queryByRole('button', { name: '关闭我的奖励' })).not.toBeInTheDocument()
    await fireEvent.click(dialog.parentElement as HTMLElement)
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '我的奖励' })).not.toBeInTheDocument())
    expect(document.querySelector('main.activity-page')).not.toHaveAttribute('inert')
  })

  test('shows no history on connect, then plays realtime marquee pushes without replaying records', async () => {
    const { unmount } = await renderHome()

    // 连接建立后不补发历史：首屏不应出现弹幕
    await screen.findByText('白昼花园')
    expect(screen.queryByRole('region', { name: '中奖播报' })).not.toBeInTheDocument()

    // 注入弹幕并触发重连，模拟连接后到达的实时推送
    apiState.marqueeItems = [
      {
        id: 301,
        nickname: '测试用户',
        item_id: 1205251,
        item_name: '月亮游记戒指',
        obtained_at: '2026-07-13T10:00:05+08:00',
      },
      {
        id: 302,
        nickname: '花间听雨',
        item_id: 1205470,
        item_name: '爱意翩跹戒指',
        obtained_at: '2026-07-13T10:00:06+08:00',
      },
    ]
    await fireEvent.click(screen.getByRole('button', { name: '刷新活动数据' }))

    let marquee = await screen.findByRole('region', { name: '中奖播报' })
    expect(within(marquee).getByText('恭喜 测试用户 获得 月亮游记戒指')).toBeInTheDocument()
    expect(document.querySelector('.marquee-slot')).not.toBeNull()

    await fireEvent.animationEnd(marquee.querySelector('.marquee__track')!)
    marquee = screen.getByRole('region', { name: '中奖播报' })
    expect(within(marquee).getByText('恭喜 花间听雨 获得 爱意翩跹戒指')).toBeInTheDocument()
    await fireEvent.animationEnd(marquee.querySelector('.marquee__track')!)
    expect(screen.queryByRole('region', { name: '中奖播报' })).not.toBeInTheDocument()

    apiState.marqueeItems = [
      ...apiState.marqueeItems,
      {
        id: 303,
        nickname: '星河入梦',
        item_id: 1207244,
        item_name: '烟花之恋戒指',
        obtained_at: '2026-07-13T10:00:07+08:00',
      },
      {
        id: 304,
        nickname: '花海漫游',
        item_id: 1203743,
        item_name: '闪闪心蝶戒指',
        obtained_at: '2026-07-13T10:00:08+08:00',
      },
    ]
    await fireEvent.click(screen.getByRole('button', { name: '刷新活动数据' }))

    marquee = await screen.findByRole('region', { name: '中奖播报' })
    expect(within(marquee).getByText('恭喜 星河入梦 获得 烟花之恋戒指')).toBeInTheDocument()
    expect(within(marquee).queryByText('恭喜 花海漫游 获得 闪闪心蝶戒指')).not.toBeInTheDocument()

    await fireEvent.animationEnd(marquee.querySelector('.marquee__track')!)
    marquee = screen.getByRole('region', { name: '中奖播报' })
    expect(within(marquee).getByText('恭喜 花海漫游 获得 闪闪心蝶戒指')).toBeInTheDocument()

    await fireEvent.animationEnd(marquee.querySelector('.marquee__track')!)
    expect(screen.queryByRole('region', { name: '中奖播报' })).not.toBeInTheDocument()
    expect(document.querySelector('.marquee-slot')).not.toBeNull()

    await fireEvent.click(screen.getByRole('button', { name: '刷新活动数据' }))
    await waitFor(() => expect(apiState.marqueeRequestCount).toBeGreaterThan(1))
    expect(screen.queryByRole('region', { name: '中奖播报' })).not.toBeInTheDocument()

    unmount()
    await renderHome()
    await screen.findByText('白昼花园')
    expect(screen.queryByRole('region', { name: '中奖播报' })).not.toBeInTheDocument()
  })

  test('uses the night marquee color scheme', async () => {
    await renderHome()

    apiState.marqueeItems = [{
      id: 401,
      nickname: '星河入梦',
      item_id: 1207244,
      item_name: '烟花之恋戒指',
      obtained_at: '2026-07-13T10:01:00+08:00',
    }]
    await fireEvent.click(screen.getByRole('button', { name: '刷新活动数据' }))

    await fireEvent.click(screen.getByRole('tab', { name: '星夜许愿' }))
    expect(await screen.findByRole('region', { name: '中奖播报' })).toHaveClass('marquee--night')
  })

  test('renders all six lit flower overlays on the day chest', async () => {
    apiState.currentLitCount = 6
    await renderHome()

    const showcase = await screen.findByRole('region', { name: '戒指奖励展示' })
    const flowerBox = await within(showcase).findByLabelText('花盒当前点亮6朵花')
    expect(flowerBox.querySelector('img.flower-box__image')).toHaveAttribute(
      'src',
      '/image/activity/flowers/flower-box-day.png',
    )
    expect(flowerBox.querySelectorAll('.flower-box__rose')).toHaveLength(6)
  })

  test('restores an unfinished summon and keeps it open until the result is confirmed', async () => {
    apiState.currentLitCount = 6
    apiState.boxChances = 1
    apiState.boxFlowState = 'ready'
    const user = userEvent.setup()
    await renderHome()

    const summonDialog = await screen.findByRole('dialog', { name: '六朵花已全部点亮' })
    expect(document.querySelector('main.activity-page')).toHaveAttribute('inert')
    expect(within(summonDialog).queryByRole('button', { name: /关闭/ })).not.toBeInTheDocument()
    await fireEvent.keyDown(summonDialog, { key: 'Escape' })
    expect(screen.getByRole('dialog', { name: '六朵花已全部点亮' })).toBeInTheDocument()

    await user.click(within(summonDialog).getByRole('button', { name: '召唤戒指' }))

    const resultDialog = await screen.findByRole('dialog', { name: '戒指已回应召唤' })
    expect(within(resultDialog).getByText('月亮游记戒指')).toBeInTheDocument()
    expect(apiState.boxOpenCount).toBe(1)

    await user.click(within(resultDialog).getByRole('button', { name: '确认收下' }))

    await waitFor(() => expect(apiState.boxConfirmCount).toBe(1))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '戒指已回应召唤' })).not.toBeInTheDocument())
    expect(document.querySelector('main.activity-page')).not.toHaveAttribute('inert')
  })

  test('restores the true-love choice popup and confirms one of three rings', async () => {
    apiState.currentLitCount = 6
    apiState.boxChances = 1
    apiState.boxFlowState = 'choice'
    apiState.boxPendingReward = {
      box_id: 9,
      chance_index: 1,
      item_id: 1207751,
      item_name: '真爱无敌戒指选择权',
      is_choice: true,
      choice_ids: [1207751, 1207752, 1207753],
    }
    const user = userEvent.setup()
    await renderHome()

    const dialog = await screen.findByRole('dialog', { name: '选择一枚戒指' })
    const confirmButton = within(dialog).getByRole('button', { name: '确定所选戒指' })
    expect(confirmButton).toBeDisabled()
    expect(within(dialog).getAllByRole('radio')).toHaveLength(3)

    await user.click(within(dialog).getByRole('radio', { name: /道具ID 1207753/ }))
    expect(confirmButton).toBeEnabled()
    await user.click(confirmButton)
    await user.click(within(dialog).getByRole('button', { name: '确认选择戒指' }))

    await waitFor(() => expect(apiState.boxConfirmedItemId).toBe(1207753))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '选择一枚戒指' })).not.toBeInTheDocument())
    const acquiredDialog = await screen.findByRole('dialog', { name: '恭喜获得真爱无敌戒指' })
    expect(acquiredDialog.querySelector(
      'img[src="/image/activity/true-love-acquired/frame-love-forever-inscription.png"]',
    )).not.toBeNull()
    await user.click(within(acquiredDialog).getByRole('button', { name: '确定' }))
  })

  test('keeps the activity usable when box status has no flow state', async () => {
    apiState.boxFlowState = undefined as unknown as typeof apiState.boxFlowState
    await renderHome()

    await screen.findByText('白昼花园')
    await waitFor(() => expect(document.querySelector('main.activity-page')).not.toHaveAttribute('inert'))
    expect(document.querySelector('.summon-overlay')).toBeNull()
  })

  test('loads balances and switches between day and night pools', async () => {
    const user = userEvent.setup()
    await renderHome()

    expect(await screen.findByText('白昼花园')).toBeInTheDocument()
    expect(document.querySelector('.draw-flight-animation')).toBeNull()
    expect(document.querySelector('.activity-scene')).not.toHaveAttribute('style')
    expect(document.querySelector('.activity-scroll')).toHaveStyle({
      '--activity-background': "url('/image/background1.png')",
    })
    expect(screen.getByLabelText('当前花瓣余额')).toHaveTextContent('300')
    expect(screen.getByRole('tab', { name: '白昼许愿' })).toHaveAttribute('aria-selected', 'true')
    expect(await screen.findByRole('button', { name: /先抽后付/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '抽奖记录' }).closest('.draw-dock')).not.toBeNull()
    expect(document.querySelector('img[src="/image/activity/guide/guide-day.png"]')).not.toBeNull()
    expect(screen.getByRole('button', { name: '查看全部奖励' }).querySelector('img')).toHaveAttribute(
      'src',
      '/image/activity/guide/all-rewards-day.png',
    )
    const showcase = screen.getByRole('region', { name: '戒指奖励展示' })
    expect(showcase.querySelectorAll('img[src="/image/activity/ring-orbs/day.png"]')).toHaveLength(3)
    expect(showcase.querySelector('img[src="/image/activity/ring-orbs/new.png"]')).not.toBeNull()

    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))

    expect(screen.getByText('星夜花园')).toBeInTheDocument()
    expect(document.querySelector('.activity-scroll')).toHaveStyle({
      '--activity-background': "url('/image/background2.png')",
    })
    expect(screen.getByRole('tab', { name: '星夜许愿' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('button', { name: /许愿1次/ })).toHaveTextContent('8')
    expect(screen.queryByRole('button', { name: /先抽后付/ })).not.toBeInTheDocument()
    expect(showcase.querySelectorAll('img[src="/image/activity/ring-orbs/night.png"]')).toHaveLength(3)
    expect(showcase.querySelectorAll('.ring-card')).toHaveLength(3)
  })

  test('uses the complete night chest cut, yellow ring orbs, and night guide assets', async () => {
    apiState.currentLitCount = 0
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))

    const showcase = screen.getByRole('region', { name: '戒指奖励展示' })
    const flowerBox = within(showcase).getByLabelText('花盒当前点亮0朵花')
    expect(flowerBox.querySelector('img.flower-box__image')).toHaveAttribute(
      'src',
      '/image/activity/flowers/flower-box-night.png',
    )
    expect(showcase.querySelectorAll('img[src="/image/activity/ring-orbs/night.png"]')).toHaveLength(3)
    expect(showcase.querySelector('img[src="/image/activity/night-showcase/night-ring-swan.png"]')).toBeNull()
    expect(within(showcase).getByText('月亮游记戒指')).toBeInTheDocument()
    expect(within(showcase).getByText('真爱无敌戒指')).toBeInTheDocument()
    expect(within(showcase).getByText('爱意翩跹戒指')).toBeInTheDocument()
    expect(showcase.querySelector('img[src="/image/activity/rings/ring-1205251-moon.png"]')).not.toBeNull()
    expect(showcase.querySelector('img[src="/image/activity/rings/ring-1207751-true-love-classic.png"]')).not.toBeNull()
    expect(showcase.querySelector('img[src="/image/activity/rings/ring-1205470-love.png"]')).not.toBeNull()
    expect(showcase.querySelector('source[srcset="/image/activity/rings/ring-1205251-moon.svg"]')).not.toBeNull()
    expect(showcase.querySelector('source[srcset="/image/activity/rings/ring-1207751-true-love-classic.svg"]')).not.toBeNull()
    expect(showcase.querySelector('source[srcset="/image/activity/rings/ring-1205470-love.svg"]')).not.toBeNull()

    expect(document.querySelector('img[src="/image/activity/guide/guide-night.png"]')).not.toBeNull()
    expect(screen.getByRole('button', { name: '查看全部奖励' }).querySelector('img')).toHaveAttribute(
      'src',
      '/image/activity/guide/all-rewards-night.png',
    )
    expect(document.querySelector('img[src="/image/activity/guide/all-rewards-day.png"]')).toBeNull()
  })

  test('marks treasure-box rewards as obtained from the backend box status', async () => {
    apiState.boxChances = 2
    apiState.boxOpened = 2
    apiState.boxOpenedRewards = [
      { chance_index: 1, item_id: 1205251, item_name: '月亮游记戒指', is_choice: false },
      { chance_index: 2, item_id: 1207752, item_name: '真爱无敌戒指铭文款', is_choice: true },
    ]
    await renderHome()

    await screen.findByText('白昼花园')
    const showcase = screen.getByRole('region', { name: '戒指奖励展示' })
    const moon = within(showcase).getByLabelText('月亮游记戒指')
    const trueLove = within(showcase).getByLabelText('真爱无敌戒指')
    const love = within(showcase).getByLabelText('爱意翩跹戒指')

    await waitFor(() => {
      expect(moon.querySelector('img[src="/image/activity/ring-orbs/obtained.png"]')).not.toBeNull()
      expect(trueLove.querySelector('img[src="/image/activity/ring-orbs/obtained.png"]')).not.toBeNull()
    })
    expect(love.querySelector('img[src="/image/activity/ring-orbs/obtained.png"]')).toBeNull()
  })

  test('opens all rewards from pool data and switches between day and night prize categories', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: '查看全部奖励' }))

    const dialog = await screen.findByRole('dialog', { name: '查看全部奖励' })
    expect(within(dialog).queryByRole('button', { name: '关闭全部奖励' })).not.toBeInTheDocument()
    expect(dialog.querySelector('img[src="/image/activity/all-rewards-popup/frame.png"]')).not.toBeNull()
    expect(dialog.querySelectorAll('img[src="/image/activity/results/reward-item-base.png"]')).toHaveLength(
      dialog.querySelectorAll('.reward-preview-tile').length,
    )
    expect(within(dialog).getByRole('tab', { name: '白昼之旅' })).toHaveAttribute('aria-selected', 'true')
    expect(within(dialog).getByRole('heading', { name: '真爱无敌' })).toBeInTheDocument()
    expect(within(dialog).getByRole('heading', { name: '其他戒指' })).toBeInTheDocument()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1207244-firework.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1203743-butterfly.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1205251-moon.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1207751-true-love-classic.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1207752-true-love-inscription.png"]')).not.toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1207753-love-forever-inscription.png"]')).not.toBeNull()
    expect(dialog.querySelectorAll('img[src="/image/activity/all-rewards-popup/new-badge.png"]')).toHaveLength(3)
    expect(within(dialog).getByText('真爱无敌戒指经典款')).toBeInTheDocument()
    expect(within(dialog).getByText('爱你万年戒指铭文款')).toBeInTheDocument()
    const dayItems = within(dialog).getByRole('heading', { name: '其他物品' }).closest('section')
    expect(dayItems).not.toBeNull()
    expect(within(dayItems as HTMLElement).getByText('x100')).toBeInTheDocument()
    expect(within(dialog).queryByText('爱意翩跹戒指')).not.toBeInTheDocument()

    await user.click(within(dialog).getByRole('tab', { name: '星夜之旅' }))

    expect(within(dialog).getByRole('tab', { name: '星夜之旅' })).toHaveAttribute('aria-selected', 'true')
    expect(within(dialog).getByText('爱意翩跹戒指')).toBeInTheDocument()
    expect(dialog.querySelector('img[src="/image/activity/rings/ring-1205470-love.png"]')).not.toBeNull()
    const nightItems = within(dialog).getByRole('heading', { name: '其他物品' }).closest('section')
    expect(nightItems).not.toBeNull()
    expect(within(nightItems as HTMLElement).getByText('x1,000')).toBeInTheDocument()
    expect(dialog.querySelector('.all-rewards-scroll')).not.toBeNull()

    await fireEvent.click(dialog.parentElement as HTMLElement)
    expect(screen.queryByRole('dialog', { name: '查看全部奖励' })).not.toBeInTheDocument()
  })

  test('renders all six lit flower overlays on the night chest', async () => {
    apiState.currentLitCount = 6
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))

    const showcase = screen.getByRole('region', { name: '戒指奖励展示' })
    const flowerBox = within(showcase).getByLabelText('花盒当前点亮6朵花')
    expect(flowerBox.querySelector('img.flower-box__image')).toHaveAttribute(
      'src',
      '/image/activity/flowers/flower-box-night.png',
    )
    const sources = within(flowerBox).getAllByAltText(/^已点亮的第\d朵花$/).map((image) => image.getAttribute('src'))
    expect(sources).toEqual([
      '/image/activity/flowers/rose-bottom-left.png',
      '/image/activity/flowers/rose-bottom-center.png',
      '/image/activity/flowers/rose-bottom-right.png',
      '/image/activity/flowers/rose-top-left.png',
      '/image/activity/flowers/rose-top-center.png',
      '/image/activity/flowers/rose-top-right.png',
    ])
  })

  test('keeps day and night guide and chest layout dimensions consistent', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    const dayGuide = document.querySelector('.guide-section')
    const dayBox = document.querySelector('.flower-box')
    expect(dayGuide).not.toBeNull()
    expect(dayBox).not.toBeNull()
    const dayGuideHeight = getComputedStyle(dayGuide as Element).height
    const dayBoxAspectRatio = getComputedStyle(dayBox as Element).aspectRatio

    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))

    expect(document.querySelector('.guide-section')).toBe(dayGuide)
    expect(document.querySelector('.flower-box')).toBe(dayBox)
    expect(dayBox).toHaveClass('flower-box--night')
    expect(getComputedStyle(dayGuide as Element).height).toBe(dayGuideHeight)
    expect(getComputedStyle(dayBox as Element).aspectRatio).toBe(dayBoxAspectRatio)
  })

  test('opens the purchase confirmation when petals are insufficient', async () => {
    apiState.petals = 2
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: /许愿1次/ }))

    const dialog = await screen.findByRole('dialog', { name: '确认购买花瓣' })
    expect(dialog).toHaveTextContent('确认花费180金币')
    expect(dialog).toHaveTextContent('购买3枚花瓣吗？')
    expect(dialog.querySelector('img[src="/image/activity/purchase-confirm/frame.png"]')).not.toBeNull()
    expect(within(dialog).getByRole('checkbox', { name: '本次不再提示' })).not.toBeChecked()
    expect(within(dialog).getByRole('button', { name: '取消购买花瓣' }).querySelector(
      'img[src="/image/activity/purchase-confirm/cancel.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByRole('button', { name: '确认购买花瓣并许愿' }).querySelector(
      'img[src="/image/activity/purchase-confirm/confirm.png"]',
    )).not.toBeNull()
  })

  test('uses the fixed 1-to-60 rate when the draw shortage matches a shop bonus tier', async () => {
    apiState.petals = 28
    apiState.gold = 10_000
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('tab', { name: '星夜许愿' }))
    await user.click(screen.getByRole('button', { name: /许愿10次/ }))

    const dialog = await screen.findByRole('dialog', { name: '确认购买花瓣' })
    expect(dialog).toHaveTextContent('确认花费3,120金币')
    expect(dialog).toHaveTextContent('购买52枚花瓣吗？')
    await user.click(within(dialog).getByRole('button', { name: '确认购买花瓣并许愿' }))

    await waitFor(() => expect(apiState.drawCount).toBe(10))
    expect(apiState.gold).toBe(6_880)
  })

  test('shows a gold warning before opening the petal shop for a normal draw', async () => {
    apiState.petals = 0
    apiState.gold = 0
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿1次/ }))

    expect(await screen.findByText('金币不足～')).toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '确认购买花瓣' })).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '获取花瓣' })).not.toBeInTheDocument()
    expect(await screen.findByRole('dialog', { name: '获取花瓣' }, { timeout: 1800 })).toBeInTheDocument()
  })

  test('disables later purchase confirmations when requested in the popup', async () => {
    apiState.petals = 2
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    const purchaseConfirmationSetting = screen.getByRole('checkbox', { name: '显示购买确认' })
    expect(purchaseConfirmationSetting).toBeChecked()

    await user.click(screen.getByRole('button', { name: /许愿1次/ }))
    const dialog = await screen.findByRole('dialog', { name: '确认购买花瓣' })
    await user.click(within(dialog).getByRole('checkbox', { name: '本次不再提示' }))
    await user.click(within(dialog).getByRole('button', { name: '确认购买花瓣并许愿' }))

    await waitFor(() => expect(purchaseConfirmationSetting).not.toBeChecked())
  })

  test('opens petal shop with Figma tiers and daily tasks', async () => {
    apiState.gold = 10_000_000
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: '购买花瓣' }))

    const dialog = await screen.findByRole('dialog', { name: '获取花瓣' })
    const goldBalance = within(dialog).getByLabelText('当前金币余额')
    expect(goldBalance).toHaveTextContent('10,000,000')
    expect(goldBalance.querySelector('strong')).toHaveClass('psp-asset__value--long')
    expect(within(dialog).getByRole('button', { name: '购买5枚花瓣，消耗300金币' })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '购买52枚花瓣，消耗3,000金币' })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '购买525枚花瓣，消耗30,000金币' })).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '购买1,580枚花瓣，消耗90,000金币' })).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: '购买5枚花瓣，消耗300金币' }))
    expect(await screen.findByText('花瓣购买成功，快去许愿吧！')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('当前花瓣余额')).toHaveTextContent('305')

    const bonusBuyButton = within(dialog).getByRole('button', { name: '购买52枚花瓣，消耗3,000金币' })
    expect(dialog.querySelector('img[src="/image/activity/petal-shop/bonus-2.png"]')).not.toBeNull()
    await user.click(bonusBuyButton)
    await waitFor(() => expect(bonusBuyButton).toBeDisabled())
    expect(bonusBuyButton).toHaveAttribute('aria-label', '52枚花瓣赠送档位，已购买')
    expect(bonusBuyButton.querySelector('img')).toHaveAttribute('src', '/image/activity/petal-shop/buy-disabled.png')
    expect(dialog.querySelector('img[src="/image/activity/petal-shop/bonus-2.png"]')).toBeNull()

    await user.click(within(dialog).getByRole('tab', { name: '每日任务' }))

    expect(within(dialog).getByText('每日00:00刷新')).toBeInTheDocument()
    expect(within(dialog).getByText('每日登录app')).toBeInTheDocument()
    expect(within(dialog).getByText('在语音派对玩25分钟')).toBeInTheDocument()
    expect(within(dialog).queryByLabelText('当前金币余额')).not.toBeInTheDocument()
  })

  test('restores purchased bonus tiers as disabled without bonus labels', async () => {
    apiState.gold = 10_000_000
    apiState.purchasedPetalShopTiers = [525]
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: '购买花瓣' }))
    const dialog = await screen.findByRole('dialog', { name: '获取花瓣' })
    const purchasedButton = within(dialog).getByRole('button', { name: '525枚花瓣赠送档位，已购买' })

    expect(purchasedButton).toBeDisabled()
    expect(purchasedButton.querySelector('img')).toHaveAttribute('src', '/image/activity/petal-shop/buy-disabled.png')
    expect(dialog.querySelector('img[src="/image/activity/petal-shop/bonus-25.png"]')).toBeNull()
    expect(within(dialog).getByRole('button', { name: '购买52枚花瓣，消耗3,000金币' })).toBeEnabled()
  })

  test('purchases the one-time 30-yuan petal gift and hides its entry', async () => {
    const user = userEvent.setup()
    await renderHome()

    const entry = await screen.findByRole('button', { name: '打开许愿花瓣礼包' })
    await user.click(entry)

    const offerDialog = await screen.findByRole('dialog', { name: '花瓣特惠礼包' })
    expect(offerDialog.querySelector('img[src="/image/activity/petal-gift/offer.png"]')).not.toBeNull()
    expect(within(offerDialog).getByRole('button', { name: '30元购买100枚花瓣礼包' }).querySelector(
      'img[src="/image/activity/petal-gift/buy-30.png"]',
    )).not.toBeNull()
    await user.click(within(offerDialog).getByRole('button', { name: '30元购买100枚花瓣礼包' }))

    await waitFor(() => expect(apiState.petalGiftPurchaseCount).toBe(1))
    expect(apiState.petals).toBe(400)
    const resultDialog = await screen.findByRole('dialog', { name: '恭喜获得花瓣礼包' })
    expect(resultDialog.querySelector('img[src="/image/activity/petal-gift/result.png"]')).not.toBeNull()
    expect(within(resultDialog).getByRole('button', { name: '确定收下100枚花瓣' }).querySelector(
      'img[src="/image/activity/petal-gift/confirm.png"]',
    )).not.toBeNull()
    expect(screen.queryByRole('button', { name: '打开许愿花瓣礼包' })).not.toBeInTheDocument()

    await user.click(within(resultDialog).getByRole('button', { name: '确定收下100枚花瓣' }))
    expect(screen.queryByRole('dialog', { name: '恭喜获得花瓣礼包' })).not.toBeInTheDocument()
  })

  test('does not show the petal gift entry after it was purchased', async () => {
    apiState.petalGiftPurchased = true
    await renderHome()

    await screen.findByText('白昼花园')
    expect(screen.queryByRole('button', { name: '打开许愿花瓣礼包' })).not.toBeInTheDocument()
  })

  test('hides try-before-pay when the backend marks it unavailable', async () => {
    apiState.tbpAvailable = false
    await renderHome()

    expect(await screen.findByText('白昼花园')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /先抽后付/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '先抽后付机会' })).not.toBeInTheDocument()
  })

  test('prompts eligible users and starts the try-before-pay preview from the offer', async () => {
    const user = userEvent.setup()
    await renderHome({ keepTryBeforePayOffer: true })

    const offer = await screen.findByRole('dialog', { name: '先抽后付机会' })
    expect(offer.querySelector('img[src="/image/activity/tbp-offer/frame.png"]')).not.toBeNull()
    expect(within(offer).getByText(/拥有1次白昼许愿先抽后付180次的机会/)).toBeInTheDocument()
    expect(within(offer).getByRole('button', { name: '再想想' }).querySelector(
      'img[src="/image/activity/tbp-offer/rethink.png"]',
    )).not.toBeNull()
    expect(within(offer).getByRole('button', { name: '使用先抽后付' }).querySelector(
      'img[src="/image/activity/tbp-offer/confirm.png"]',
    )).not.toBeNull()

    await user.click(within(offer).getByRole('button', { name: '使用先抽后付' }))

    expect(await screen.findByRole('dialog', { name: '许愿奖励预览' })).toBeInTheDocument()
    expect(apiState.tbpPreviewCount).toBe(1)
  })

  test('restores an active try-before-pay preview after refresh until the user decides', async () => {
    const user = userEvent.setup()
    const firstRender = await renderHome()

    await user.click(await screen.findByRole('button', { name: /先抽后付/ }))
    expect(await screen.findByRole('dialog', { name: '许愿奖励预览' })).toBeInTheDocument()
    expect(apiState.tbpPreviewCount).toBe(1)

    firstRender.unmount()
    localStorage.removeItem('lottery_new_ring_benefit_seen:1')
    await renderHome({ firstVisit: true })

    const restoredPreview = await screen.findByRole('dialog', { name: '许愿奖励预览' })
    expect(restoredPreview).toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '先抽后付机会' })).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '新戒福利' })).not.toBeInTheDocument()
    expect(apiState.tbpPreviewCount).toBe(1)
    expect(apiState.tbpAbandoned).toBe(false)
    expect(apiState.tbpActiveSession).toBe(true)
  })

  test('previews and confirms try-before-pay for first-time users', async () => {
    apiState.petals = 1_000
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /先抽后付/ }))

    const dialog = await screen.findByRole('dialog', { name: '许愿奖励预览' })
    expect(dialog.querySelector('img[src="/image/activity/tbp-preview/frame.png"]')).not.toBeNull()
    expect(within(dialog).queryByRole('button', { name: '取消' })).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '预览戒指召唤奖励' }).querySelector(
      'img[src="/image/activity/tbp-preview/ring-preview.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByText('月亮游记戒指')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('预计点亮花朵6朵')).toHaveTextContent('x6')
    expect(within(dialog).getAllByRole('listitem')).toHaveLength(3)
    expect(within(dialog).getByLabelText('预计点亮花朵6朵').querySelector(
      'img[src="/image/activity/flowers/rose-bottom-center.png"]',
    )).not.toBeNull()
    expect(apiState.tbpPreviewCount).toBe(1)
    expect(apiState.tbpConfirmed).toBe(false)
    expect(apiState.petals).toBe(1_000)

    await user.click(within(dialog).getByRole('button', { name: '预览戒指召唤奖励' }))
    const ringDialog = await screen.findByRole('dialog', { name: '戒指召唤奖励' })
    const ringRewardsFrame = ringDialog.querySelector('img[src="/image/activity/tbp-preview/ring-rewards-frame.png"]')
    const ringRewardsActions = ringDialog.querySelector('.tbp-actions')
    expect(ringDialog).toHaveClass('tbp-dialog--ring-rewards')
    expect(ringRewardsFrame).not.toBeNull()
    expect(ringRewardsActions).not.toBeNull()
    expect(within(ringDialog).getByRole('button', { name: '取消' }).querySelector(
      'img[src="/image/activity/tbp-preview/cancel.png"]',
    )).not.toBeNull()
    expect(within(ringDialog).getByRole('button', { name: '900花瓣获得以上奖励' }).querySelector(
      'img[src="/image/activity/tbp-preview/confirm.png"]',
    )).not.toBeNull()

    await user.click(within(ringDialog).getByRole('button', { name: '900花瓣获得以上奖励' }))

    await waitFor(() => expect(apiState.tbpConfirmed).toBe(true))
    expect(apiState.petals).toBe(100)
    expect(screen.queryByRole('dialog', { name: '许愿奖励预览' })).not.toBeInTheDocument()
    const flowerDialog = await screen.findByRole('dialog', { name: '恭喜点亮' })
    expect(flowerDialog.querySelector(
      'img[src="/image/activity/flowers/flower-lit-result-frame.png"]',
    )).not.toBeNull()
    expect(flowerDialog).toHaveTextContent('恭喜您本次点亮6朵花')
    await user.click(within(flowerDialog).getByRole('button', { name: '确定' }))
    await waitFor(() => expect(screen.queryByRole('button', { name: /先抽后付/ })).not.toBeInTheDocument())
  })

  test('completes a true-love choice from try-before-pay before continuing the flower result flow', async () => {
    apiState.petals = 1_000
    apiState.tbpChoiceOnConfirm = true
    apiState.tbpEstimatedLitFlowers = 6
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /先抽后付/ }))
    const previewDialog = await screen.findByRole('dialog', { name: '许愿奖励预览' })
    await user.click(within(previewDialog).getByRole('button', { name: '预览戒指召唤奖励' }))
    const rewardDialog = await screen.findByRole('dialog', { name: '戒指召唤奖励' })
    await user.click(within(rewardDialog).getByRole('button', { name: '900花瓣获得以上奖励' }))

    await waitFor(() => expect(apiState.tbpConfirmed).toBe(true))
    const choiceDialog = await screen.findByRole('dialog', { name: '选择一枚戒指' })
    expect(screen.queryByRole('dialog', { name: '恭喜点亮' })).not.toBeInTheDocument()

    await user.click(within(choiceDialog).getByRole('radio', { name: /道具ID 1207753/ }))
    await user.click(within(choiceDialog).getByRole('button', { name: '确定所选戒指' }))
    await user.click(within(choiceDialog).getByRole('button', { name: '确认选择戒指' }))

    await waitFor(() => expect(apiState.drawConfirmedItemIds).toEqual([1207753]))
    const acquiredDialog = await screen.findByRole('dialog', { name: '恭喜获得真爱无敌戒指' })
    expect(acquiredDialog).toHaveTextContent('爱你万年戒指铭文款，道具ID 1207753')
    expect(screen.queryByRole('dialog', { name: '恭喜点亮' })).not.toBeInTheDocument()

    await user.click(within(acquiredDialog).getByRole('button', { name: '确定' }))
    const flowerDialog = await screen.findByRole('dialog', { name: '恭喜点亮' })
    expect(flowerDialog).toHaveTextContent('恭喜您本次点亮6朵花')
  })

  test('shows a gold warning before opening the petal shop when TBP cannot be paid', async () => {
    apiState.petals = 0
    apiState.gold = 0
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /先抽后付/ }))
    const previewDialog = await screen.findByRole('dialog', { name: '许愿奖励预览' })
    const ringPreviewButton = within(previewDialog).queryByRole('button', { name: '预览戒指召唤奖励' })
    if (ringPreviewButton) await user.click(ringPreviewButton)
    const confirmDialog = ringPreviewButton
      ? await screen.findByRole('dialog', { name: '戒指召唤奖励' })
      : previewDialog
    await user.click(within(confirmDialog).getByRole('button', { name: '900花瓣获得以上奖励' }))

    expect(await screen.findByText('金币不足～')).toBeInTheDocument()
    expect(apiState.tbpConfirmed).toBe(false)
    expect(apiState.tbpAbandoned).toBe(false)
    expect(apiState.tbpActiveSession).toBe(true)
    expect(screen.queryByRole('dialog', { name: '许愿奖励预览' })).not.toBeInTheDocument()
    const petalShop = await screen.findByRole('dialog', { name: '获取花瓣' }, { timeout: 1800 })
    await user.click(within(petalShop).getByRole('button', { name: '关闭花瓣购买弹窗' }))

    expect(await screen.findByRole('dialog', { name: '许愿奖励预览' })).toBeInTheDocument()
    expect(apiState.tbpActiveSession).toBe(true)
  })

  test('abandons a try-before-pay preview without granting its rewards', async () => {
    apiState.tbpEstimatedLitFlowers = 5
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /先抽后付/ }))
    const dialog = await screen.findByRole('dialog', { name: '许愿奖励预览' })
    expect(within(dialog).queryByRole('button', { name: '预览戒指召唤奖励' })).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '取消' }).querySelector(
      'img[src="/image/activity/tbp-preview/cancel.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByRole('button', { name: '900花瓣获得以上奖励' }).querySelector(
      'img[src="/image/activity/tbp-preview/confirm.png"]',
    )).not.toBeNull()

    await user.click(within(dialog).getByRole('button', { name: '取消' }))

    const abandonDialog = await screen.findByRole('dialog', { name: '确认放弃本次奖励' })
    expect(abandonDialog.querySelector('img[src="/image/activity/tbp-abandon/frame.png"]')).not.toBeNull()
    expect(within(abandonDialog).getByRole('button', { name: '确认放弃' }).querySelector(
      'img[src="/image/activity/tbp-abandon/confirm.png"]',
    )).not.toBeNull()
    expect(within(abandonDialog).getByRole('button', { name: '不放弃' }).querySelector(
      'img[src="/image/activity/tbp-abandon/keep.png"]',
    )).not.toBeNull()

    await user.click(within(abandonDialog).getByRole('button', { name: '不放弃' }))
    expect(screen.queryByRole('dialog', { name: '确认放弃本次奖励' })).not.toBeInTheDocument()
    expect(screen.getByRole('dialog', { name: '许愿奖励预览' })).toBeInTheDocument()
    expect(apiState.tbpAbandoned).toBe(false)

    await user.click(within(dialog).getByRole('button', { name: '取消' }))
    const confirmDialog = await screen.findByRole('dialog', { name: '确认放弃本次奖励' })
    await user.click(within(confirmDialog).getByRole('button', { name: '确认放弃' }))

    await waitFor(() => expect(apiState.tbpAbandoned).toBe(true))
    expect(apiState.tbpConfirmed).toBe(false)
    expect(apiState.petals).toBe(300)
    expect(screen.queryByRole('dialog', { name: '许愿奖励预览' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /先抽后付/ })).not.toBeInTheDocument()
  })

  test('shows the single-draw result cut and closes it with the confirm cut', async () => {
    apiState.petals = 1_000
    const user = userEvent.setup()
    await renderHome()

    const drawButton = await screen.findByRole('button', { name: /许愿1次/ })
    await waitFor(() => expect(drawButton).toBeEnabled())
    vi.spyOn(drawButton, 'getBoundingClientRect').mockReturnValue(createDomRect(40, 580, 90, 72))
    const flowerTarget = document.querySelector('.flower-box__draw-animation-target') as HTMLElement
    vi.spyOn(flowerTarget, 'getBoundingClientRect').mockReturnValue(createDomRect(242.5, 294.5, 1, 1))
    const activityScroll = document.querySelector('.activity-scroll') as HTMLElement
    Object.defineProperties(activityScroll, {
      scrollHeight: { configurable: true, value: 1200 },
      clientHeight: { configurable: true, value: 500 },
    })
    const scrollTo = vi.fn((options: ScrollToOptions) => {
      activityScroll.scrollTop = Number(options.top ?? 0)
    })
    Object.defineProperty(activityScroll, 'scrollTo', { configurable: true, value: scrollTo })
    await fireEvent.click(drawButton)
    expect(scrollTo).toHaveBeenCalledWith({ top: 700, behavior: 'smooth' })
    await waitFor(() => expect(document.querySelector('.draw-flight-animation')).not.toBeNull())
    const animation = document.querySelector('.draw-flight-animation')
    expect(animation).toHaveAttribute('data-pag-src', '/image/activity/draw/animations/wish-left.pag')
    const expectedTransform = createDrawAnimationTransform(1, { x: 85, y: 616 }, { x: 243, y: 295 })
    expect(expectedTransform).not.toBeNull()
    expect((animation as HTMLElement).style.transform).toBe(toCssMatrix(expectedTransform!))
    expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument()

    const dialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(document.querySelector('.draw-flight-animation')).toBeNull()
    expect(dialog.querySelector('img[src="/image/activity/results/draw-result-single.png"]')).not.toBeNull()
    expect(within(dialog).getAllByRole('listitem')).toHaveLength(2)
    expect(within(dialog).getByText('花瓣')).toBeInTheDocument()
    expect(within(dialog).getByLabelText('本次点亮花朵1朵')).toHaveTextContent('x1')
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm.png"]',
    )).not.toBeNull()

    await user.click(within(dialog).getByRole('button', { name: '确定' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument())
    const flowerDialog = await screen.findByRole('dialog', { name: '恭喜点亮' })
    expect(flowerDialog).toHaveClass('popup-center-expand')
    expect(flowerDialog).toHaveTextContent('恭喜您本次点亮1朵花')
    expect(flowerDialog).toHaveTextContent('当前盒子已点亮花朵数量：5')
    expect(flowerDialog).toHaveTextContent('距离获得戒指宝箱还需点亮1朵花')
    expect(within(flowerDialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/flowers/flower-lit-result-confirm.png"]',
    )).not.toBeNull()
  })

  test('skips the draw animation when the setting is enabled', async () => {
    apiState.petals = 1_000
    apiState.drawLitFlowers = 0
    await renderHome()

    await fireEvent.click(screen.getByRole('checkbox', { name: '跳过动画' }))
    const drawButton = screen.getByRole('button', { name: /许愿10次/ })
    await waitFor(() => expect(drawButton).toBeEnabled())
    await fireEvent.click(drawButton)

    expect(document.querySelector('.draw-flight-animation')).toBeNull()
    expect(await screen.findByRole('dialog', { name: '恭喜获得' })).toBeInTheDocument()
  })

  test('does not show a flower reward or notice when no flower is lit', async () => {
    apiState.drawLitFlowers = 0
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿1次/ }))

    const dialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(within(dialog).getAllByRole('listitem')).toHaveLength(1)
    expect(within(dialog).queryByText('点亮花朵')).not.toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm-pair.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByRole('button', { name: '再抽1次' }).querySelector(
      'img[src="/image/activity/results/redraw-day-1.png"]',
    )).not.toBeNull()
    await user.click(within(dialog).getByRole('button', { name: '确定' }))

    await waitFor(() => expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument())
    expect(screen.queryByRole('dialog', { name: '恭喜点亮' })).not.toBeInTheDocument()
  })

  test('uses the normal draw flow to repeat the same pool and draw count', async () => {
    apiState.petals = 1_000
    apiState.drawLitFlowers = 0
    const user = userEvent.setup()
    await renderHome()

    await user.click(screen.getByRole('checkbox', { name: '跳过动画' }))
    await user.click(screen.getByRole('button', { name: /许愿10次/ }))

    const dialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(apiState.drawRequestCount).toBe(1)
    expect(within(dialog).getByRole('button', { name: '再抽10次' }).querySelector(
      'img[src="/image/activity/results/redraw-day-10.png"]',
    )).not.toBeNull()

    await user.click(within(dialog).getByRole('button', { name: '再抽10次' }))

    await waitFor(() => expect(apiState.drawRequestCount).toBe(2))
    expect(apiState.drawPoolType).toBe(1)
    expect(apiState.drawCount).toBe(10)
    expect(await screen.findByRole('dialog', { name: '恭喜获得' })).toBeInTheDocument()
  })

  test('groups repeated multi-draw rewards and sums their quantities', async () => {
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿30次/ }))

    const dialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(dialog.querySelector('img[src="/image/activity/results/draw-result-multi.png"]')).not.toBeNull()
    expect(within(dialog).getAllByRole('listitem')).toHaveLength(7)
    expect(dialog.querySelectorAll('img[src="/image/activity/results/reward-item-base.png"]')).toHaveLength(7)
    expect(dialog.querySelectorAll('img[src="/image/activity/all-rewards-popup/category-rings.png"]')).toHaveLength(3)
    expect(dialog.querySelectorAll('img[src="/image/activity/all-rewards-popup/category-items.png"]')).toHaveLength(3)
    expect(dialog.querySelector('img[src="/image/activity/all-rewards-popup/category-featured.png"]')).toBeNull()
    expect(within(dialog).getByText('金币').closest('.reward-preview-tile')).toHaveTextContent('x500')
    expect(within(dialog).getByText('花瓣').closest('.reward-preview-tile')).toHaveTextContent('x5')
    const flowerReward = within(dialog).getByLabelText('本次点亮花朵3朵')
    expect(flowerReward).toHaveTextContent('x3')
    expect(flowerReward.querySelector('.draw-result-list__quality-tag')).toBeNull()

    await user.click(within(dialog).getByRole('button', { name: '确定' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument())
  })

  test('shows the round refund popup after the 18th flower and then opens the summon flow', async () => {
    apiState.currentLitCount = 17
    apiState.drawActualCount = 2
    apiState.drawRefundPetals = 40
    apiState.drawLitFlowers = 1
    apiState.drawLitCountAfter = 18
    apiState.boxOpened = 2
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿10次/ }))

    const resultDialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(within(resultDialog).getAllByRole('listitem')).toHaveLength(3)
    expect(screen.queryByRole('dialog', { name: '花瓣退回提示' })).not.toBeInTheDocument()
    await user.click(within(resultDialog).getByRole('button', { name: '确定' }))

    const flowerDialog = await screen.findByRole('dialog', { name: '恭喜点亮' })
    expect(flowerDialog).toHaveTextContent('当前盒子已点亮花朵数量：6')
    expect(flowerDialog).toHaveTextContent('距离获得戒指宝箱还需点亮0朵花')
    expect(screen.queryByRole('dialog', { name: '花瓣退回提示' })).not.toBeInTheDocument()
    await user.click(within(flowerDialog).getByRole('button', { name: '确定' }))

    const refundDialog = await screen.findByRole('dialog', { name: '花瓣退回提示' })
    expect(refundDialog).toHaveClass('popup-center-expand')
    expect(refundDialog.querySelector('img[src="/image/activity/refund/round-refund-frame.png"]')).not.toBeNull()
    expect(within(refundDialog).getByText('40')).toBeInTheDocument()
    expect(refundDialog).toHaveTextContent('本次实际完成2抽')
    expect(screen.queryByRole('dialog', { name: '六朵花已全部点亮' })).not.toBeInTheDocument()
    expect(within(refundDialog).getByRole('button', { name: '确认花瓣退回' }).querySelector(
      'img[src="/image/activity/refund/round-refund-confirm.png"]',
    )).not.toBeNull()

    await user.click(within(refundDialog).getByRole('button', { name: '确认花瓣退回' }))

    expect(await screen.findByRole('dialog', { name: '六朵花已全部点亮' })).toBeInTheDocument()
    expect(screen.queryByRole('dialog', { name: '花瓣退回提示' })).not.toBeInTheDocument()
  })

  test('waits for draw-result confirmation before opening a newly earned ring summon', async () => {
    apiState.boxChances = 1
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿1次/ }))

    const resultDialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(screen.queryByRole('dialog', { name: '六朵花已全部点亮' })).not.toBeInTheDocument()

    await user.click(within(resultDialog).getByRole('button', { name: '确定' }))

    const flowerDialog = await screen.findByRole('dialog', { name: '恭喜点亮' })
    expect(screen.queryByRole('dialog', { name: '六朵花已全部点亮' })).not.toBeInTheDocument()
    await user.click(within(flowerDialog).getByRole('button', { name: '确定' }))

    expect(await screen.findByRole('dialog', { name: '六朵花已全部点亮' })).toBeInTheDocument()
  })

  test('opens and confirms the true-love choice won from a regular draw', async () => {
    apiState.drawChoiceOnNextDraw = true
    apiState.drawLitFlowers = 0
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', { name: /许愿1次/ }))
    const resultDialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(within(resultDialog).getByText('真爱无敌戒指选择权')).toBeInTheDocument()
    expect(resultDialog.querySelector('img[src="/image/activity/all-rewards-popup/category-featured.png"]')).not.toBeNull()
    expect(within(resultDialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/results/draw-result-confirm.png"]',
    )).not.toBeNull()
    expect(within(resultDialog).queryByRole('button', { name: '再抽1次' })).not.toBeInTheDocument()
    await user.click(within(resultDialog).getByRole('button', { name: '确定' }))

    const choiceDialog = await screen.findByRole('dialog', { name: '选择一枚戒指' })
    await user.click(within(choiceDialog).getByRole('radio', { name: /道具ID 1207752/ }))
    await user.click(within(choiceDialog).getByRole('button', { name: '确定所选戒指' }))
    await user.click(within(choiceDialog).getByRole('button', { name: '确认选择戒指' }))

    await waitFor(() => expect(apiState.drawConfirmedItemIds).toEqual([1207752]))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '选择一枚戒指' })).not.toBeInTheDocument())
    const acquiredDialog = await screen.findByRole('dialog', { name: '恭喜获得真爱无敌戒指' })
    expect(acquiredDialog).toHaveTextContent('真爱无敌戒指铭文款，道具ID 1207752')
    expect(within(acquiredDialog).getByText('测试用户')).toBeInTheDocument()
    expect(acquiredDialog.querySelector(
      'img[src="/image/activity/true-love-acquired/frame-true-love-inscription.png"]',
    )).not.toBeNull()
    await user.click(within(acquiredDialog).getByRole('button', { name: '确定' }))
  })

  test('restores an unfinished regular-draw choice after refresh', async () => {
    apiState.drawPendingChoices = [{
      reward_record_id: 902,
      item_id: 1207751,
      item_name: '真爱无敌戒指选择权',
      quantity: 1,
      choice_ids: [1207751, 1207752, 1207753],
    }]
    await renderHome()

    expect(await screen.findByRole('dialog', { name: '选择一枚戒指' })).toBeInTheDocument()
  })

  test('hides try-before-pay after a regular draw creates history', async () => {
    const user = userEvent.setup()
    await renderHome()

    expect(await screen.findByRole('button', { name: /先抽后付/ })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /许愿1次/ }))

    await waitFor(() => expect(apiState.drawCount).toBe(1))
    await waitFor(() => expect(screen.queryByRole('button', { name: /先抽后付/ })).not.toBeInTheDocument())
  })

  test('claims an available stage reward and refreshes its visible status', async () => {
    const user = userEvent.setup()
    await renderHome()

    const claimButton = await screen.findByRole('button', { name: /金币，需要点亮3朵花，可领取/ })
    await user.click(claimButton)

    await waitFor(() => expect(apiState.claimedSlot).toBe(0))
    const claimedButton = await screen.findByRole('button', { name: /金币，需要点亮3朵花，已领取/ })
    expect(claimedButton).toBeDisabled()
    expect(claimedButton.querySelector('img[src="/image/activity/stage/claimed.png"]')).not.toBeNull()

    const dialog = await screen.findByRole('dialog', { name: '恭喜获得' })
    expect(dialog).toHaveClass('popup-center-expand')
    expect(dialog.querySelector(
      'img[src="/image/activity/stage/stage-reward-result-frame.png"]',
    )).not.toBeNull()
    expect(dialog.querySelector(
      'img[src="/image/activity/results/reward-item-base.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByText('金币')).toBeInTheDocument()
    expect(within(dialog).getByText('x500')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: '确定' }).querySelector(
      'img[src="/image/activity/stage/stage-reward-result-confirm.png"]',
    )).not.toBeNull()

    await user.click(within(dialog).getByRole('button', { name: '确定' }))
    expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument()
  })

  test('requires a true-love choice before completing the final stage reward', async () => {
    apiState.currentLitCount = 17
    apiState.rewards = apiState.rewards.map((reward) => ({
      ...reward,
      claimed: reward.slot_index < 5,
      reachable: reward.slot_index === 5,
    }))
    const user = userEvent.setup()
    await renderHome()

    await user.click(await screen.findByRole('button', {
      name: /真爱无敌戒指选择权，需要点亮17朵花，可领取/,
    }))

    const choiceDialog = await screen.findByRole('dialog', { name: '选择一枚戒指' })
    expect(screen.queryByRole('dialog', { name: '恭喜获得' })).not.toBeInTheDocument()
    const loveForever = within(choiceDialog).getByRole('radio', {
      name: '爱你万年戒指铭文款，道具ID 1207753',
    })
    expect(loveForever.querySelector('img')).toHaveAttribute(
      'src',
      '/image/activity/true-love-choice/love-forever-inscription.png',
    )
    await user.click(loveForever)
    await user.click(within(choiceDialog).getByRole('button', { name: '确定所选戒指' }))
    await user.click(within(choiceDialog).getByRole('button', { name: '确认选择戒指' }))

    await waitFor(() => expect(apiState.stageConfirmedItemId).toBe(1207753))
    const resultDialog = await screen.findByRole('dialog', { name: '恭喜获得真爱无敌戒指' })
    expect(resultDialog).toHaveClass('popup-center-expand')
    expect(resultDialog).toHaveTextContent('爱你万年戒指铭文款，道具ID 1207753')
    expect(resultDialog.querySelector(
      'img[src="/image/activity/true-love-acquired/frame-love-forever-inscription.png"]',
    )).not.toBeNull()
    expect(within(resultDialog).getByRole('button', { name: '分享到玩友圈' }).querySelector(
      'img[src="/image/activity/true-love-acquired/share.png"]',
    )).not.toBeNull()
    expect(within(resultDialog).getByRole('button', { name: '相机' }).querySelector(
      'img[src="/image/activity/true-love-acquired/camera.png"]',
    )).not.toBeNull()

    await user.click(within(resultDialog).getByRole('button', { name: '分享到玩友圈' }))
    await user.click(within(resultDialog).getByRole('button', { name: '相机' }))
    expect(screen.getByRole('dialog', { name: '恭喜获得真爱无敌戒指' })).toBeInTheDocument()
  })

  test('restores an unfinished final stage reward choice after refresh', async () => {
    apiState.currentLitCount = 17
    apiState.stagePendingChoice = {
      reward_record_id: 502,
      slot_index: 5,
      item_id: 1207751,
      item_name: '真爱无敌戒指选择权',
      quantity: 1,
      choice_ids: [1207751, 1207752, 1207753],
    }
    apiState.rewards = apiState.rewards.map((reward) => ({
      ...reward,
      claimed: true,
      reachable: false,
    }))

    await renderHome()

    expect(await screen.findByRole('dialog', { name: '选择一枚戒指' })).toBeInTheDocument()
    expect(document.querySelector('main.activity-page')).toHaveAttribute('inert')
  })

  test('shows the new-round cut at 18 flowers and rejects starting while stage rewards remain', async () => {
    apiState.currentLitCount = 18
    apiState.boxChances = 3
    apiState.boxOpened = 3
    apiState.boxFlowState = 'none'
    apiState.rewards = apiState.rewards.map((reward) => ({
      ...reward,
      claimed: false,
      reachable: true,
    }))
    const user = userEvent.setup()
    await renderHome()

    const startButton = await screen.findByRole('button', { name: '开始新一轮' })
    expect(startButton.parentElement?.querySelector(
      'img[src="/image/activity/draw/start-new-round.png"]',
    )).not.toBeNull()
    expect(screen.getByRole('button', { name: /许愿1次/ })).toBeDisabled()

    await user.click(startButton)

    expect(apiState.newRoundRequestCount).toBe(0)
    expect(apiState.newRoundStarted).toBe(false)
    expect(apiState.roundNumber).toBe(1)
    expect(await screen.findByText('请领取花朵累计任务奖励～')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '开始新一轮' })).toBeInTheDocument()
  })

  test('starts a reset second round after all boxes and stage rewards are completed', async () => {
    apiState.currentLitCount = 18
    apiState.boxChances = 3
    apiState.boxOpened = 3
    apiState.boxFlowState = 'none'
    apiState.rewards = apiState.rewards.map((reward) => ({
      ...reward,
      claimed: true,
      reachable: false,
    }))
    const user = userEvent.setup()
    await renderHome()

    await screen.findByRole('button', { name: '开始新一轮' })
    const showcase = await screen.findByRole('region', { name: '戒指奖励展示' })
    expect(within(showcase).getByLabelText('花盒当前点亮6朵花')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '开始新一轮' }))

    await waitFor(() => expect(apiState.newRoundStarted).toBe(true))
    expect(apiState.roundNumber).toBe(2)
    expect(apiState.currentLitCount).toBe(0)
    await waitFor(() => expect(screen.queryByRole('button', { name: '开始新一轮' })).not.toBeInTheDocument())
    expect(screen.getByRole('button', { name: /许愿1次/ })).toBeEnabled()
    expect(screen.getByRole('region', { name: '累计点亮花朵领戒指' })).toHaveTextContent('当前已点亮：0朵')
  })

  test('uses stage reward cut assets for both wish modes and all thresholds', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    const stagePanel = screen.getByRole('region', { name: '累计点亮花朵领戒指' })
    expect(stagePanel.querySelector('img[src="/image/activity/flower-count-bg-day.png"]')).not.toBeNull()
    const petalReward = await within(stagePanel).findByRole('button', { name: '花瓣，需要点亮5朵花，未解锁' })
    expect(petalReward.querySelector('img[src="/image/activity/petal.png"]')).not.toBeNull()
    expect(stagePanel.querySelector('img[src="/image/activity/stage/stage-panel-day.png"]')).not.toBeNull()

    expect(await within(stagePanel).findByAltText('点亮3朵花')).toHaveAttribute(
      'src',
      '/image/activity/stage/stage-threshold-3.png',
    )
    for (const threshold of [3, 5, 8, 11, 15, 17]) {
      expect(within(stagePanel).getByAltText(`点亮${threshold}朵花`)).toHaveAttribute(
        'src',
        `/image/activity/stage/stage-threshold-${threshold}.png`,
      )
    }
    expect(stagePanel.querySelector(
      'img[src="/image/activity/rings/ring-1207244-firework.png"]',
    )).not.toBeNull()
    expect(stagePanel.querySelector(
      'img[src="/image/activity/rings/ring-1205251-moon.png"]',
    )).not.toBeNull()
    expect(stagePanel.querySelector(
      'img[src="/image/activity/rings/ring-1207751-true-love-classic.png"]',
    )).not.toBeNull()
    expect(stagePanel.querySelector(
      'source[srcset="/image/activity/rings/ring-1207244-firework.svg"]',
    )).not.toBeNull()
    expect(stagePanel.querySelector(
      'source[srcset="/image/activity/rings/ring-1205251-moon.svg"]',
    )).not.toBeNull()
    expect(stagePanel.querySelector(
      'source[srcset="/image/activity/rings/ring-1207751-true-love-classic.svg"]',
    )).not.toBeNull()
    expect(stagePanel.querySelectorAll('.reward-item__picture')).toHaveLength(3)

    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))

    expect(stagePanel.querySelector('img[src="/image/activity/stage/stage-panel-night.png"]')).not.toBeNull()
    expect(stagePanel.querySelector('img[src="/image/activity/flower-count-bg-night.png"]')).not.toBeNull()
  })

  test('uses a single threshold button and grays out rewards that are not reached', async () => {
    await renderHome()

    const stagePanel = await screen.findByRole('region', { name: '累计点亮花朵领戒指' })
    const availableButton = await within(stagePanel).findByRole('button', { name: '金币，需要点亮3朵花，可领取' })
    const lockedButton = await within(stagePanel).findByRole('button', { name: '花瓣，需要点亮5朵花，未解锁' })

    expect(availableButton.querySelectorAll('.reward-item__threshold')).toHaveLength(1)
    expect(within(availableButton).queryByAltText('可领取')).not.toBeInTheDocument()
    expect(lockedButton.closest('.reward-item')).toHaveClass('reward-item--locked')
    expect(within(lockedButton).getByAltText('点亮5朵花')).toHaveClass('reward-item__threshold')
  })

  test('shows draw record tabs with reward and flower rows', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: '抽奖记录' }))

    const dialog = await screen.findByRole('dialog', { name: '抽奖记录' })
    expect(within(dialog).getByRole('tab', { name: '白昼之旅' })).toHaveAttribute('aria-selected', 'true')
    expect(within(dialog).getByRole('tab', { name: '星夜之旅' })).toBeInTheDocument()
    expect(within(dialog).getByRole('tab', { name: '点亮花朵' })).toBeInTheDocument()
    expect(await within(dialog).findByText('88元暖心欢乐礼包')).toBeInTheDocument()
    expect(within(dialog).getByText('告白气球语音派对背景1天')).toBeInTheDocument()

    await user.click(within(dialog).getByRole('tab', { name: '点亮花朵' }))

    expect(within(dialog).getByText('花朵x2')).toBeInTheDocument()
    expect(within(dialog).queryByText('花朵x0')).not.toBeInTheDocument()
    expect(within(dialog).queryByText('加载更多')).not.toBeInTheDocument()
  })

  test('defaults draw records to the currently selected wish pool', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: '抽奖记录' }))

    let dialog = await screen.findByRole('dialog', { name: '抽奖记录' })
    expect(within(dialog).getByRole('tab', { name: '白昼之旅' })).toHaveAttribute('aria-selected', 'true')

    await user.click(within(dialog).getByRole('button', { name: '关闭抽奖记录' }))
    await waitFor(() => expect(screen.queryByRole('dialog', { name: '抽奖记录' })).not.toBeInTheDocument())
    await user.click(screen.getByRole('tab', { name: '星夜许愿' }))
    await user.click(screen.getByRole('button', { name: '抽奖记录' }))

    dialog = await screen.findByRole('dialog', { name: '抽奖记录' })
    expect(within(dialog).getByRole('tab', { name: '星夜之旅' })).toHaveAttribute('aria-selected', 'true')
    expect(within(dialog).getByRole('tab', { name: '白昼之旅' })).toHaveAttribute('aria-selected', 'false')
  })

  test('opens ring summon records from the right side of the treasure box', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    const showcase = screen.getByRole('region', { name: '戒指奖励展示' })
    const entry = within(showcase).getByRole('button', { name: '戒指召唤记录' })
    expect(entry).toBeInTheDocument()

    await user.click(entry)

    const dialog = await screen.findByRole('dialog', { name: '戒指召唤记录' })
    expect(dialog.querySelector('img[src="/image/activity/records/ring-records-popup-frame.png"]')).not.toBeNull()
    expect(await within(dialog).findByText('月亮游记戒指')).toBeInTheDocument()
    expect(within(dialog).getByText('真爱无敌戒指选择权')).toBeInTheDocument()
    expect(within(dialog).getByText('12月24日20:00')).toBeInTheDocument()
    expect(within(dialog).queryByText('加载更多')).not.toBeInTheDocument()
  })

  test('navigates to leaderboard and opens the activity rules dialog', async () => {
    const user = userEvent.setup()
    const { router } = await renderHome()

    await screen.findByText('白昼花园')
    await user.click(screen.getByRole('button', { name: '排行榜' }))
    await waitFor(() => expect(router.currentRoute.value.name).toBe('Leaderboard'))

    await router.push('/')
    await user.click(screen.getByRole('button', { name: '活动说明' }))
    const dialog = await screen.findByRole('dialog', { name: '活动说明' })
    expect(within(dialog).getByTestId('activity-rules-scroll')).toHaveTextContent('活动上线至2026年4月14日23:59:59')
    expect(document.querySelector('main.activity-page')).toHaveAttribute('inert')

    await fireEvent.keyDown(dialog, { key: 'Escape' })
    expect(screen.queryByRole('dialog', { name: '活动说明' })).not.toBeInTheDocument()
  })

  test('automatically opens the new-ring benefit popup once per user in the current browser', async () => {
    await renderHome({ firstVisit: true })

    const dialog = await screen.findByRole('dialog', { name: '新戒福利' })
    expect(dialog).toBeInTheDocument()
    expect(localStorage.getItem('lottery_new_ring_benefit_seen:1')).toBe('true')
  })

  test('opens and closes the new-ring benefit cut popup from the entry after the first visit', async () => {
    const user = userEvent.setup()
    await renderHome()

    await screen.findByText('白昼花园')
    expect(screen.queryByRole('dialog', { name: '新戒福利' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '新戒福利' }))

    const dialog = await screen.findByRole('dialog', { name: '新戒福利' })
    expect(dialog.querySelector(
      'img[src="/image/activity/benefit/new-ring-benefit-popup.png"]',
    )).not.toBeNull()
    expect(within(dialog).getByTestId('new-ring-benefit-scroll')).toBeInTheDocument()
    expect(document.querySelector('main.activity-page')).toHaveAttribute('inert')

    expect(within(dialog).queryByRole('button', { name: '关闭新戒福利' })).not.toBeInTheDocument()
    await fireEvent.keyDown(dialog, { key: 'Escape' })
    expect(screen.queryByRole('dialog', { name: '新戒福利' })).not.toBeInTheDocument()
  })
})
