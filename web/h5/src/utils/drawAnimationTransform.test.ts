import { describe, expect, test } from 'vitest'
import {
  createDrawAnimationTransform,
  DRAW_ANIMATION_DESIGN_PATHS,
  toCssMatrix,
  transformPoint,
  type DrawAnimationCount,
  type Point,
} from './drawAnimationTransform'

function expectPointClose(actual: Point, expected: Point) {
  expect(actual.x).toBeCloseTo(expected.x, 6)
  expect(actual.y).toBeCloseTo(expected.y, 6)
}

describe('drawAnimationTransform', () => {
  test.each([
    {
      name: 'iPhone 12 Pro',
      starts: {
        1: { x: 86, y: 760 },
        10: { x: 195, y: 760 },
        30: { x: 304, y: 760 },
      },
      end: { x: 251, y: 449 },
    },
    {
      name: 'iPhone SE',
      starts: {
        1: { x: 82, y: 590 },
        10: { x: 188, y: 590 },
        30: { x: 294, y: 590 },
      },
      end: { x: 241, y: 293 },
    },
  ])('$name 会把三条 PAG 轨迹映射到实际按钮和花朵', ({ starts, end }) => {
    for (const count of [1, 10, 30] as DrawAnimationCount[]) {
      const transform = createDrawAnimationTransform(count, starts[count], end)
      expect(transform).not.toBeNull()
      if (!transform) continue

      const designPath = DRAW_ANIMATION_DESIGN_PATHS[count]
      expectPointClose(transformPoint(transform, designPath.start), starts[count])
      expectPointClose(transformPoint(transform, designPath.end), end)
      expect(transform.a).toBeCloseTo(transform.d, 10)
      expect(transform.b).toBeCloseTo(-transform.c, 10)
      expect(toCssMatrix(transform)).toMatch(/^matrix\(/)
    }
  })
})
