<template>
  <Teleport to="body">
    <div class="tbp-abandon-overlay" @keydown.esc.stop.prevent="$emit('keep')">
      <section
        ref="dialogRef"
        class="tbp-abandon-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="tbp-abandon-title"
        tabindex="-1"
      >
        <h2 id="tbp-abandon-title" class="sr-only">确认放弃本次奖励</h2>
        <img class="tbp-abandon-dialog__frame" src="/image/activity/tbp-abandon/frame.png" alt="" aria-hidden="true" draggable="false">
        <div class="tbp-abandon-dialog__actions">
          <button type="button" aria-label="确认放弃" :disabled="loading" @click="$emit('confirm')">
            <img src="/image/activity/tbp-abandon/confirm.png" alt="" aria-hidden="true" draggable="false">
          </button>
          <button type="button" aria-label="不放弃" :disabled="loading" @click="$emit('keep')">
            <img src="/image/activity/tbp-abandon/keep.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'

defineProps<{ loading: boolean }>()
defineEmits<{ confirm: []; keep: [] }>()

const dialogRef = ref<HTMLElement | null>(null)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.tbp-abandon-overlay {
  position: fixed;
  inset: 0;
  z-index: 1650;
  display: grid;
  padding: 18px;
  place-items: center;
  background: rgba(10, 6, 30, 0.28);
  backdrop-filter: blur(1px);
  -webkit-backdrop-filter: blur(1px);
}

.tbp-abandon-dialog {
  position: relative;
  width: min(94vw, 470px);
  aspect-ratio: 654 / 480;
  outline: 0;
  transform-origin: center;
  animation: tbp-abandon-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.tbp-abandon-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.tbp-abandon-dialog__actions {
  position: absolute;
  z-index: 1;
  right: 10%;
  bottom: 10%;
  left: 10%;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7%;
}

.tbp-abandon-dialog__actions button {
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 160ms ease, opacity 160ms ease, transform 160ms ease;
}

.tbp-abandon-dialog__actions img {
  display: block;
  width: 100%;
  height: auto;
}

.tbp-abandon-dialog__actions button:active:not(:disabled) { transform: scale(0.97); filter: brightness(1.06); }
.tbp-abandon-dialog__actions button:disabled { cursor: wait; opacity: 0.58; }

@keyframes tbp-abandon-dialog-expand {
  0% { opacity: 0; transform: scale(0.3); }
  72% { opacity: 1; transform: scale(1.03); }
  100% { opacity: 1; transform: scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .tbp-abandon-dialog { animation: none; }
  .tbp-abandon-dialog__actions button { transition-duration: 0.01ms; }
}
</style>
