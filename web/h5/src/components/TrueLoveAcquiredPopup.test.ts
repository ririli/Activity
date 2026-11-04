import { render, screen } from '@testing-library/vue'
import TrueLoveAcquiredPopup from './TrueLoveAcquiredPopup.vue'

describe('TrueLoveAcquiredPopup', () => {
  test.each([
    [1207751, '真爱无敌戒指经典款', '/image/activity/true-love-acquired/frame.png'],
    [1207752, '真爱无敌戒指铭文款', '/image/activity/true-love-acquired/frame-true-love-inscription.png'],
    [1207753, '爱你万年戒指铭文款', '/image/activity/true-love-acquired/frame-love-forever-inscription.png'],
  ])('uses the matching share artwork for item %s', (itemId, itemName, frame) => {
    render(TrueLoveAcquiredPopup, {
      props: {
        itemId,
        itemName,
        nickname: '测试用户',
      },
    })

    const dialog = screen.getByRole('dialog', { name: '恭喜获得真爱无敌戒指' })
    expect(dialog).toHaveClass('popup-center-expand')
    expect(screen.getByAltText(itemName)).toHaveAttribute('src', frame)
    expect(screen.getByAltText('测试用户头像')).toHaveAttribute(
      'src',
      '/image/leaderboard/profile/default-avatar.png',
    )
    expect(dialog).toHaveTextContent(`道具ID ${itemId}`)
  })
})
