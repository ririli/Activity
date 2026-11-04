<template>
  <Teleport to="body">
    <div class="brp-overlay">
      <section
        ref="dialogRef"
        class="brp-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="brp-title"
        tabindex="-1"
        @keydown.esc.stop.prevent="emit('close')"
      >
        <h2 id="brp-title" class="sr-only">确认购买花瓣</h2>
        <img
          class="brp-dialog__frame"
          src="/image/activity/purchase-confirm/frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <p class="brp-message">
          确认花费<strong>{{ costGold.toLocaleString() }}</strong>金币<br>
          购买<strong>{{ shortage.toLocaleString() }}</strong>枚花瓣吗？
        </p>
        <p class="sr-only">{{ poolName }}许愿{{ count }}次，共需{{ totalCost }}枚花瓣，当前拥有{{ petals }}枚花瓣。</p>

        <label class="brp-skip">
          <input v-model="dontShowAgain" type="checkbox">
          <span aria-hidden="true"></span>
          本次不再提示
        </label>

        <p v-if="!canAfford" class="brp-warning" role="alert">
          金币不足，当前{{ gold.toLocaleString() }}金币
        </p>

        <div class="brp-actions">
          <button type="button" aria-label="取消购买花瓣" @click="emit('close')">
            <img src="/image/activity/purchase-confirm/cancel.png" alt="取消" draggable="false">
          </button>
          <button
            type="button"
            :aria-label="canAfford ? '确认购买花瓣并许愿' : '金币不足，前往花瓣商店'"
            @click="handlePrimary"
          >
            <img src="/image/activity/purchase-confirm/confirm.png" :alt="canAfford ? '确定' : '前往商店'" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'

const props = defineProps<{
  poolName: string
  count: number
  costPerDraw: number
  shortage: number
  petals: number
  gold: number
}>()

const emit = defineEmits<{
  close: []
  confirm: [dontShowAgain: boolean]
  'go-shop': [dontShowAgain: boolean]
}>()

const dialogRef = ref<HTMLElement | null>(null)
const dontShowAgain = ref(false)
const totalCost = computed(() => props.costPerDraw * props.count)
const costGold = computed(() => props.shortage * 60)
const canAfford = computed(() => props.gold >= costGold.value)

function handlePrimary() {
  if (canAfford.value) emit('confirm', dontShowAgain.value)
  else emit('go-shop', dontShowAgain.value)
}

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.brp-overlay {
  position: fixed;
  inset: 0;
  z-index: 1600;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 12px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(11, 6, 29, 0.72);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}

.brp-dialog {
  position: relative;
  width: min(94vw, 436px);
  aspect-ratio: 654 / 480;
  outline: 0;
  animation: brp-dialog-in 220ms ease-out both;
}

.brp-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
}

.brp-message {
  position: absolute;
  z-index: 1;
  top: 42%;
  right: 9%;
  left: 9%;
  margin: 0;
  color: #fff;
  font-family: 'Source Han Serif CN', 'Songti SC', serif;
  font-size: clamp(15px, 4.2vw, 21px);
  font-weight: 700;
  line-height: 1.5;
  text-align: center;
  text-shadow: 0 2px 5px rgba(52, 27, 99, 0.58);
  transform: translateY(-50%);
}

.brp-message strong {
  margin-inline: 2px;
  color: #fff0a8;
  font-size: 1.05em;
}

.brp-skip {
  position: absolute;
  z-index: 2;
  top: 57%;
  right: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  color: rgba(255, 255, 255, 0.88);
  font-size: clamp(12px, 3.4vw, 15px);
  cursor: pointer;
}

.brp-skip input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  opacity: 0;
}

.brp-skip span {
  position: relative;
  width: 21px;
  height: 21px;
  flex: 0 0 auto;
  background: rgba(255, 242, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.94);
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(57, 28, 106, 0.38);
}

.brp-skip input:checked + span::after {
  position: absolute;
  top: 2px;
  left: 6px;
  width: 6px;
  height: 11px;
  content: '';
  border-right: 3px solid #9e61bb;
  border-bottom: 3px solid #9e61bb;
  transform: rotate(45deg);
}

.brp-skip input:focus-visible + span {
  outline: 2px solid #fff2b1;
  outline-offset: 2px;
}

.brp-warning {
  position: absolute;
  z-index: 2;
  top: 65%;
  right: 10%;
  left: 10%;
  margin: 0;
  color: #ffe0e7;
  font-size: 12px;
  text-align: center;
}

.brp-actions {
  position: absolute;
  z-index: 2;
  right: 8%;
  bottom: 8%;
  left: 8%;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 7%;
}

.brp-actions button {
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, transform 150ms ease;
}

.brp-actions button img {
  display: block;
  width: 100%;
  height: auto;
}

.brp-actions button:active { transform: scale(0.97); filter: brightness(1.08); }

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

@keyframes brp-dialog-in {
  from { opacity: 0; transform: translateY(10px) scale(0.96); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .brp-dialog { animation: none; }
}
</style>
