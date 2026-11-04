import { render } from '@testing-library/vue'
import FallingPetals from './FallingPetals.vue'

describe('FallingPetals', () => {
  test('使用两种花瓣切图生成循环飘落层', () => {
    const { container } = render(FallingPetals)
    const layer = container.querySelector('.falling-petals')
    const petals = container.querySelectorAll('.falling-petals__item')

    expect(layer).toHaveAttribute('aria-hidden', 'true')
    expect(petals).toHaveLength(12)
    expect(container.querySelectorAll('img[src="/image/activity/effects/falling-petal-1.png"]')).toHaveLength(6)
    expect(container.querySelectorAll('img[src="/image/activity/effects/falling-petal-2.png"]')).toHaveLength(6)
  })
})
