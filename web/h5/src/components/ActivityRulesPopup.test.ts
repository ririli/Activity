import { fireEvent, render, screen, within } from '@testing-library/vue'
import ActivityRulesPopup from './ActivityRulesPopup.vue'

describe('ActivityRulesPopup', () => {
  test('使用活动说明切图并在独立滚动区展示完整文案', async () => {
    const result = render(ActivityRulesPopup)

    const dialog = screen.getByRole('dialog', { name: '活动说明' })
    const copy = within(dialog).getByTestId('activity-rules-scroll')
    expect(dialog.querySelector('img[src="/image/activity/rules/rules-popup-frame.png"]')).not.toBeNull()
    expect(copy).toHaveTextContent('活动上线至2026年4月14日23:59:59')
    expect(copy).toHaveTextContent('星夜许愿必得稀有及以上等级奖励')
    expect(copy).toHaveTextContent('三、新戒福利')
    expect(copy).toHaveTextContent('点击查看概率公示')
    expect(within(dialog).queryByRole('button', { name: '关闭活动说明' })).not.toBeInTheDocument()

    await fireEvent.keyDown(dialog, { key: 'Escape' })
    expect(result.emitted().close).toHaveLength(1)
  })
})
