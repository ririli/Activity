export interface RingCatalogItem {
  name: string
  shortName: string
  image: string
  animatedImage?: string
}

export const RING_CATALOG: Readonly<Record<number, RingCatalogItem>> = {
  1203743: {
    name: '闪闪心蝶戒指',
    shortName: '闪闪心蝶',
    image: '/image/activity/rings/ring-1203743-butterfly.png',
  },
  1205251: {
    name: '月亮游记戒指',
    shortName: '月亮游记',
    image: '/image/activity/rings/ring-1205251-moon.png',
    animatedImage: '/image/activity/rings/ring-1205251-moon.svg',
  },
  1205470: {
    name: '爱意翩跹戒指',
    shortName: '爱意翩跹',
    image: '/image/activity/rings/ring-1205470-love.png',
    animatedImage: '/image/activity/rings/ring-1205470-love.svg',
  },
  1207244: {
    name: '烟花之恋戒指',
    shortName: '烟花之恋',
    image: '/image/activity/rings/ring-1207244-firework.png',
    animatedImage: '/image/activity/rings/ring-1207244-firework.svg',
  },
  1207751: {
    name: '真爱无敌戒指经典款',
    shortName: '经典款',
    image: '/image/activity/rings/ring-1207751-true-love-classic.png',
    animatedImage: '/image/activity/rings/ring-1207751-true-love-classic.svg',
  },
  1207752: {
    name: '真爱无敌戒指铭文款',
    shortName: '真爱无敌铭文款',
    image: '/image/activity/rings/ring-1207752-true-love-inscription.png',
  },
  1207753: {
    name: '爱你万年戒指铭文款',
    shortName: '爱你万年铭文款',
    image: '/image/activity/rings/ring-1207753-love-forever-inscription.png',
  },
}

export function getRingCatalogItem(itemId: number) {
  return RING_CATALOG[itemId]
}
