<template>
  <div class="reward-preview-tile">
    <div
      class="reward-preview-tile__visual"
      :class="{ 'reward-preview-tile__visual--custom-base': baseImageSrc }"
    >
      <img
        v-if="baseImageSrc"
        class="reward-preview-tile__base"
        :src="baseImageSrc"
        alt=""
        aria-hidden="true"
        draggable="false"
      >
      <img
        v-if="resolvedRewardImage"
        class="reward-preview-tile__image"
        :class="rewardImageClass"
        :src="resolvedRewardImage"
        :alt="itemName"
      >
      <RewardIcon v-else class="reward-preview-tile__icon" :kind="rewardKind" />
      <strong class="reward-preview-tile__quantity">x{{ quantity.toLocaleString() }}</strong>
    </div>
    <span class="reward-preview-tile__name">{{ itemName }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import RewardIcon from '@/components/RewardIcon.vue'
import type { RewardIconKind } from '@/types/activity'
import { getRingCatalogItem } from '@/constants/ringCatalog'

const props = defineProps<{
  itemId: number
  itemName: string
  quantity: number
  imageSrc?: string
  baseImageSrc?: string
}>()

const rewardImages: Record<number, string> = {
  1002: '/image/activity/petal.png',
  1323: '/image/leaderboard/rewards/custom-title.png',
  3633: '/image/leaderboard/rewards/spring-princess-title.png',
  1207809: '/image/leaderboard/rewards/spring-princess-frame.png',
  1207811: '/image/leaderboard/rewards/spring-princess-naming.png',
}

const resolvedRewardImage = computed(() => (
  props.imageSrc ?? getRingCatalogItem(props.itemId)?.image ?? rewardImages[props.itemId]
))

const rewardImageClass = computed(() => {
  if (props.itemId === 1323) return 'reward-preview-tile__image--custom-title'
  if (props.itemId === 3633) return 'reward-preview-tile__image--title'
  if (props.itemId === 1207809 || props.itemId === 1207811) {
    return 'reward-preview-tile__image--avatar-frame'
  }
  return ''
})

const rewardKind = computed<RewardIconKind>(() => {
  if (props.itemId === 1001 || props.itemName.includes('金币')) return 'coin'
  if (props.itemId === 1002 || props.itemName.includes('花瓣')) return 'petal'
  if (props.itemName.includes('戒指')) return 'ring'
  return 'gift'
})
</script>

<style scoped>
.reward-preview-tile {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  color: #fff;
  text-align: center;
}

.reward-preview-tile__visual {
  position: relative;
  isolation: isolate;
  width: 82%;
  max-width: 108px;
  aspect-ratio: 1;
  margin: 7px auto 8px;
  background: linear-gradient(145deg, rgba(145, 105, 226, 0.84), rgba(48, 18, 75, 0.92));
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 4px;
  box-shadow:
    7px 7px 0 rgba(42, 14, 68, 0.72),
    -7px -7px 0 rgba(255, 255, 255, 0.16),
    0 0 14px rgba(218, 195, 255, 0.48);
}

.reward-preview-tile__visual::before {
  position: absolute;
  inset: -5px 5px 5px -5px;
  z-index: -1;
  content: '';
  border: 1px solid rgba(255, 255, 255, 0.38);
  border-radius: 4px;
}

.reward-preview-tile__visual--custom-base {
  background: none;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.reward-preview-tile__visual--custom-base::before {
  display: none;
}

.reward-preview-tile__base {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.reward-preview-tile__image,
.reward-preview-tile__icon {
  position: absolute;
  inset: 13%;
  z-index: 1;
  width: 74%;
  height: 74%;
  object-fit: contain;
  filter: drop-shadow(0 5px 7px rgba(34, 10, 54, 0.48));
}

.reward-preview-tile__image--title {
  inset: 23% 2%;
  width: 96%;
  height: 54%;
}

.reward-preview-tile__image--custom-title {
  inset: 18% 2%;
  width: 96%;
  height: 64%;
}

.reward-preview-tile__image--avatar-frame {
  inset: 5%;
  width: 90%;
  height: 90%;
}

.reward-preview-tile__quantity {
  position: absolute;
  right: -13px;
  bottom: -12px;
  z-index: 2;
  color: #fff;
  font-family: Arial, sans-serif;
  font-size: 18px;
  font-weight: 900;
  line-height: 1;
  text-shadow:
    -2px -2px 0 #6f2ba5,
    2px -2px 0 #6f2ba5,
    -2px 2px 0 #6f2ba5,
    2px 2px 0 #6f2ba5;
}

.reward-preview-tile__name {
  display: -webkit-box;
  width: 100%;
  min-height: 2.8em;
  padding-inline: 1px;
  overflow: hidden;
  color: #fffdfb;
  font-size: 12px;
  line-height: 1.4;
  text-shadow: 0 2px 4px rgba(50, 14, 76, 0.72);
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
</style>
