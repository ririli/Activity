<template>
  <Teleport to="body">
    <div class="round-refund-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="round-refund-dialog popup-center-expand"
        role="dialog"
        aria-modal="true"
        aria-labelledby="round-refund-title"
        tabindex="-1"
      >
        <h2 id="round-refund-title" class="sr-only">花瓣退回提示</h2>
        <img
          class="round-refund-dialog__frame"
          src="/image/activity/refund/round-refund-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >
        <p class="round-refund-dialog__message">
          恭喜您本次抽奖提前点亮全部花朵，<br>
          本次实际完成{{ result.actual_count }}抽，为您退回<br>
          <strong>{{ result.refund_petals.toLocaleString() }}</strong>花瓣
        </p>
        <button type="button" aria-label="确认花瓣退回" @click="$emit('confirm')">
          <img
            src="/image/activity/refund/round-refund-confirm.png"
            alt="确定"
            draggable="false"
          >
        </button>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import type { DrawResponse } from '@/api/lottery'

defineProps<{
  result: DrawResponse
}>()

defineEmits<{
  confirm: []
}>()

const dialogRef = ref<HTMLElement | null>(null)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.round-refund-overlay {
  position: fixed;
  inset: 0;
  z-index: 1550;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 12px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.round-refund-dialog {
  position: relative;
  width: min(94vw, 436px);
  aspect-ratio: 654 / 480;
  outline: 0;
}

.round-refund-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
}

.round-refund-dialog__message {
  position: absolute;
  top: 43%;
  right: 10%;
  left: 10%;
  z-index: 1;
  margin: 0;
  color: #fff;
  font-family: 'Source Han Serif CN', 'Songti SC', serif;
  font-size: clamp(15px, 4.2vw, 21px);
  font-weight: 700;
  line-height: 1.55;
  text-align: center;
  text-shadow: 0 2px 5px rgba(58, 28, 108, 0.55);
  transform: translateY(-50%);
}

.round-refund-dialog__message strong {
  color: #fff3ae;
  font-size: 1.18em;
}

.round-refund-dialog button {
  position: absolute;
  right: 31%;
  bottom: 8%;
  z-index: 2;
  width: 38%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.round-refund-dialog button img {
  display: block;
  width: 100%;
  height: auto;
}

.round-refund-dialog button:active { transform: scale(0.97); }

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
</style>
