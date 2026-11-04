<template>
  <div
    class="flower-box"
    :class="`flower-box--${mode}`"
    :aria-label="`花盒当前点亮${safeLitCount}朵花`"
  >
    <span
      ref="drawAnimationTargetRef"
      class="flower-box__draw-animation-target"
      aria-hidden="true"
    ></span>
    <img
      class="flower-box__image"
      :src="boxImage"
      alt="盛放六朵玫瑰的戒指花盒"
      draggable="false"
    >
    <img
      v-for="(flower, index) in visibleFlowers"
      :key="flower.slot"
      class="flower-box__rose"
      :class="`flower-box__rose--${flower.slot}`"
      :src="flower.src"
      :alt="`已点亮的第${index + 1}朵花`"
      :style="{ animationDelay: `${index * 70}ms` }"
      draggable="false"
    >
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(defineProps<{
  litCount: number
  mode?: 'day' | 'night'
}>(), {
  mode: 'day',
})

const safeLitCount = computed(() => Math.max(0, Math.min(6, props.litCount)))
const mode = computed(() => props.mode)
const drawAnimationTargetRef = ref<HTMLElement | null>(null)
const flowers = [
  { slot: 'bottom-left', src: '/image/activity/flowers/rose-bottom-left.png' },
  { slot: 'bottom-center', src: '/image/activity/flowers/rose-bottom-center.png' },
  { slot: 'bottom-right', src: '/image/activity/flowers/rose-bottom-right.png' },
  { slot: 'top-left', src: '/image/activity/flowers/rose-top-left.png' },
  { slot: 'top-center', src: '/image/activity/flowers/rose-top-center.png' },
  { slot: 'top-right', src: '/image/activity/flowers/rose-top-right.png' },
] as const
const visibleFlowers = computed(() => flowers.slice(0, safeLitCount.value))
const boxImage = computed(() => props.mode === 'night'
  ? '/image/activity/flowers/flower-box-night.png'
  : '/image/activity/flowers/flower-box-day.png')

function getDrawAnimationTargetRect() {
  return drawAnimationTargetRef.value?.getBoundingClientRect() ?? null
}

defineExpose({ getDrawAnimationTargetRect })
</script>

<style scoped>
.flower-box {
  position: relative;
  aspect-ratio: 538 / 547;
}

.flower-box__image {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.flower-box__draw-animation-target {
  position: absolute;
  top: 43%;
  left: 52%;
  width: 1px;
  height: 1px;
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.flower-box__rose {
  position: absolute;
  z-index: 2;
  height: auto;
  object-fit: contain;
  pointer-events: none;
  filter: drop-shadow(0 2px 5px rgba(117, 0, 31, 0.55)) saturate(1.08);
  animation: flower-light-in 460ms cubic-bezier(0.2, 0.88, 0.28, 1) both;
}

.flower-box__rose--bottom-left { top: 38.4%; left: 36.6%; width: 18.96%; }
.flower-box__rose--bottom-center { top: 45.3%; left: 53.2%; width: 17.84%; }
.flower-box__rose--bottom-right { top: 39.7%; left: 67.3%; width: 16.73%; }
.flower-box__rose--top-left { top: 32%; left: 21%; width: 17.66%; }
.flower-box__rose--top-center { top: 25.8%; left: 36.6%; width: 17.66%; }
.flower-box__rose--top-right { top: 27.4%; left: 53%; width: 19.89%; }

@keyframes flower-light-in {
  0% {
    opacity: 0;
    transform: scale(0.55);
    filter: brightness(2.2) drop-shadow(0 0 20px rgba(255, 225, 235, 1));
  }
  58% {
    opacity: 1;
    transform: scale(1.12);
    filter: brightness(1.35) drop-shadow(0 0 13px rgba(255, 106, 151, 0.9));
  }
  100% {
    opacity: 1;
    transform: scale(1);
    filter: drop-shadow(0 2px 5px rgba(117, 0, 31, 0.55)) saturate(1.08);
  }
}

@media (prefers-reduced-motion: reduce) {
  .flower-box__rose {
    animation-duration: 0.01ms;
    animation-delay: 0ms !important;
  }
}
</style>
