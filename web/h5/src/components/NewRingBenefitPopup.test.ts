import { render, screen } from '@testing-library/vue'
import NewRingBenefitPopup from './NewRingBenefitPopup.vue'

describe('NewRingBenefitPopup', () => {
  test('在戒指物语左侧展示真爱无敌动效', async () => {
    render(NewRingBenefitPopup)

    const dialog = await screen.findByRole('dialog', { name: '新戒福利' })
    const artwork = dialog.querySelector('.benefit-artwork')
    const animatedRing = dialog.querySelector('.benefit-artwork__ring')

    expect(artwork).not.toBeNull()
    expect(animatedRing?.querySelector('source')).toHaveAttribute(
      'srcset',
      '/image/activity/rings/ring-1207751-true-love-classic.svg',
    )
    expect(animatedRing?.querySelector('img')).toHaveAttribute(
      'src',
      '/image/activity/rings/ring-1207751-true-love-classic.png',
    )
  })
})
