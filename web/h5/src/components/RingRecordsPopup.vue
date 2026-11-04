<template>
  <Teleport to="body">
    <div class="rrp-overlay" @click.self="emit('close')">
      <section
        class="rrp-panel"
        :class="{ 'rrp-panel--visible': visible }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="ring-records-popup-title"
      >
        <h2 id="ring-records-popup-title" class="sr-only">戒指召唤记录</h2>

        <img
          class="rrp-frame"
          src="/image/activity/records/ring-records-popup-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >

        <button class="rrp-close-btn" type="button" aria-label="关闭戒指召唤记录" @click="emit('close')">
          <span aria-hidden="true"></span>
        </button>

        <div class="rrp-table" role="table" aria-label="戒指召唤记录">
          <div class="rrp-table__head" role="row">
            <span class="sr-only" role="columnheader">时间</span>
            <span class="sr-only" role="columnheader">奖励</span>
          </div>

          <div class="rrp-scroll" role="rowgroup" aria-label="戒指召唤记录滑动区域">
            <div v-if="loading" class="rrp-state" role="status">加载中…</div>
            <div v-else-if="loadError" class="rrp-state" role="alert">
              <span>记录加载失败</span>
              <button type="button" @click="loadAllRingRecords">重新加载</button>
            </div>
            <div v-else-if="records.length === 0" class="rrp-state">暂无戒指召唤记录</div>

            <template v-else>
              <article v-for="item in records" :key="item.id" class="rrp-row" role="row">
                <time class="rrp-row__time" :datetime="item.created_at" role="cell">
                  {{ formatTime(item.created_at) }}
                </time>
                <span class="rrp-row__reward" role="cell">{{ item.item_name }}</span>
              </article>
            </template>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchRingRecords, type RingRecordItem } from '@/api/records'

const emit = defineEmits<{ close: [] }>()

const visible = ref(false)
const records = ref<RingRecordItem[]>([])
const loading = ref(false)
const loadError = ref(false)

onMounted(() => {
  requestAnimationFrame(() => {
    visible.value = true
  })
  loadAllRingRecords()
})

function formatTime(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso

  const month = date.getMonth() + 1
  const day = date.getDate()
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${month}月${day}日${hour}:${minute}`
}

async function loadAllRingRecords() {
  loading.value = true
  loadError.value = false

  try {
    const all: RingRecordItem[] = []
    let page = 1
    let total = 0

    while (page <= 100) {
      const result = await fetchRingRecords(page)
      if (page === 1) total = result.total
      all.push(...result.list)

      if (all.length >= result.total || result.list.length === 0) break
      page += 1
    }

    records.value = all.slice(0, total || all.length)
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

.rrp-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  animation: rrp-overlay-in 180ms ease-out;
}

.rrp-panel {
  position: relative;
  width: min(100vw, 654px, calc((100dvh - 8px) * 0.913408));
  aspect-ratio: 654 / 716;
  max-height: calc(100dvh - 8px);
  overflow: hidden;
  color: #fff;
  filter: drop-shadow(0 8px 18px rgba(48, 27, 98, 0.22));
  transform: translateY(100%);
  transition: transform 360ms cubic-bezier(0.32, 0.72, 0, 1);
}

.rrp-panel--visible {
  transform: translateY(0);
}

.rrp-frame {
  position: absolute;
  inset: 0;
  z-index: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.rrp-close-btn {
  position: absolute;
  top: 4.5%;
  right: 2.5%;
  z-index: 3;
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

.rrp-close-btn span,
.rrp-close-btn span::after {
  display: block;
  width: 18px;
  height: 2px;
  background: #fff;
  border-radius: 999px;
  content: '';
}

.rrp-close-btn span {
  transform: rotate(45deg);
}

.rrp-close-btn span::after {
  transform: rotate(90deg);
}

.rrp-table {
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

.rrp-table__head {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  align-items: center;
}

.rrp-scroll {
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.rrp-scroll::-webkit-scrollbar {
  display: none;
}

.rrp-row {
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

.rrp-row__time,
.rrp-row__reward {
  min-width: 0;
  padding: 7px 10px;
  overflow-wrap: anywhere;
}

.rrp-row__reward {
  display: -webkit-box;
  overflow: hidden;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.rrp-state {
  display: grid;
  min-height: 180px;
  place-items: center;
  align-content: center;
  gap: 12px;
  color: rgba(255, 255, 255, 0.72);
  font-size: 14px;
  text-align: center;
}

.rrp-state button {
  min-width: 92px;
  min-height: 36px;
  color: #762040;
  font-weight: 700;
  background: linear-gradient(180deg, #fff8ee 0%, #f5d8a6 100%);
  border: 1px solid rgba(255, 255, 255, 0.8);
  border-radius: 999px;
}

.rrp-close-btn:focus-visible,
.rrp-state button:focus-visible {
  outline: 3px solid rgba(255, 255, 255, 0.95);
  outline-offset: 2px;
}

@keyframes rrp-overlay-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 350px) {
  .rrp-row {
    min-height: 50px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .rrp-overlay,
  .rrp-panel {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
