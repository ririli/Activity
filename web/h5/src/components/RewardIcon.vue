<template>
  <svg v-if="kind === 'coin'" viewBox="0 0 64 64" role="img" aria-label="金币">
    <defs>
      <linearGradient id="coin-fill" x1="12" y1="9" x2="52" y2="55" gradientUnits="userSpaceOnUse">
        <stop stop-color="#FFF7B2" />
        <stop offset="0.48" stop-color="#F6C95D" />
        <stop offset="1" stop-color="#C27A24" />
      </linearGradient>
    </defs>
    <circle cx="32" cy="32" r="24" fill="url(#coin-fill)" stroke="#FFF4C3" stroke-width="3" />
    <circle cx="32" cy="32" r="17" fill="none" stroke="#BA7026" stroke-width="2" opacity="0.72" />
    <path d="M32 18l4.2 8.5 9.4 1.4-6.8 6.6 1.6 9.3-8.4-4.4-8.4 4.4 1.6-9.3-6.8-6.6 9.4-1.4L32 18Z" fill="#FFF7B2" />
  </svg>

  <svg v-else-if="kind === 'petal'" viewBox="0 0 64 64" role="img" aria-label="花瓣">
    <defs>
      <linearGradient id="petal-fill" x1="13" y1="11" x2="50" y2="54" gradientUnits="userSpaceOnUse">
        <stop stop-color="#FFF2F7" />
        <stop offset="0.5" stop-color="#FA9AB9" />
        <stop offset="1" stop-color="#C84C7A" />
      </linearGradient>
    </defs>
    <path d="M32 31C27 17 17 10 10 14c-8 5-2 20 14 24-13 5-17 17-9 21 7 4 16-4 18-16 4 12 15 18 21 12 7-7-1-18-13-20 12-7 14-20 5-23-8-3-14 7-14 19Z" fill="url(#petal-fill)" stroke="#FFF5FA" stroke-width="2" />
    <circle cx="32" cy="35" r="6" fill="#F8D67B" />
  </svg>

  <svg v-else-if="kind === 'ring'" viewBox="0 0 64 64" role="img" aria-label="戒指">
    <defs>
      <linearGradient id="ring-fill" x1="15" y1="9" x2="49" y2="54" gradientUnits="userSpaceOnUse">
        <stop stop-color="#FFFFFF" />
        <stop offset="0.4" stop-color="#F3C8FF" />
        <stop offset="1" stop-color="#9D72D8" />
      </linearGradient>
    </defs>
    <path d="M22 19 32 9l10 10-10 10-10-10Z" fill="url(#ring-fill)" stroke="#FFF" stroke-width="2" />
    <path d="M23 28c-6 4-10 11-8 18 2 9 10 14 19 13 9-1 16-9 15-18-1-7-5-12-11-15" fill="none" stroke="#F7D793" stroke-width="5" stroke-linecap="round" />
    <path d="m22 19 10 2 10-2M32 9v12" fill="none" stroke="#B26EDA" stroke-width="1.5" />
  </svg>

  <svg v-else viewBox="0 0 64 64" role="img" aria-label="礼物">
    <defs>
      <linearGradient id="gift-fill" x1="12" y1="10" x2="52" y2="57" gradientUnits="userSpaceOnUse">
        <stop stop-color="#FFF2F7" />
        <stop offset="0.5" stop-color="#F38EB2" />
        <stop offset="1" stop-color="#A94D82" />
      </linearGradient>
    </defs>
    <path d="M10 27h44v29H10V27Z" fill="url(#gift-fill)" stroke="#FFF4F8" stroke-width="2" />
    <path d="M7 20h50v12H7V20Z" fill="#F7D793" stroke="#FFF4D4" stroke-width="2" />
    <path d="M32 20v36" stroke="#FFF5D9" stroke-width="5" />
    <path d="M31 20C19 20 15 9 22 7c6-2 10 7 10 13Zm2 0c12 0 16-11 9-13-6-2-10 7-10 13Z" fill="#F8C4D9" stroke="#FFF" stroke-width="2" />
  </svg>
</template>

<script setup lang="ts">
import type { RewardIconKind } from '@/types/activity'

defineProps<{
  kind: RewardIconKind
}>()
</script>

<style scoped>
svg {
  display: block;
  overflow: visible;
}
</style>
