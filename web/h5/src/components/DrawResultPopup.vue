<template>
  <Teleport to="body">
    <div class="draw-result-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="draw-result-dialog"
        :class="{ 'draw-result-dialog--single': isSingle }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="draw-result-title"
        tabindex="-1"
      >
        <h2 id="draw-result-title" class="sr-only">恭喜获得</h2>
        <img
          class="draw-result-dialog__frame"
          :src="frameAsset"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <ul class="draw-result-list" aria-label="本次许愿奖励">
          <li v-for="reward in groupedRewards" :key="reward.item_id">
            <img
              class="draw-result-list__quality-tag"
              :src="getRewardQualityTag(reward).asset"
              :alt="getRewardQualityTag(reward).label"
              draggable="false"
            >
            <RewardPreviewTile
              :item-id="reward.item_id"
              :item-name="reward.item_name"
              :quantity="reward.quantity"
              base-image-src="/image/activity/results/reward-item-base.png"
            />
          </li>
          <li
            v-if="result.lit_flowers > 0"
            class="draw-result-list__flowers"
            :aria-label="`本次点亮花朵${result.lit_flowers}朵`"
          >
            <RewardPreviewTile
              :item-id="-1"
              item-name="点亮花朵"
              :quantity="result.lit_flowers"
              image-src="/image/activity/flowers/rose-bottom-center.png"
              base-image-src="/image/activity/results/reward-item-base.png"
            />
          </li>
        </ul>

        <button
          v-if="useSingleConfirm"
          class="draw-result-confirm"
          type="button"
          :disabled="confirming"
          :aria-label="confirming ? '正在确认奖励' : '确定'"
          @click="$emit('confirm')"
        >
          <img src="/image/activity/results/draw-result-confirm.png" alt="" aria-hidden="true" draggable="false">
        </button>

        <div v-else class="draw-result-actions">
          <button
            class="draw-result-action"
            type="button"
            :disabled="confirming"
            :aria-label="confirming ? '正在确认奖励' : '确定'"
            @click="$emit('confirm')"
          >
            <img
              src="/image/activity/results/draw-result-confirm-pair.png"
              alt=""
              aria-hidden="true"
              draggable="false"
            >
          </button>
          <button
            class="draw-result-action"
            type="button"
            :disabled="confirming"
            :aria-label="confirming ? '正在处理再抽' : `再抽${result.requested_count}次`"
            @click="$emit('redraw', result.requested_count)"
          >
            <img :src="redrawAsset" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import RewardPreviewTile from '@/components/RewardPreviewTile.vue'
import type { DrawResponse } from '@/api/lottery'

const props = defineProps<{
  result: DrawResponse
  confirming: boolean
  poolType: number
}>()

defineEmits<{
  confirm: []
  redraw: [count: number]
}>()

const dialogRef = ref<HTMLElement | null>(null)
const isSingle = computed(() => props.result.requested_count === 1)
const hasLitFlowers = computed(() => props.result.lit_flowers > 0)
const redrawAssets: Record<number, Record<number, string>> = {
  1: {
    1: '/image/activity/results/redraw-day-1.png',
    10: '/image/activity/results/redraw-day-10.png',
    30: '/image/activity/results/redraw-day-30.png',
  },
  2: {
    1: '/image/activity/results/redraw-night-1.png',
    10: '/image/activity/results/redraw-night-10.png',
    30: '/image/activity/results/redraw-night-30.png',
  },
}
const redrawAsset = computed(() => (
  redrawAssets[props.poolType]?.[props.result.requested_count]
  ?? redrawAssets[1][1]
))
const rewardQualityTags = {
  featured: {
    label: '臻品',
    asset: '/image/activity/all-rewards-popup/category-featured.png',
  },
  ring: {
    label: '极品',
    asset: '/image/activity/all-rewards-popup/category-rings.png',
  },
  item: {
    label: '精美',
    asset: '/image/activity/all-rewards-popup/category-items.png',
  },
} as const

function isTrueLoveReward(reward: DrawResponse['rewards'][number]) {
  return reward.is_choice
    || [1207751, 1207752, 1207753].includes(reward.item_id)
    || reward.item_name.includes('真爱无敌')
    || reward.item_name.includes('爱你万年')
}

function getRewardQualityTag(reward: DrawResponse['rewards'][number]) {
  if (isTrueLoveReward(reward)) return rewardQualityTags.featured
  if (reward.item_name.includes('戒指')) return rewardQualityTags.ring
  return rewardQualityTags.item
}

const useSingleConfirm = computed(() => (
  hasLitFlowers.value || props.result.rewards.some(isTrueLoveReward)
))

const groupedRewards = computed(() => {
  const rewardsByItemId = new Map<number, DrawResponse['rewards'][number]>()

  for (const reward of props.result.rewards) {
    const groupedReward = rewardsByItemId.get(reward.item_id)
    if (groupedReward) {
      groupedReward.quantity += reward.quantity
      continue
    }

    rewardsByItemId.set(reward.item_id, { ...reward })
  }

  return [...rewardsByItemId.values()]
})
const frameAsset = computed(() => isSingle.value
  ? '/image/activity/results/draw-result-single.png'
  : '/image/activity/results/draw-result-multi.png')

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.draw-result-overlay {
  position: fixed;
  inset: 0;
  z-index: 1500;
  display: grid;
  padding: max(10px, env(safe-area-inset-top)) 10px max(10px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.draw-result-dialog {
  position: relative;
  width: min(94vw, 370px);
  max-height: 94dvh;
  aspect-ratio: 654 / 896;
  color: #fff;
  outline: 0;
  transform-origin: center;
  animation: draw-result-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.draw-result-dialog--single {
  width: min(94vw, 390px);
  aspect-ratio: 1;
}

.draw-result-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
  user-select: none;
}

.draw-result-list {
  position: absolute;
  top: 15.5%;
  right: 8.5%;
  bottom: 16.5%;
  left: 8.5%;
  z-index: 1;
  display: flex;
  min-height: 0;
  margin: 0;
  padding: 7px 4px 16px;
  overflow-x: hidden;
  overflow-y: auto;
  flex-wrap: wrap;
  align-content: safe center;
  justify-content: center;
  gap: 17px 8px;
  list-style: none;
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.draw-result-list::-webkit-scrollbar {
  display: none;
}

.draw-result-dialog--single .draw-result-list {
  top: 24%;
  right: 15%;
  bottom: 25%;
  left: 15%;
  align-items: start;
  padding: 0 4%;
  overflow: hidden;
}

.draw-result-list li {
  position: relative;
  flex: 0 0 calc((100% - 16px) / 3);
  min-width: 0;
}

.draw-result-list__quality-tag {
  position: absolute;
  z-index: 3;
  top: 7px;
  left: calc(9% - 10px);
  display: block;
  width: 40%;
  height: auto;
  transform: translateY(-50%);
  pointer-events: none;
  user-select: none;
}

.draw-result-dialog--single .draw-result-list li {
  max-width: 132px;
  flex-basis: calc((100% - 8px) / 2);
}

.draw-result-confirm {
  position: absolute;
  z-index: 2;
  right: 24.3%;
  bottom: calc(3.7% + 20px);
  width: 51.4%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, transform 150ms ease, opacity 150ms ease;
}

.draw-result-confirm img {
  display: block;
  width: 100%;
  height: auto;
}

.draw-result-confirm:active {
  transform: scale(0.97);
}

.draw-result-confirm:disabled {
  cursor: wait;
  opacity: 0.58;
}

.draw-result-actions {
  position: absolute;
  z-index: 2;
  right: 8.5%;
  bottom: calc(3.7% + 20px);
  left: 8.5%;
  display: grid;
  align-items: center;
  grid-template-columns: 196fr 328fr;
  gap: 2.5%;
}

.draw-result-action {
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, transform 150ms ease, opacity 150ms ease;
}

.draw-result-action img {
  display: block;
  width: 100%;
  height: auto;
}

.draw-result-action:active {
  transform: scale(0.97);
}

.draw-result-action:disabled {
  cursor: wait;
  opacity: 0.58;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@keyframes draw-result-dialog-expand {
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
  .draw-result-dialog {
    animation: none;
  }
}

@media (max-height: 560px) and (orientation: landscape) {
  .draw-result-dialog,
  .draw-result-dialog--single {
    width: auto;
    height: 94dvh;
  }
}
</style>
