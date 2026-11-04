<template>
  <Teleport to="body">
    <div class="all-rewards-overlay" @click.self="emit('close')">
      <section ref="dialogRef" class="all-rewards-dialog" role="dialog" aria-modal="true" aria-labelledby="all-rewards-title" tabindex="-1" @keydown.esc.stop.prevent="emit('close')">
        <h2 id="all-rewards-title" class="sr-only">查看全部奖励</h2>
        <img class="all-rewards-dialog__frame" src="/image/activity/all-rewards-popup/frame.png" alt="" aria-hidden="true" draggable="false">

        <div class="all-rewards-tabs" role="tablist" aria-label="选择奖池">
          <img :src="tabAsset" alt="" aria-hidden="true" draggable="false">
          <button type="button" role="tab" :aria-selected="selectedPoolType === 1" aria-label="白昼之旅" @click="selectedPoolType = 1" />
          <button type="button" role="tab" :aria-selected="selectedPoolType === 2" aria-label="星夜之旅" @click="selectedPoolType = 2" />
        </div>

        <div class="all-rewards-scroll" aria-live="polite">
          <p v-if="!selectedPool" class="all-rewards-state">奖池奖励加载中</p>
          <template v-else>
            <section v-for="category in categories" :key="category.key" class="all-rewards-category" :aria-labelledby="`all-rewards-${category.key}`">
              <h3 :id="`all-rewards-${category.key}`">
                <img :src="category.asset" :alt="category.accessibleName">
              </h3>
              <ul v-if="category.items.length > 0" class="all-rewards-grid">
                <li v-for="prize in category.items" :key="prize.displayKey">
                  <img
                    v-if="category.key === 'featured'"
                    class="all-rewards-new"
                    src="/image/activity/all-rewards-popup/new-badge.png"
                    alt="New"
                    draggable="false"
                  >
                  <RewardPreviewTile
                    :item-id="prize.item_id"
                    :item-name="prize.item_name"
                    :quantity="prize.quantity"
                    :image-src="getRingAsset(prize.item_id)"
                    base-image-src="/image/activity/results/reward-item-base.png"
                  />
                </li>
              </ul>
              <p v-else class="all-rewards-empty">本奖池暂无此类奖励</p>
            </section>
          </template>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import RewardPreviewTile from '@/components/RewardPreviewTile.vue'
import type { PoolConfig, PoolPrize } from '@/api/lottery'
import { getRingCatalogItem } from '@/constants/ringCatalog'

const props = defineProps<{ pools: PoolConfig[]; initialPoolType: number }>()
const emit = defineEmits<{ close: [] }>()
const dialogRef = ref<HTMLElement | null>(null)
const selectedPoolType = ref(props.initialPoolType)
const selectedPool = computed(() => props.pools.find((pool) => pool.pool_type === selectedPoolType.value))
const tabAsset = computed(() => selectedPoolType.value === 1
  ? '/image/activity/all-rewards-popup/tabs-day.png'
  : '/image/activity/all-rewards-popup/tabs-night.png')

type DisplayPrize = PoolPrize & { displayKey: string }

function getRingAsset(itemId: number) {
  return getRingCatalogItem(itemId)?.image
}

function toDisplayPrize(prize: PoolPrize): DisplayPrize {
  return { ...prize, displayKey: `${prize.id}-${prize.item_id}` }
}

function expandFeaturedPrize(prize: PoolPrize): DisplayPrize[] {
  if (prize.is_choice !== 1 || !prize.choice_ids) return [toDisplayPrize(prize)]

  return prize.choice_ids
    .split(',')
    .map(Number)
    .filter((itemId) => Number.isInteger(itemId) && getRingCatalogItem(itemId))
    .map((itemId) => ({
      ...prize,
      item_id: itemId,
      item_name: getRingCatalogItem(itemId)?.name ?? prize.item_name,
      displayKey: `${prize.id}-${itemId}`,
    }))
}

function isFeatured(prize: PoolPrize) {
  return prize.is_choice === 1 || prize.item_id === 1207751 || prize.item_name.includes('真爱无敌')
}

function isRing(prize: PoolPrize) {
  return !isFeatured(prize) && prize.item_name.includes('戒指')
}

const categories = computed(() => {
  const prizes = selectedPool.value?.prizes ?? []
  return [
    { key: 'featured', accessibleName: '真爱无敌', asset: '/image/activity/all-rewards-popup/category-featured.png', items: prizes.filter(isFeatured).flatMap(expandFeaturedPrize) },
    { key: 'rings', accessibleName: '其他戒指', asset: '/image/activity/all-rewards-popup/category-rings.png', items: prizes.filter(isRing).map(toDisplayPrize) },
    { key: 'items', accessibleName: '其他物品', asset: '/image/activity/all-rewards-popup/category-items.png', items: prizes.filter((prize) => !isFeatured(prize) && !isRing(prize)).map(toDisplayPrize) },
  ]
})

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.all-rewards-overlay {
  position: fixed;
  inset: 0;
  z-index: 1500;
  display: grid;
  padding: max(8px, env(safe-area-inset-top)) 8px max(8px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.all-rewards-dialog {
  position: relative;
  width: min(94vw, 430px);
  max-height: 96dvh;
  aspect-ratio: 654 / 896;
  outline: 0;
  transform-origin: center;
  animation: all-rewards-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.all-rewards-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
}

.all-rewards-tabs {
  position: absolute;
  z-index: 3;
  top: 0.5%;
  left: 9.3%;
  display: grid;
  width: 81.35%;
  aspect-ratio: 532 / 80;
  grid-template-columns: 1fr 1fr;
}

.all-rewards-tabs img { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: fill; pointer-events: none; }
.all-rewards-tabs button { position: relative; z-index: 1; padding: 0; background: transparent; border: 0; cursor: pointer; }

.all-rewards-scroll {
  position: absolute;
  z-index: 2;
  top: 12%;
  right: 7.5%;
  bottom: 6.5%;
  left: 7.5%;
  padding: 6px 8px 28px;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.all-rewards-scroll::-webkit-scrollbar { display: none; }
.all-rewards-category { margin-bottom: 22px; }
.all-rewards-category:last-child { margin-bottom: 0; }
.all-rewards-category h3 { width: 22%; margin: 0 auto 10px; }
.all-rewards-category h3 img { display: block; width: 100%; height: auto; }
.all-rewards-grid { display: flex; margin: 0; padding: 0; flex-wrap: wrap; justify-content: center; gap: 14px 6px; list-style: none; }
.all-rewards-grid li { position: relative; width: 27.5%; min-width: 0; }

.all-rewards-new {
  position: absolute;
  z-index: 4;
  top: 7px;
  left: 9%;
  display: block;
  width: 32px;
  max-width: 35%;
  height: auto;
  pointer-events: none;
  user-select: none;
}

.all-rewards-state,
.all-rewards-empty { margin: 24px 0; color: rgba(255, 255, 255, 0.82); font-size: 13px; text-align: center; }

.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@keyframes all-rewards-dialog-expand {
  0% {
    opacity: 0;
    transform: scale(0.3);
  }

  72% {
    opacity: 1;
    transform: scale(1.03);
  }

  100% {
    opacity: 1;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .all-rewards-dialog {
    animation: none;
  }
}

@media (max-height: 600px) and (orientation: landscape) {
  .all-rewards-dialog { width: auto; height: 96dvh; }
}
</style>
