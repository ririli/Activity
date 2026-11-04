<template>
  <Teleport to="body">
    <div class="flower-lit-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="flower-lit-dialog popup-center-expand"
        role="dialog"
        aria-modal="true"
        aria-labelledby="flower-lit-title"
        tabindex="-1"
      >
        <h2 id="flower-lit-title" class="sr-only">恭喜点亮</h2>
        <img
          class="flower-lit-dialog__frame"
          src="/image/activity/flowers/flower-lit-result-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <p class="flower-lit-dialog__message">
          恭喜您本次点亮<strong>{{ litFlowers }}</strong>朵花！<br>
          当前盒子已点亮花朵数量：<strong>{{ currentBoxFlowers }}</strong><br>
          距离获得戒指宝箱还需点亮<strong>{{ flowersUntilBox }}</strong>朵花
        </p>

        <button type="button" aria-label="确定" @click="$emit('confirm')">
          <img
            src="/image/activity/flowers/flower-lit-result-confirm.png"
            alt=""
            aria-hidden="true"
            draggable="false"
          >
        </button>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'

const props = defineProps<{
  litFlowers: number
  litCountAfter: number
}>()

defineEmits<{
  confirm: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
const currentBoxFlowers = computed(() => props.litCountAfter <= 0
  ? 0
  : ((props.litCountAfter - 1) % 6) + 1)
const flowersUntilBox = computed(() => 6 - currentBoxFlowers.value)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.flower-lit-overlay {
  position: fixed;
  inset: 0;
  z-index: 1600;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 12px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.flower-lit-dialog {
  position: relative;
  width: min(94vw, 470px);
  aspect-ratio: 654 / 480;
  outline: 0;
}

.flower-lit-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.flower-lit-dialog__message {
  position: absolute;
  top: 42%;
  right: 13%;
  left: 13%;
  z-index: 1;
  margin: 0;
  color: #fff;
  font-family: 'Source Han Serif CN', 'Songti SC', serif;
  font-size: clamp(14px, 4.1vw, 20px);
  font-weight: 700;
  line-height: 1.55;
  text-align: center;
  text-shadow: 0 2px 5px rgba(58, 28, 108, 0.65);
  transform: translateY(-50%);
}

.flower-lit-dialog__message strong {
  color: #fff4a9;
  font-size: 1.08em;
}

.flower-lit-dialog button {
  position: absolute;
  right: 31%;
  bottom: 9.5%;
  z-index: 2;
  width: 38%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.flower-lit-dialog button img {
  display: block;
  width: 100%;
  height: auto;
}

.flower-lit-dialog button:active {
  transform: scale(0.97);
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
</style>
