<template>
  <Teleport to="body">
    <div class="true-love-result-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="true-love-result-dialog popup-center-expand"
        role="dialog"
        aria-modal="true"
        aria-labelledby="true-love-result-title"
        tabindex="-1"
      >
        <h2 id="true-love-result-title" class="sr-only">恭喜获得真爱无敌戒指</h2>
        <p class="sr-only">{{ itemName }}，道具ID {{ itemId }}</p>
        <img
          class="true-love-result-dialog__frame"
          :src="frameAsset"
          :alt="itemName"
          draggable="false"
        >
        <img
          class="true-love-result-dialog__avatar"
          src="/image/leaderboard/profile/default-avatar.png"
          :alt="`${nickname}头像`"
          draggable="false"
        >
        <p class="true-love-result-dialog__nickname" :title="nickname">{{ nickname }}</p>

        <div class="true-love-result-actions">
          <button type="button" aria-label="确定" @click="$emit('confirm')">
            <img src="/image/activity/true-love-acquired/confirm.png" alt="" aria-hidden="true" draggable="false">
          </button>
          <button type="button" aria-label="分享到玩友圈" @click="noop">
            <img src="/image/activity/true-love-acquired/share.png" alt="" aria-hidden="true" draggable="false">
          </button>
          <button type="button" aria-label="相机" @click="noop">
            <img src="/image/activity/true-love-acquired/camera.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'

const props = defineProps<{
  itemId: number
  itemName: string
  nickname: string
}>()

defineEmits<{
  confirm: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
const noop = () => undefined
const frameAssets: Record<number, string> = {
  1207752: '/image/activity/true-love-acquired/frame-true-love-inscription.png',
  1207753: '/image/activity/true-love-acquired/frame-love-forever-inscription.png',
}
const frameAsset = computed(() => frameAssets[props.itemId] ?? '/image/activity/true-love-acquired/frame.png')

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.true-love-result-overlay {
  position: fixed;
  inset: 0;
  z-index: 1800;
  display: grid;
  padding: max(8px, env(safe-area-inset-top)) 8px max(8px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(7, 6, 18, 0.72);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.true-love-result-dialog {
  position: relative;
  width: min(94vw, 430px);
  max-height: 96dvh;
  aspect-ratio: 750 / 990;
  outline: 0;
}

.true-love-result-dialog__frame {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.true-love-result-dialog__avatar {
  position: absolute;
  z-index: 1;
  top: 14.15%;
  left: 24.1%;
  width: 17.1%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: 50%;
  pointer-events: none;
  user-select: none;
}

.true-love-result-dialog__nickname {
  position: absolute;
  z-index: 2;
  top: 17.8%;
  right: 29.5%;
  left: 42%;
  margin: 0;
  overflow: hidden;
  color: #fff;
  font-size: clamp(12px, 3.8vw, 17px);
  font-weight: 700;
  line-height: 1.8;
  text-align: center;
  text-overflow: ellipsis;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.85);
  white-space: nowrap;
}

.true-love-result-actions {
  position: absolute;
  right: 8%;
  bottom: 1.5%;
  left: 8%;
  display: grid;
  grid-template-columns: 1.1fr 1.84fr 0.52fr;
  align-items: center;
  gap: 3%;
  transform: translateY(40px);
}

.true-love-result-actions button {
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, transform 150ms ease;
}

.true-love-result-actions img {
  display: block;
  width: 100%;
  height: auto;
}

.true-love-result-actions button:active { transform: scale(0.96); filter: brightness(1.08); }

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

@media (max-height: 620px) and (orientation: landscape) {
  .true-love-result-dialog {
    width: auto;
    height: 96dvh;
  }
}

@media (prefers-reduced-motion: reduce) {
  .true-love-result-actions button { transition-duration: 0.01ms; }
}
</style>
