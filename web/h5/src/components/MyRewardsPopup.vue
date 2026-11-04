<template>
  <Teleport to="body">
    <div class="my-rewards-overlay" @click.self="emit('close')" @keydown.esc.stop.prevent="emit('close')">
      <section
        ref="dialogRef"
        class="my-rewards-dialog popup-center-expand"
        role="dialog"
        aria-modal="true"
        aria-labelledby="my-rewards-title"
        tabindex="-1"
      >
        <h2 id="my-rewards-title" class="sr-only">我的奖励</h2>
        <img
          class="my-rewards-dialog__frame"
          src="/image/activity/my-rewards/popup-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <div v-if="loading" class="my-rewards-state" role="status">奖励加载中…</div>
        <div v-else-if="error" class="my-rewards-state" role="alert">
          <span>{{ error }}</span>
          <button type="button" @click="load">重新加载</button>
        </div>
        <div v-else-if="rewards.length === 0" class="my-rewards-state">暂无奖励</div>
        <ul v-else class="my-rewards-list" aria-label="我的奖励列表">
          <li v-for="reward in rewards" :key="reward.item_id">
            <RewardPreviewTile
              :item-id="reward.item_id"
              :item-name="reward.item_name"
              :quantity="reward.quantity"
              base-image-src="/image/activity/results/reward-item-base.png"
            />
          </li>
        </ul>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'
import RewardPreviewTile from '@/components/RewardPreviewTile.vue'
import { fetchMyRewards, type MyRewardItem } from '@/api/myRewards'

const emit = defineEmits<{ close: [] }>()
const dialogRef = ref<HTMLElement | null>(null)
const rewards = ref<MyRewardItem[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const result = await fetchMyRewards()
    rewards.value = [...(result.list ?? [])].reverse()
  } catch (cause: unknown) {
    error.value = cause instanceof Error ? cause.message : '奖励加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([load(), nextTick()])
  dialogRef.value?.focus()
})
</script>

<style scoped>
.my-rewards-overlay {
  position: fixed;
  inset: 0;
  z-index: 1650;
  display: grid;
  padding: max(10px, env(safe-area-inset-top)) 10px max(10px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.5);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.my-rewards-dialog {
  position: relative;
  width: min(94vw, 390px);
  max-height: 94dvh;
  aspect-ratio: 654 / 818;
  color: #fff;
  outline: 0;
}

.my-rewards-dialog__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
  user-select: none;
}

.my-rewards-list {
  position: absolute;
  top: 15%;
  right: 7.5%;
  bottom: 7%;
  left: 7.5%;
  z-index: 2;
  display: grid;
  min-height: 0;
  margin: 0;
  padding: 12px 4px 20px;
  overflow-x: hidden;
  overflow-y: auto;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  align-content: start;
  gap: 18px 8px;
  list-style: none;
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.my-rewards-list::-webkit-scrollbar { display: none; }
.my-rewards-list li { min-width: 0; }

.my-rewards-state {
  position: absolute;
  inset: 18% 10% 10%;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  flex-direction: column;
  color: rgba(255, 255, 255, 0.9);
  font-size: 15px;
  text-align: center;
}

.my-rewards-state button {
  min-width: 96px;
  height: 36px;
  color: #fff;
  background: rgba(73, 61, 170, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 6px;
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
