export type DrawAnimationCount = 1 | 10 | 30

export type Point = {
  x: number
  y: number
}

export type AffineTransform = {
  a: number
  b: number
  c: number
  d: number
  e: number
  f: number
}

export const DRAW_ANIMATION_CANVAS_SIZE = {
  width: 750,
  height: 1820,
} as const

// 坐标来自三个 PAG 文件中花瓣轨迹的首帧中心与汇聚点。
export const DRAW_ANIMATION_DESIGN_PATHS: Record<DrawAnimationCount, { start: Point; end: Point }> = {
  1: {
    start: { x: 149, y: 1667 },
    end: { x: 483, y: 1075 },
  },
  10: {
    start: { x: 377, y: 1663 },
    end: { x: 483, y: 1075 },
  },
  30: {
    start: { x: 609, y: 1662 },
    end: { x: 483, y: 1075 },
  },
}

export function createSimilarityTransform(
  designStart: Point,
  designEnd: Point,
  viewportStart: Point,
  viewportEnd: Point,
): AffineTransform | null {
  const designX = designEnd.x - designStart.x
  const designY = designEnd.y - designStart.y
  const viewportX = viewportEnd.x - viewportStart.x
  const viewportY = viewportEnd.y - viewportStart.y
  const designLengthSquared = designX * designX + designY * designY
  const viewportLengthSquared = viewportX * viewportX + viewportY * viewportY

  if (
    designLengthSquared <= Number.EPSILON
    || viewportLengthSquared <= Number.EPSILON
    || ![designX, designY, viewportX, viewportY].every(Number.isFinite)
  ) return null

  const a = (viewportX * designX + viewportY * designY) / designLengthSquared
  const b = (viewportY * designX - viewportX * designY) / designLengthSquared
  const c = -b
  const d = a
  const e = viewportStart.x - a * designStart.x - c * designStart.y
  const f = viewportStart.y - b * designStart.x - d * designStart.y

  if (![a, b, c, d, e, f].every(Number.isFinite)) return null
  return { a, b, c, d, e, f }
}

export function createDrawAnimationTransform(
  count: DrawAnimationCount,
  viewportStart: Point,
  viewportEnd: Point,
) {
  const designPath = DRAW_ANIMATION_DESIGN_PATHS[count]
  return createSimilarityTransform(designPath.start, designPath.end, viewportStart, viewportEnd)
}

export function transformPoint(transform: AffineTransform, point: Point): Point {
  return {
    x: transform.a * point.x + transform.c * point.y + transform.e,
    y: transform.b * point.x + transform.d * point.y + transform.f,
  }
}

export function toCssMatrix(transform: AffineTransform) {
  const values = [transform.a, transform.b, transform.c, transform.d, transform.e, transform.f]
    .map((value) => Math.abs(value) < 1e-10 ? 0 : Number(value.toFixed(6)))
  return `matrix(${values.join(', ')})`
}
