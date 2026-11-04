<template>
  <Teleport to="body">
    <div class="tbp-offer-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="tbp-offer-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="tbp-offer-title"
        tabindex="-1"
      >
        <h2 id="tbp-offer-title" class="sr-only">先抽后付机会</h2>
        <img class="tbp-offer-dialog__frame" src="/image/activity/tbp-offer/frame.png" alt="" aria-hidden="true" draggable="false">
        <p class="tbp-offer-dialog__copy">
          您当前拥有1次白昼许愿先抽后付180次的机会，可免费许愿，预览奖励再决定是否付费，是否使用？
        </p>
        <p class="tbp-offer-dialog__note">*仅在未进行过许愿的时候提供先抽后付机会哦</p>
        <div class="tbp-offer-dialog__actions">
          <button type="button" aria-label="再想想" :disabled="loading" @click="$emit('close')">
            <img src="/image/activity/tbp-offer/rethink.png" alt="" aria-hidden="true" draggable="false">
          </button>
          <button type="button" :aria-label="loading ? '正在生成先抽后付奖励' : '使用先抽后付'" :disabled="loading" @click="$emit('use')">
            <img src="/image/activity/tbp-offer/confirm.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'

defineProps<{ loading: boolean }>()
defineEmits<{ close: []; use: [] }>()

const dialogRef = ref<HTMLElement | null>(null)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.tbp-offer-overlay {
  position: fixed;
  inset: 0;
  z-index: 1550;
  display: grid;
  padding: 18px;
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.tbp-offer-dialog {
  position: relative;
  width: min(94vw, 470px);
  aspect-ratio: 654 / 480;
  color: #fff;
  outline: 0;
  transform-origin: center;
  animation: tbp-offer-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.tbp-offer-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.tbp-offer-dialog__copy,
.tbp-offer-dialog__note {
  position: absolute;
  z-index: 1;
  right: 11.5%;
  left: 11.5%;
  margin: 0;
  font-family: 'Songti SC', 'STSong', serif;
  font-weight: 700;
  text-shadow: 0 2px 5px rgba(45, 22, 91, 0.72);
}

.tbp-offer-dialog__copy {
  top: 32%;
  font-size: clamp(14px, 4vw, 20px);
  line-height: 1.32;
}

.tbp-offer-dialog__note {
  top: 57%;
  font-size: clamp(12px, 3.3vw, 16px);
  line-height: 1.45;
}

.tbp-offer-dialog__actions {
  position: absolute;
  z-index: 2;
  right: 9%;
  bottom: 10%;
  left: 9%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7%;
}

.tbp-offer-dialog__actions button {
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 160ms ease, opacity 160ms ease, transform 160ms ease;
}

.tbp-offer-dialog__actions img {
  display: block;
  width: 100%;
  height: auto;
}

.tbp-offer-dialog__actions button:active:not(:disabled) { transform: scale(0.97); filter: brightness(1.06); }
.tbp-offer-dialog__actions button:disabled { cursor: wait; opacity: 0.58; }

@keyframes tbp-offer-dialog-expand {
  0% { opacity: 0; transform: scale(0.3); }
  72% { opacity: 1; transform: scale(1.03); }
  100% { opacity: 1; transform: scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .tbp-offer-dialog { animation: none; }
  .tbp-offer-dialog__actions button { transition-duration: 0.01ms; }
}
</style>
