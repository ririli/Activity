<template>
  <Teleport to="body">
    <div class="psp-overlay" @click.self="emit('close')">
      <section
        class="psp-panel"
        :class="{ 'psp-panel--visible': visible }"
        :style="panelStyle"
        role="dialog"
        aria-modal="true"
        aria-labelledby="petal-shop-title"
      >
        <h2 id="petal-shop-title" class="sr-only">获取花瓣</h2>

        <button class="psp-close" type="button" aria-label="关闭花瓣购买弹窗" @click="emit('close')">
          <span aria-hidden="true"></span>
        </button>

        <div class="psp-tabs" role="tablist" aria-label="花瓣获取方式">
          <button
            class="psp-tab psp-tab--shop"
            :class="{ 'psp-tab--active': activeTab === 'shop' }"
            type="button"
            role="tab"
            :aria-selected="activeTab === 'shop'"
            @click="activeTab = 'shop'"
          >
            <span class="sr-only">获取花瓣</span>
          </button>
          <button
            class="psp-tab psp-tab--tasks"
            :class="{ 'psp-tab--active': activeTab === 'tasks' }"
            type="button"
            role="tab"
            :aria-selected="activeTab === 'tasks'"
            @click="activeTab = 'tasks'"
          >
            <span class="sr-only">每日任务</span>
          </button>
        </div>

        <div class="psp-assets" :class="{ 'psp-assets--tasks': activeTab === 'tasks' }" aria-label="当前资产">
          <div class="psp-asset psp-asset--petal" aria-label="当前花瓣余额">
            <span class="sr-only">当前花瓣余额</span>
            <strong>{{ balance.petals.toLocaleString() }}</strong>
          </div>
          <div v-if="activeTab === 'shop'" class="psp-asset psp-asset--gold" aria-label="当前金币余额">
            <span class="sr-only">当前金币余额</span>
            <strong :class="goldBalanceSizeClass">{{ balance.gold.toLocaleString() }}</strong>
          </div>
        </div>
        <p v-if="activeTab === 'tasks'" class="psp-refresh">每日00:00刷新</p>

        <div class="psp-content">
          <template v-if="activeTab === 'shop'">
            <div class="psp-package-list">
              <article
                v-for="pkg in packages"
                :key="pkg.tier"
                class="psp-package"
                :style="{ '--package-index': packageIndex(pkg) }"
              >
                <span class="sr-only">{{ pkg.tier.toLocaleString() }}枚花瓣，{{ packagePrice(pkg).toLocaleString() }}金币</span>

                <button
                  class="psp-buy"
                  :class="{ 'psp-buy--purchased': isPurchased(pkg) }"
                  type="button"
                  :aria-label="getBuyAriaLabel(pkg)"
                  :disabled="pkg.loading || isGoldInsufficient(pkg) || isPurchased(pkg)"
                  @click="handleBuy(pkg)"
                >
                  <img :src="isPurchased(pkg) ? '/image/activity/petal-shop/buy-disabled.png' : '/image/activity/petal-shop/buy.png'" alt="" aria-hidden="true">
                  <span class="sr-only">{{ getBuyButtonText(pkg) }}</span>
                </button>
                <img
                  v-if="getBonusAsset(pkg) && !isPurchased(pkg)"
                  class="psp-package__bonus"
                  :class="`psp-package__bonus--${pkg.tier}`"
                  :src="getBonusAsset(pkg)"
                  alt=""
                  aria-hidden="true"
                >
              </article>
            </div>
            <p v-if="buyErr" class="psp-buy-err" role="alert">{{ buyErr }}</p>
          </template>

          <div v-else class="psp-task-panel" role="tabpanel" aria-label="每日任务">
            <div class="psp-task-list">
              <article v-for="task in dailyTasks" :key="task.title" class="psp-task">
                <div class="psp-task__visual" aria-hidden="true">
                  <img src="/image/activity/petal.png" alt="">
                  <span>x1</span>
                </div>
                <h3>{{ task.title }}</h3>
                <div class="psp-task__action">
                  <button type="button" @click="showTaskPending">领取</button>
                  <span>领取次数：0/1</span>
                </div>
              </article>
            </div>
          </div>
        </div>
      </section>
    </div>
    <div v-if="purchaseSuccessNotice" class="psp-success-notice" role="status" aria-live="polite">
      {{ purchaseSuccessNotice }}
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { showToast } from 'vant'
import { buyPetals, getAssets, getPetalShopStatus } from '@/api/asset'

const emit = defineEmits<{ close: []; 'balance-change': [] }>()

interface PetalPackage {
  tier: number
  costGold: number
  visualPetals: number
  loading: boolean
}

const PACKAGE_DEFS: Array<Omit<PetalPackage, 'loading'>> = [
  { tier: 5, costGold: 300, visualPetals: 1 },
  { tier: 52, costGold: 3000, visualPetals: 2 },
  { tier: 525, costGold: 30000, visualPetals: 4 },
  { tier: 1580, costGold: 90000, visualPetals: 6 },
]

const dailyTasks = [
  { title: '每日登录app' },
  { title: '在语音派对送出天鹅之舞礼物' },
  { title: '在语音派对玩25分钟' },
  { title: '每日参与并结算一次龙主对决游戏' },
  { title: '每日登入app' },
]

const packages = reactive<PetalPackage[]>(
  PACKAGE_DEFS.map((pkg) => ({ ...pkg, loading: false }))
)

const activeTab = ref<'shop' | 'tasks'>('shop')
const balance = reactive({ petals: 0, gold: 0 })
const balanceLoaded = ref(false)
const purchasedTiers = ref<Set<number>>(new Set())
const visible = ref(false)
const buyErr = ref('')
const purchaseSuccessNotice = ref('')
let purchaseSuccessNoticeTimer: ReturnType<typeof setTimeout> | null = null
const panelStyle = computed(() => ({
  '--psp-bg': `url('/image/activity/petal-shop-bg-${activeTab.value === 'shop' ? 'shop' : 'tasks'}.png')`,
}))
const goldBalanceSizeClass = computed(() => {
  const length = balance.gold.toLocaleString().length
  if (length >= 12) return 'psp-asset__value--extra-long'
  if (length >= 10) return 'psp-asset__value--long'
  return ''
})

onMounted(() => {
  requestAnimationFrame(() => {
    visible.value = true
  })
})

onMounted(loadBalance)
onMounted(loadPurchaseStatus)

onUnmounted(() => {
  if (purchaseSuccessNoticeTimer) clearTimeout(purchaseSuccessNoticeTimer)
})

function packagePrice(pkg: PetalPackage) {
  return pkg.costGold
}

function packageIndex(pkg: PetalPackage) {
  return PACKAGE_DEFS.findIndex((item) => item.tier === pkg.tier)
}

function getBonusAsset(pkg: PetalPackage) {
  const assets: Record<number, string> = {
    52: '/image/activity/petal-shop/bonus-2.png',
    525: '/image/activity/petal-shop/bonus-25.png',
    1580: '/image/activity/petal-shop/bonus-80.png',
  }
  return assets[pkg.tier]
}

function isGoldInsufficient(pkg: PetalPackage) {
  return balanceLoaded.value && balance.gold < packagePrice(pkg)
}

function isPurchased(pkg: PetalPackage) {
  return purchasedTiers.value.has(pkg.tier)
}

function getBuyButtonText(pkg: PetalPackage) {
  if (pkg.loading) return '购买中'
  if (isPurchased(pkg)) return '已购买'
  if (isGoldInsufficient(pkg)) return '金币不足'
  return '购买'
}

function getBuyAriaLabel(pkg: PetalPackage) {
  if (pkg.loading) return `购买${pkg.tier.toLocaleString()}枚花瓣，购买中`
  if (isPurchased(pkg)) return `${pkg.tier.toLocaleString()}枚花瓣赠送档位，已购买`
  if (isGoldInsufficient(pkg)) return `购买${pkg.tier.toLocaleString()}枚花瓣，金币不足`
  return `购买${pkg.tier.toLocaleString()}枚花瓣，消耗${packagePrice(pkg).toLocaleString()}金币`
}

function getErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : '购买失败'
}

async function loadBalance() {
  try {
    const res = await getAssets()
    balance.petals = res.petals
    balance.gold = res.gold
    balanceLoaded.value = true
  } catch {
    balanceLoaded.value = false
  }
}

async function loadPurchaseStatus() {
  try {
    const result = await getPetalShopStatus()
    purchasedTiers.value = new Set(result.purchased_tiers)
  } catch {
    purchasedTiers.value = new Set()
  }
}

async function handleBuy(pkg: PetalPackage) {
  if (pkg.loading || isPurchased(pkg)) return

  if (isGoldInsufficient(pkg)) {
    buyErr.value = '金币不足，请选择较小档位。'
    showToast('金币不足，请选择较小档位')
    return
  }

  pkg.loading = true
  buyErr.value = ''
  try {
    await buyPetals(pkg.tier)
    await loadBalance()
    if (getBonusAsset(pkg)) {
      purchasedTiers.value = new Set([...purchasedTiers.value, pkg.tier])
    }
    emit('balance-change')
    showPurchaseSuccessNotice()
  } catch (error: unknown) {
    buyErr.value = getErrorMessage(error)
    if (buyErr.value.includes('限购一次')) await loadPurchaseStatus()
    showToast(buyErr.value)
  } finally {
    pkg.loading = false
  }
}

function showPurchaseSuccessNotice() {
  purchaseSuccessNotice.value = '花瓣购买成功，快去许愿吧！'
  if (purchaseSuccessNoticeTimer) clearTimeout(purchaseSuccessNoticeTimer)
  purchaseSuccessNoticeTimer = setTimeout(() => {
    purchaseSuccessNotice.value = ''
    purchaseSuccessNoticeTimer = null
  }, 2200)
}

function showTaskPending() {
  showToast('每日任务暂未开放')
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

.psp-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
  animation: psp-overlay-in 180ms ease-out;
}

.psp-panel {
  position: relative;
  width: min(100%, 430px);
  aspect-ratio: 750 / 930;
  height: auto;
  max-height: calc(100dvh - 16px);
  overflow: hidden;
  border-radius: 14px 14px 0 0;
  color: #fff;
  isolation: isolate;
  transform: translateY(100%);
  transition: transform 360ms cubic-bezier(0.32, 0.72, 0, 1);
}

.psp-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -2;
  background: var(--psp-bg) top center / 100% 100% no-repeat;
}

.psp-panel--visible {
  transform: translateY(0);
}

.psp-success-notice {
  position: fixed;
  top: 50%;
  left: 50%;
  z-index: 1200;
  max-width: min(76vw, 310px);
  padding: 14px 20px;
  color: #fff;
  font-size: 16px;
  line-height: 1.45;
  text-align: left;
  white-space: normal;
  background: rgba(35, 33, 33, 0.9);
  border-radius: 8px;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
  pointer-events: none;
  transform: translate(-50%, -50%);
}

.psp-close {
  position: absolute;
  z-index: 8;
  top: 12px;
  right: 10px;
  display: grid;
  width: 44px;
  height: 44px;
  padding: 0;
  place-items: center;
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  opacity: 0;
}

.psp-close span,
.psp-close span::after {
  display: block;
  width: 17px;
  height: 2px;
  background: #9a64b7;
  border-radius: 999px;
  content: '';
}

.psp-close span {
  transform: rotate(45deg);
}

.psp-close span::after {
  transform: rotate(90deg);
}

.psp-tabs {
  position: absolute;
  z-index: 5;
  top: 11.83%;
  left: 50%;
  display: grid;
  width: min(72%, 312px);
  height: 64px;
  grid-template-columns: 1fr 1fr;
  align-items: center;
  transform: translateX(-50%);
}

.psp-tabs::before {
  content: none;
}

.psp-tab {
  position: relative;
  z-index: 2;
  height: 56px;
  min-width: 0;
  margin: 0;
  padding: 0;
  overflow: hidden;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.psp-assets {
  position: absolute;
  z-index: 4;
  top: 24.62%;
  right: 8.37%;
  left: 8.37%;
  height: 34px;
}

.psp-assets--tasks {
  right: auto;
  width: 38%;
}

.psp-asset {
  position: absolute;
  top: 0;
  display: flex;
  height: 34px;
  min-width: 0;
  align-items: center;
  color: #fffaf7;
  font-size: clamp(13px, 3.8vw, 16px);
  font-weight: 700;
  line-height: 1;
  border-radius: 18px;
  text-shadow: 0 1px 3px rgba(60, 44, 124, 0.35);
}

.psp-asset--petal {
  left: 0;
  width: 48%;
  justify-content: flex-start;
  padding-left: min(17.21vw, 74px);
}

.psp-asset--gold {
  right: 0;
  width: 47%;
  justify-content: flex-start;
  padding-left: min(14.88vw, 64px);
  padding-right: min(1.4vw, 6px);
  font-size: clamp(12px, 3.26vw, 14px);
}

.psp-assets--tasks .psp-asset--petal {
  width: 100%;
  padding-left: min(17.21vw, 74px);
}

.psp-asset strong {
  min-width: 0;
  overflow: hidden;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 34px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.psp-asset--petal strong {
  transform: translateY(-4px);
}

.psp-asset--gold strong {
  transform: translate(3px, -4px);
}

.psp-asset--gold .psp-asset__value--long {
  font-size: 10px;
  transform: translate(0, -4px);
}

.psp-asset--gold .psp-asset__value--extra-long {
  font-size: 9px;
  transform: translate(0, -4px);
}

.psp-price-coin {
  display: inline-grid;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 50%;
  background: radial-gradient(circle at 38% 30%, #fff4be 0 20%, #ffc344 42%, #e48418 78%);
  border: 2px solid rgba(255, 232, 148, 0.95);
  box-shadow: 0 2px 4px rgba(125, 72, 26, 0.32);
}

.psp-price-coin::before {
  width: 52%;
  height: 52%;
  background: #fff;
  clip-path: polygon(50% 0, 64% 35%, 100% 38%, 72% 60%, 82% 100%, 50% 78%, 18% 100%, 28% 60%, 0 38%, 36% 35%);
  content: '';
  opacity: 0.82;
}

.psp-content {
  position: absolute;
  z-index: 4;
  top: 36.56%;
  right: 7.47%;
  bottom: 2.15%;
  left: 7.47%;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 2px 2px 8px;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.psp-content::-webkit-scrollbar {
  display: none;
}

.psp-package-list {
  display: grid;
  gap: 0;
}

.psp-package {
  position: relative;
  width: 100%;
  aspect-ratio: 638 / 124;
  min-height: 0;
  padding: 0;
  text-align: left;
  background-image: url('/image/activity/petal-shop/packages.png');
  background-repeat: no-repeat;
  background-position: center calc(var(--package-index) * 33.333333%);
  background-size: 100% 400%;
  border: 0;
  border-radius: 0;
}

.psp-buy {
  position: absolute;
  z-index: 2;
  top: 50%;
  right: 1.5%;
  width: 25.4%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: opacity 180ms ease, filter 180ms ease, transform 180ms ease;
  transform: translateY(-50%);
}

.psp-buy img { display: block; width: 100%; height: auto; }

.psp-buy:active {
  filter: brightness(1.06);
  transform: translateY(-50%) scale(0.96);
}

.psp-buy:disabled {
  cursor: not-allowed;
  opacity: 0.58;
  filter: grayscale(0.12);
}

.psp-buy--purchased:disabled {
  opacity: 1;
  filter: none;
}

.psp-package__bonus {
  position: absolute;
  z-index: 3;
  right: 0;
  bottom: 52%;
  width: auto;
  height: clamp(19px, 5.1vw, 22px);
  object-fit: contain;
  pointer-events: none;
}

.psp-package__bonus--525 {
  right: -1.5%;
  height: clamp(30px, 8vw, 35px);
}

.psp-package__bonus--52,
.psp-package__bonus--1580 {
  transform: translateY(-10px);
}

.psp-buy-err {
  margin: 10px 4px 0;
  color: #a53655;
  font-size: 12px;
  text-align: center;
}

.psp-close:focus-visible,
.psp-tab:focus-visible,
.psp-buy:focus-visible,
.psp-task__action button:focus-visible {
  outline: 3px solid rgba(255, 255, 255, 0.95);
  outline-offset: 2px;
}

.psp-task-panel {
  position: relative;
  padding-top: 0;
}

.psp-refresh {
  position: absolute;
  top: 25.59%;
  right: 10%;
  z-index: 5;
  margin: 0;
  color: #352568;
  font-size: clamp(15px, 4.6vw, 18px);
  font-weight: 700;
  line-height: 1;
}

.psp-task-list {
  display: grid;
  gap: 8px;
}

.psp-task {
  display: grid;
  min-height: clamp(64px, 19.53vw, 84px);
  grid-template-columns: clamp(52px, 16.28vw, 70px) minmax(0, 1fr) clamp(80px, 21.4vw, 92px);
  align-items: center;
  gap: clamp(6px, 1.86vw, 8px);
  padding: clamp(8px, 2.33vw, 10px) clamp(6px, 1.86vw, 8px) clamp(8px, 2.33vw, 10px) clamp(10px, 3.26vw, 14px);
  background: linear-gradient(180deg, #fff 0%, #fff0f5 100%);
  border-radius: 17px;
}

.psp-task__visual {
  position: relative;
  display: grid;
  width: clamp(50px, 13.49vw, 58px);
  height: clamp(50px, 13.49vw, 58px);
  place-items: center;
  overflow: hidden;
  background: linear-gradient(180deg, #fff 0%, #ffeef3 100%);
  border-radius: 0 8px 0 8px;
}

.psp-task__visual img {
  width: clamp(38px, 10.23vw, 44px);
  height: clamp(38px, 10.23vw, 44px);
  object-fit: contain;
}

.psp-task__visual span {
  position: absolute;
  right: -2px;
  bottom: -2px;
  color: #c53c37;
  font-size: 17px;
  font-weight: 800;
  -webkit-text-stroke: 1px #fffbef;
}

.psp-task h3 {
  margin: 0;
  color: #4b3592;
  font-size: clamp(14px, 4.1vw, 18px);
  font-weight: 600;
  line-height: 1.25;
}

.psp-task__action {
  display: grid;
  gap: 4px;
  justify-items: center;
}

.psp-task__action button {
  width: clamp(78px, 20.47vw, 88px);
  height: clamp(32px, 8.14vw, 35px);
  color: #fff;
  font-size: clamp(16px, 4.19vw, 18px);
  font-weight: 700;
  background: linear-gradient(180deg, #ff929d 0%, #ff6f85 100%);
  border: 2px solid rgba(255, 235, 238, 0.95);
  border-radius: 999px;
  box-shadow: inset 0 1px rgba(255, 255, 255, 0.58), 0 2px 4px rgba(170, 91, 130, 0.3);
  cursor: pointer;
}

.psp-task__action span {
  color: #8e87bf;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

@keyframes psp-overlay-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@media (max-width: 350px) {
  .psp-tabs { width: 75%; }
  .psp-assets { right: 6.55%; left: 6.55%; gap: 15%; }
  .psp-assets--tasks { right: auto; width: 42%; }
  .psp-content { right: 7.78%; left: 7.78%; }
  .psp-buy { right: 1%; }
  .psp-task { grid-template-columns: 58px minmax(0, 1fr) 82px; padding-inline: 10px 6px; }
  .psp-task__visual { width: 50px; height: 50px; }
  .psp-task__action button { width: 78px; font-size: 16px; }
  .psp-task__action span { font-size: 11px; }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
