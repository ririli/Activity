<template>
  <Teleport to="body">
    <div class="gift-overlay" @click.self="!showResult && emit('close')">
      <section
        ref="dialogRef"
        class="gift-dialog"
        :class="{ 'gift-dialog--result': showResult }"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="showResult ? 'gift-result-title' : 'gift-offer-title'"
        tabindex="-1"
        @keydown.esc.stop.prevent="!showResult && emit('close')"
      >
        <h2 v-if="showResult" id="gift-result-title" class="sr-only">恭喜获得花瓣礼包</h2>
        <h2 v-else id="gift-offer-title" class="sr-only">花瓣特惠礼包</h2>

        <img
          class="gift-dialog__frame"
          :src="showResult ? '/image/activity/petal-gift/result.png' : '/image/activity/petal-gift/offer.png'"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <button
          v-if="showResult"
          class="gift-dialog__action gift-dialog__action--confirm"
          type="button"
          aria-label="确定收下100枚花瓣"
          @click="emit('confirm')"
        >
          <img src="/image/activity/petal-gift/confirm.png" alt="" aria-hidden="true" draggable="false">
        </button>
        <button
          v-else
          class="gift-dialog__action gift-dialog__action--purchase"
          type="button"
          :disabled="purchasing"
          :aria-label="purchasing ? '正在购买30元花瓣礼包' : '30元购买100枚花瓣礼包'"
          @click="emit('purchase')"
        >
          <img src="/image/activity/petal-gift/buy-30.png" alt="" aria-hidden="true" draggable="false">
        </button>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'

defineProps<{ showResult: boolean; purchasing: boolean }>()
const emit = defineEmits<{ close: []; purchase: []; confirm: [] }>()
const dialogRef = ref<HTMLElement | null>(null)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.gift-overlay { position: fixed; inset: 0; z-index: 1650; display: grid; padding: max(12px, env(safe-area-inset-top)) 10px max(12px, env(safe-area-inset-bottom)); place-items: center; background: rgba(10, 6, 30, 0.72); backdrop-filter: blur(3px); -webkit-backdrop-filter: blur(3px); }
.gift-dialog { position: relative; width: min(94vw, 430px); aspect-ratio: 606 / 738; outline: 0; transform-origin: center; animation: gift-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both; }
.gift-dialog--result { width: min(94vw, 430px); aspect-ratio: 1; }
.gift-dialog__frame { position: absolute; inset: 0; display: block; width: 100%; height: 100%; object-fit: contain; pointer-events: none; user-select: none; }
.gift-dialog__action { position: absolute; z-index: 2; right: 25.35%; width: 49.3%; padding: 0; background: transparent; border: 0; cursor: pointer; }
.gift-dialog__action--purchase { bottom: 5.2%; }
.gift-dialog__action--confirm { bottom: 8.8%; }
.gift-dialog__action img { display: block; width: 100%; height: auto; }
.gift-dialog__action:active { transform: scale(0.97); }
.gift-dialog__action:disabled { cursor: wait; opacity: 0.58; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@keyframes gift-dialog-expand {
  0% { opacity: 0; transform: scale(0.3); }
  72% { opacity: 1; transform: scale(1.03); }
  100% { opacity: 1; transform: scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .gift-dialog { animation: none; }
}
</style>
