<template>
  <Teleport to="body">
    <div class="tbp-overlay" @click.self="handleClose" @keydown.esc.stop.prevent="handleClose">
      <section
        ref="dialogRef"
        class="tbp-dialog"
        :class="{ 'tbp-dialog--ring-rewards': showRingRewards }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="tbp-title"
        tabindex="-1"
      >
        <h2 id="tbp-title" class="sr-only">{{ showRingRewards ? '戒指召唤奖励' : '许愿奖励预览' }}</h2>
        <p class="sr-only">
          本次预览{{ preview.total_draws }}次白昼许愿，确认后支付{{ preview.total_cost }}花瓣并获得以下奖励。
        </p>
        <img
          class="tbp-dialog__frame"
          :src="showRingRewards
            ? '/image/activity/tbp-preview/ring-rewards-frame.png'
            : '/image/activity/tbp-preview/frame.png'"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <ul v-if="!showRingRewards" class="tbp-reward-list" aria-label="许愿奖励预览列表">
          <li v-for="reward in preview.reward_summary" :key="reward.item_id">
            <RewardPreviewTile
              :item-id="reward.item_id"
              :item-name="reward.item_name"
              :quantity="reward.total_quantity"
            />
          </li>
          <li
            v-if="preview.estimated_lit_flowers > 0"
            class="tbp-reward-list__flowers"
            :aria-label="`预计点亮花朵${preview.estimated_lit_flowers}朵`"
          >
            <RewardPreviewTile
              :item-id="-1"
              item-name="点亮花朵"
              :quantity="preview.estimated_lit_flowers"
              image-src="/image/activity/flowers/rose-bottom-center.png"
            />
          </li>
        </ul>

        <div v-if="!showRingRewards && hasRingPreview" class="tbp-ring-preview-action">
          <button type="button" aria-label="预览戒指召唤奖励" :disabled="confirming" @click="showRingRewards = true">
            <img src="/image/activity/tbp-preview/ring-preview.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>

        <div v-else class="tbp-actions">
          <button
            class="tbp-action tbp-action--cancel"
            type="button"
            aria-label="取消"
            :disabled="confirming"
            @click="handleClose"
          >
            <img src="/image/activity/tbp-preview/cancel.png" alt="" aria-hidden="true" draggable="false">
          </button>
          <button
            class="tbp-action tbp-action--confirm"
            type="button"
            :aria-label="confirming ? '正在支付900花瓣' : '900花瓣获得以上奖励'"
            :disabled="confirming"
            @click="$emit('confirm')"
          >
            <img src="/image/activity/tbp-preview/confirm.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import RewardPreviewTile from '@/components/RewardPreviewTile.vue'
import type { TryBeforePayPreview } from '@/api/tbp'

const props = defineProps<{
  preview: TryBeforePayPreview
  confirming: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
const showRingRewards = ref(false)
const hasRingPreview = computed(() => props.preview.estimated_lit_flowers >= 6)

watch(() => props.preview.session_id, () => {
  showRingRewards.value = false
})

function handleClose() {
  emit('close')
}

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.tbp-overlay {
  position: fixed;
  inset: 0;
  z-index: 1500;
  display: grid;
  padding: max(10px, env(safe-area-inset-top)) 10px max(10px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.tbp-dialog {
  position: relative;
  width: min(94vw, 370px);
  max-height: 94dvh;
  aspect-ratio: 654 / 896;
  color: #fff;
  outline: 0;
  transform-origin: center;
  animation: tbp-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.tbp-dialog--ring-rewards {
  animation-name: tbp-ring-dialog-expand;
}

.tbp-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
  user-select: none;
}

.tbp-reward-list {
  position: absolute;
  top: 15.5%;
  right: 8.5%;
  bottom: 17.5%;
  left: 8.5%;
  z-index: 1;
  display: grid;
  min-height: 0;
  margin: 0;
  padding: 7px 4px 16px;
  overflow-x: hidden;
  overflow-y: auto;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-content: start;
  gap: 17px 8px;
  list-style: none;
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.tbp-reward-list::-webkit-scrollbar {
  display: none;
}

.tbp-reward-list li {
  min-width: 0;
}

.tbp-actions {
  position: absolute;
  right: 8.7%;
  bottom: 3.7%;
  left: 8.7%;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2.5%;
}

.tbp-dialog--ring-rewards .tbp-actions {
  transform: translateY(-20px);
}

.tbp-action {
  flex: 0 0 auto;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, transform 150ms ease, opacity 150ms ease;
}

.tbp-action--cancel {
  width: 36.3%;
}

.tbp-action--confirm {
  width: 60.7%;
}

.tbp-action img {
  display: block;
  width: 100%;
  height: auto;
}

.tbp-ring-preview-action {
  position: absolute;
  bottom: 3.5%;
  left: 50%;
  z-index: 2;
  width: 42.2%;
  transform: translateX(-50%);
}

.tbp-ring-preview-action button {
  width: 100%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 150ms ease, opacity 150ms ease, transform 150ms ease;
}

.tbp-ring-preview-action img {
  display: block;
  width: 100%;
  height: auto;
}

.tbp-ring-preview-action button:active:not(:disabled) { transform: scale(0.98); filter: brightness(1.05); }
.tbp-ring-preview-action button:disabled { cursor: wait; opacity: 0.58; }

.tbp-action:active {
  transform: scale(0.97);
}

.tbp-action:disabled {
  cursor: wait;
  opacity: 0.58;
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

@keyframes tbp-dialog-expand {
  0% { opacity: 0; transform: scale(0.3); }
  72% { opacity: 1; transform: scale(1.03); }
  100% { opacity: 1; transform: scale(1); }
}

@keyframes tbp-ring-dialog-expand {
  0% { opacity: 0; transform: scale(0.3); }
  72% { opacity: 1; transform: scale(1.03); }
  100% { opacity: 1; transform: scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .tbp-dialog {
    animation: none;
  }
}

@media (max-height: 560px) and (orientation: landscape) {
  .tbp-dialog {
    width: auto;
    height: 94dvh;
  }
}
</style>
