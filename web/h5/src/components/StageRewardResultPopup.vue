<template>
  <Teleport to="body">
    <div class="stage-result-overlay" @keydown.esc.stop.prevent>
      <section
        ref="dialogRef"
        class="stage-result-dialog popup-center-expand"
        role="dialog"
        aria-modal="true"
        aria-labelledby="stage-result-title"
        tabindex="-1"
      >
        <h2 id="stage-result-title" class="sr-only">恭喜获得</h2>
        <img
          class="stage-result-dialog__frame"
          src="/image/activity/stage/stage-reward-result-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <div class="stage-result-dialog__reward" aria-label="阶段奖励内容">
          <RewardPreviewTile
            :item-id="reward.item_id"
            :item-name="reward.item_name"
            :quantity="reward.quantity"
            base-image-src="/image/activity/results/reward-item-base.png"
          />
        </div>

        <button type="button" aria-label="确定" @click="$emit('confirm')">
          <img
            src="/image/activity/stage/stage-reward-result-confirm.png"
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
import { nextTick, onMounted, ref } from 'vue'
import RewardPreviewTile from '@/components/RewardPreviewTile.vue'
import type { ClaimStageRewardResult } from '@/api/stage'

defineProps<{
  reward: ClaimStageRewardResult
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
.stage-result-overlay {
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

.stage-result-dialog {
  position: relative;
  width: min(94vw, 470px);
  max-height: 94dvh;
  aspect-ratio: 1;
  outline: 0;
}

.stage-result-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.stage-result-dialog__reward {
  position: absolute;
  top: 28%;
  right: 31%;
  left: 31%;
  z-index: 1;
}

.stage-result-dialog button {
  position: absolute;
  right: 24.3%;
  bottom: 10.2%;
  z-index: 2;
  width: 51.4%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.stage-result-dialog button img {
  display: block;
  width: 100%;
  height: auto;
}

.stage-result-dialog button:active {
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
