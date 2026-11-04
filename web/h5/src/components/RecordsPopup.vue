<template>
  <Teleport to="body">
    <div class="rp-overlay" @click.self="emit('close')">
      <section
        class="rp-panel"
        :class="{ 'rp-panel--visible': visible }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="records-popup-title"
      >
        <h2 id="records-popup-title" class="sr-only">抽奖记录</h2>
        <button class="rp-close-btn" type="button" aria-label="关闭抽奖记录" @click="emit('close')">
          <span aria-hidden="true"></span>
        </button>

        <img
          class="rp-tabs-art"
          :class="`rp-tabs-art--${activeTab}`"
          :src="activeTabArt"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <div class="rp-tabs" role="tablist" aria-label="抽奖记录类型">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            class="rp-tab"
            :class="{ 'rp-tab--active': activeTab === tab.key }"
            type="button"
            role="tab"
            :aria-selected="activeTab === tab.key"
            @click="activeTab = tab.key"
          >
            <span class="sr-only">{{ tab.label }}</span>
          </button>
        </div>

        <div class="rp-table" role="table" :aria-label="activeTabLabel">
          <div class="rp-table__head" role="row">
            <span class="sr-only" role="columnheader">时间</span>
            <span class="sr-only" role="columnheader">奖励</span>
          </div>

          <div class="rp-scroll" role="rowgroup" aria-label="抽奖记录滑动区域">
            <div v-if="loading" class="rp-state" role="status">加载中…</div>
            <div v-else-if="loadError" class="rp-state" role="alert">
              <span>记录加载失败</span>
              <button type="button" @click="loadAllDrawHistory">重新加载</button>
            </div>
            <div v-else-if="activeRows.length === 0" class="rp-state">{{ emptyText }}</div>

            <template v-else>
              <article v-for="row in activeRows" :key="row.id" class="rp-row" role="row">
                <time class="rp-row__time" :datetime="row.createdAt" role="cell">{{ formatTime(row.createdAt) }}</time>
                <span class="rp-row__reward" role="cell">{{ row.rewardText }}</span>
              </article>
            </template>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { fetchDrawHistory, type DrawHistoryItem } from '@/api/records'

const props = defineProps<{ initialPoolType: number }>()
const emit = defineEmits<{ close: [] }>()

const tabs = [
  { key: 'day', label: '白昼之旅' },
  { key: 'night', label: '星夜之旅' },
  { key: 'flower', label: '点亮花朵' },
] as const

type TabKey = typeof tabs[number]['key']

const tabArtByKey: Record<TabKey, string> = {
  day: '/image/activity/records/records-tabs-day.png',
  night: '/image/activity/records/records-tabs-night.png',
  flower: '/image/activity/records/records-tabs-flower.png',
}

interface RecordRow {
  id: string
  createdAt: string
  rewardText: string
}

const activeTab = ref<TabKey>(props.initialPoolType === 2 ? 'night' : 'day')
const visible = ref(false)
const drawHistory = ref<DrawHistoryItem[]>([])
const loading = ref(false)
const loadError = ref(false)

const dayRows = computed(() => buildRewardRows(1))
const nightRows = computed(() => buildRewardRows(2))
const flowerRows = computed(() => drawHistory.value
  .filter((item) => item.lit_flowers > 0)
  .map((item) => ({
    id: `flower-${item.draw_log_id}`,
    createdAt: item.created_at,
    rewardText: `花朵x${item.lit_flowers}`,
  })))

const activeRows = computed(() => {
  if (activeTab.value === 'day') return dayRows.value
  if (activeTab.value === 'night') return nightRows.value
  return flowerRows.value
})

const activeTabArt = computed(() => tabArtByKey[activeTab.value])
const activeTabLabel = computed(() => tabs.find((tab) => tab.key === activeTab.value)?.label ?? '抽奖记录')
const emptyText = computed(() => {
  if (activeTab.value === 'day') return '暂无白昼之旅记录'
  if (activeTab.value === 'night') return '暂无星夜之旅记录'
  return '暂无点亮花朵记录'
})

onMounted(() => {
  requestAnimationFrame(() => {
    visible.value = true
  })
  loadAllDrawHistory()
})

function buildRewardRows(poolType: number): RecordRow[] {
  return drawHistory.value
    .filter((item) => item.pool_type === poolType)
    .flatMap((item) => {
      if (item.rewards.length === 0) {
        return [{
          id: `${item.draw_log_id}-empty`,
          createdAt: item.created_at,
          rewardText: '奖励记录生成中',
        }]
      }

      return item.rewards.map((reward, index) => ({
        id: `${item.draw_log_id}-${index}`,
        createdAt: item.created_at,
        rewardText: formatRewardName(reward.item_name, reward.quantity),
      }))
    })
}

function formatRewardName(itemName: string, quantity: number) {
  if (quantity <= 1) return itemName
  return `${itemName}x${quantity}`
}

function formatTime(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso

  const month = date.getMonth() + 1
  const day = date.getDate()
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${month}月${day}日${hour}:${minute}`
}

async function loadAllDrawHistory() {
  loading.value = true
  loadError.value = false

  try {
    const all: DrawHistoryItem[] = []
    let page = 1
    let total = 0

    while (page <= 100) {
      const result = await fetchDrawHistory(page)
      if (page === 1) total = result.total
      all.push(...result.list)

      if (all.length >= result.total || result.list.length === 0) break
      page += 1
    }

    drawHistory.value = all.slice(0, total || all.length)
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
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

.rp-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  animation: rp-overlay-in 180ms ease-out;
}

.rp-panel {
  position: relative;
  width: min(100vw, 654px, calc((100dvh - 8px) * 0.913408));
  aspect-ratio: 654 / 716;
  max-height: calc(100dvh - 8px);
  padding: 0;
  overflow: hidden;
  color: #fff;
  background: url('/image/activity/records/records-popup-frame.png') center / 100% 100% no-repeat;
  filter: drop-shadow(0 8px 18px rgba(48, 27, 98, 0.22));
  transform: translateY(100%);
  transition: transform 360ms cubic-bezier(0.32, 0.72, 0, 1);
}

.rp-panel--visible {
  transform: translateY(0);
}

.rp-close-btn {
  position: absolute;
  top: 4.5%;
  right: 2.5%;
  z-index: 5;
  display: grid;
  width: 44px;
  height: 44px;
  padding: 0;
  place-items: center;
  background: transparent;
  border: 0;
  cursor: pointer;
  opacity: 0;
}

.rp-close-btn span,
.rp-close-btn span::after {
  display: block;
  width: 18px;
  height: 2px;
  background: #fff;
  border-radius: 999px;
  content: '';
}

.rp-close-btn span {
  transform: rotate(45deg);
}

.rp-close-btn span::after {
  transform: rotate(90deg);
}

.rp-tabs-art {
  position: absolute;
  left: 16.972%;
  z-index: 1;
  height: auto;
  pointer-events: none;
  user-select: none;
}

.rp-tabs-art--day {
  top: 10.894%;
  width: 66.208%;
}

.rp-tabs-art--night,
.rp-tabs-art--flower {
  top: 2.793%;
  width: 66.514%;
}

.rp-tabs {
  position: absolute;
  top: 13.687%;
  left: 16.972%;
  z-index: 3;
  display: grid;
  width: 66.514%;
  height: 7.263%;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.rp-tab {
  min-width: 0;
  min-height: 100%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  touch-action: manipulation;
}

.rp-table {
  position: absolute;
  top: 23.743%;
  left: 7.187%;
  z-index: 2;
  display: grid;
  width: 85.78%;
  height: 68.855%;
  grid-template-rows: 12.17% minmax(0, 1fr);
  overflow: hidden;
  border-radius: 8px;
}

.rp-table__head {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: center;
}

.rp-scroll {
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.rp-scroll::-webkit-scrollbar {
  display: none;
}

.rp-row {
  display: grid;
  min-height: min(13.02vw, 56px);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: center;
  color: rgba(255, 255, 255, 0.95);
  font-size: clamp(11px, 3.35vw, 16px);
  line-height: 1.24;
  text-align: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
}

.rp-row__time,
.rp-row__reward {
  min-width: 0;
  padding: 7px 10px;
  overflow-wrap: anywhere;
}

.rp-row__reward {
  display: -webkit-box;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.rp-state {
  display: grid;
  min-height: 180px;
  place-items: center;
  align-content: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.72);
  font-size: 14px;
  text-align: center;
}

.rp-state button {
  min-width: 92px;
  min-height: 36px;
  color: #762040;
  font-weight: 700;
  background: linear-gradient(180deg, #fff8ee 0%, #f5d8a6 100%);
  border: 1px solid rgba(255, 255, 255, 0.8);
  border-radius: 999px;
}

.rp-close-btn:focus-visible,
.rp-tab:focus-visible,
.rp-state button:focus-visible {
  outline: 3px solid rgba(255, 255, 255, 0.95);
  outline-offset: 2px;
}

@keyframes rp-overlay-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 350px) {
  .rp-row {
    min-height: 50px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .rp-overlay,
  .rp-panel {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
