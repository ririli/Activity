import { render, screen, waitFor, within } from '@testing-library/vue'
import userEvent from '@testing-library/user-event'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import Leaderboard from './Leaderboard.vue'
import { apiState } from '@/test/server'

async function renderLeaderboard() {
  localStorage.setItem('lottery_access_token', 'test-token')
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'Home', component: { template: '<div>首页</div>' } },
      { path: '/leaderboard', name: 'Leaderboard', component: Leaderboard },
    ],
  })
  await router.push('/leaderboard')
  await router.isReady()

  const result = render(Leaderboard, {
    global: {
      plugins: [createPinia(), router],
    },
  })
  return { ...result, router }
}

describe('Leaderboard page', () => {
  test('使用 Figma 奖励资源并展示默认头像排行榜', async () => {
    await renderLeaderboard()

    const rewardSection = screen.getByRole('region', { name: '榜单奖励' })
    expect(within(rewardSection).getByAltText('冠名定制说明')).toHaveAttribute(
      'src',
      '/image/leaderboard/naming-guide.png',
    )
    expect(within(rewardSection).getByAltText('春日公主冠名头像框90天')).toHaveAttribute(
      'src',
      '/image/leaderboard/rewards/spring-princess-naming.png',
    )
    expect(rewardSection.querySelector('source[srcset="/image/leaderboard/rewards/spring-princess-naming.svg"]')).not.toBeNull()
    expect(within(rewardSection).getByAltText('春日公主称号60天')).toBeInTheDocument()
    expect(within(rewardSection).getByAltText('春日公主头像框60天')).toBeInTheDocument()
    expect(within(rewardSection).getByAltText('春日公主称号30天')).toBeInTheDocument()
    expect(within(rewardSection).getByAltText('春日公主头像框30天')).toBeInTheDocument()
    expect(within(rewardSection).getByAltText('春日公主称号10天')).toBeInTheDocument()
    expect(rewardSection).not.toHaveTextContent('春日王子')
    expect(await screen.findByText('月下花语')).toBeInTheDocument()
    expect(screen.getByText('春风十里')).toBeInTheDocument()
    expect(screen.getByText('星河入梦')).toBeInTheDocument()
    expect(screen.getAllByRole('listitem')).toHaveLength(17)
    expect(screen.queryByText('超额用户21')).not.toBeInTheDocument()
    expect(screen.queryByText('超额用户22')).not.toBeInTheDocument()
    expect(document.querySelectorAll('img[src="/image/leaderboard/rank-emblem.png"]')).toHaveLength(17)
    expect(document.querySelectorAll('img[src="/image/leaderboard/profile/default-avatar.png"]')).toHaveLength(21)
    expect(document.querySelectorAll('img[src="/image/leaderboard/profile/rank-base.png"]')).toHaveLength(17)
    expect(document.querySelector('img[src="/image/leaderboard/profile/crown-gold.png"]')).not.toBeNull()
    expect(document.querySelector('img[src="/image/leaderboard/profile/crown-silver.png"]')).not.toBeNull()
    expect(document.querySelector('img[src="/image/leaderboard/profile/crown-bronze.png"]')).not.toBeNull()
  })

  test('本人排名栏位于滚动区之外并只显示昵称', async () => {
    await renderLeaderboard()

    const footer = await screen.findByRole('contentinfo', { name: '我的排名' })
    const scroll = screen.getByTestId('leaderboard-scroll')

    expect(scroll).not.toContainElement(footer)
    expect(await within(footer).findByAltText('未上榜')).toHaveAttribute('src', '/image/leaderboard/unranked.png')
    expect(footer).toHaveTextContent('测试用户')
    expect(footer).toHaveTextContent('1,280')
    expect(within(footer).getByAltText('测试用户头像')).toHaveAttribute(
      'src',
      '/image/leaderboard/profile/default-avatar.png',
    )
  })

  test('打开冠名说明弹窗并在独立滚动区展示完整文案', async () => {
    const user = userEvent.setup()
    await renderLeaderboard()

    await user.click(screen.getByRole('button', { name: '查看冠名定制说明' }))

    const dialog = screen.getByRole('dialog', { name: '定制称号说明' })
    expect(dialog.querySelector('.naming-dialog-panel')).toHaveClass('popup-center-expand')
    const copy = within(dialog).getByTestId('naming-dialog-scroll')
    expect(copy).toHaveTextContent('【定制&修改V8称号】说明：')
    expect(copy).toHaveTextContent('文字最多支持10个字符')
    expect(copy).toHaveTextContent('【头像框冠名】说明：')
    expect(copy).toHaveTextContent('活动结束后的30天内完成冠名')
    expect(screen.getByTestId('leaderboard-scroll')).toHaveAttribute('inert')

    await user.click(within(dialog).getByRole('button', { name: '关闭定制称号说明' }))
    expect(screen.queryByRole('dialog', { name: '定制称号说明' })).not.toBeInTheDocument()
  })

  test('展示音乐和刷新按钮并重新请求排行榜数据', async () => {
    const user = userEvent.setup()
    await renderLeaderboard()

    const musicButton = screen.getByRole('button', { name: '关闭背景音乐' })
    const refreshButton = screen.getByRole('button', { name: '刷新排行榜数据' })
    await waitFor(() => expect(apiState.rankingRequestCount).toBe(1))

    await user.click(musicButton)
    expect(screen.getByRole('button', { name: '开启背景音乐' })).toBeInTheDocument()
    expect(musicButton.querySelector('img')).toHaveAttribute('src', '/image/activity/nav-music-off.png')

    await user.click(refreshButton)
    await waitFor(() => expect(apiState.rankingRequestCount).toBe(2))
    expect(await screen.findByText('排行榜已刷新')).toBeInTheDocument()
    expect(refreshButton).toBeEnabled()

    await user.click(musicButton)
  })
})
