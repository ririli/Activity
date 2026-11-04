<template>
  <main class="leaderboard-page">
    <svg class="icon-sprite" aria-hidden="true">
      <symbol id="ranking-gift" viewBox="0 0 24 24">
        <path d="M4 10h16v10H4zM3 7h18v4H3zM12 7v13M8.1 7C6.5 7 5 6.2 5 4.8 5 3.8 5.8 3 6.9 3 9.1 3 12 7 12 7M15.9 7C17.5 7 19 6.2 19 4.8 19 3.8 18.2 3 17.1 3 14.9 3 12 7 12 7" />
      </symbol>
    </svg>

    <div class="lb-scroll" data-testid="leaderboard-scroll" :inert="namingDialogOpen">
      <section class="reward-scene" aria-label="榜单奖励">
        <header class="lb-nav" aria-label="页面导航">
          <img class="lb-nav__backdrop" src="/image/activity/nav-bar.svg" alt="" aria-hidden="true">
          <button class="lb-nav__button lb-nav__button--back" type="button" aria-label="返回" @click="router.back()">
            <img src="/image/activity/nav-back.svg" alt="">
          </button>
          <div class="lb-nav__actions">
            <button
              class="lb-nav__button"
              type="button"
              :aria-label="activityAudioEnabled ? '关闭背景音乐' : '开启背景音乐'"
              @click="toggleActivityAudio"
            >
              <img :src="activityAudioEnabled ? '/image/activity/nav-music.png' : '/image/activity/nav-music-off.png'" alt="">
            </button>
            <button
              class="lb-nav__button"
              :class="{ 'lb-nav__button--loading': isRefreshing }"
              type="button"
              aria-label="刷新排行榜数据"
              :disabled="isRefreshing"
              @click="handleRefresh"
            >
              <img src="/image/activity/nav-refresh.svg" alt="">
            </button>
          </div>
        </header>

        <button
          ref="namingGuideButton"
          class="naming-guide-button"
          type="button"
          aria-label="查看冠名定制说明"
          aria-haspopup="dialog"
          @click="openNamingDialog"
        >
          <img
            class="naming-guide"
            src="/image/leaderboard/naming-guide.png"
            alt="冠名定制说明"
          >
        </button>
        <img
          class="reward-panel"
          src="/image/leaderboard/reward-panel.png"
          alt=""
        >
        <img
          class="reward-content"
          src="/image/leaderboard/reward-content.png"
          alt=""
        >
        <div class="reward-assets" aria-label="榜单奖励道具展示">
          <picture class="reward-asset reward-asset--first-naming">
            <source srcset="/image/leaderboard/rewards/spring-princess-naming.svg" type="image/svg+xml">
            <img src="/image/leaderboard/rewards/spring-princess-naming.png" alt="春日公主冠名头像框90天">
          </picture>
          <img
            class="reward-asset reward-asset--second-title"
            src="/image/leaderboard/rewards/spring-princess-title.png"
            alt="春日公主称号60天"
          >
          <img
            class="reward-asset reward-asset--second-frame"
            src="/image/leaderboard/rewards/spring-princess-frame.png"
            alt="春日公主头像框60天"
          >
          <img
            class="reward-asset reward-asset--fourth-title"
            src="/image/leaderboard/rewards/spring-princess-title.png"
            alt="春日公主称号30天"
          >
          <img
            class="reward-asset reward-asset--fourth-frame"
            src="/image/leaderboard/rewards/spring-princess-frame.png"
            alt="春日公主头像框30天"
          >
          <img
            class="reward-asset reward-asset--eleventh-title"
            src="/image/leaderboard/rewards/spring-princess-title.png"
            alt="春日公主称号10天"
          >
        </div>

        <div class="sr-only">
          <h1>榜单奖励</h1>
          <p>活动期间每使用1枚花瓣，获得1点榜单值，按照活动期间用户获得的榜单值排名。</p>
          <p>第1名获得定制称号90天或修改V8称号、春日公主冠名头像框90天。</p>
          <p>第2至3名获得春日公主称号及头像框60天。</p>
          <p>第4至10名获得春日公主称号及头像框30天。</p>
          <p>第11至20名获得春日公主称号10天。</p>
        </div>
      </section>

      <section class="ranking-section" aria-label="总榜排名">
        <div v-if="loading" class="ranking-loading" aria-live="polite" aria-label="排行榜加载中">
          <div class="podium-skeleton"></div>
          <div v-for="n in 5" :key="n" class="rank-skeleton"></div>
        </div>

        <div v-else-if="error" class="ranking-state" role="alert">
          <p>{{ error }}</p>
          <button class="retry-button" type="button" @click="load">重新加载</button>
        </div>

        <div v-else-if="top20.length === 0" class="ranking-state">
          <p>暂无排名数据</p>
          <span>参与许愿后将自动生成排名</span>
        </div>

        <template v-else>
          <div class="podium" aria-label="排行榜前三名">
            <article v-if="secondPlace" class="podium-player podium-player--second">
              <div class="podium-profile">
                <img class="podium-avatar" src="/image/leaderboard/profile/default-avatar.png" :alt="`${secondPlace.nickname}头像`">
                <img class="podium-crown podium-crown--silver" src="/image/leaderboard/profile/crown-silver.png" alt="" aria-hidden="true">
              </div>
              <div class="podium-base podium-base--second">
                <strong class="podium-place">NO.2</strong>
                <p class="podium-nickname" :title="secondPlace.nickname">{{ secondPlace.nickname }}</p>
                <p class="podium-score">
                  <svg class="gift-icon" aria-hidden="true"><use href="#ranking-gift" /></svg>
                  <span>{{ fmtScore(secondPlace.score) }}</span>
                </p>
              </div>
            </article>

            <article v-if="firstPlace" class="podium-player podium-player--first">
              <div class="podium-profile">
                <img class="podium-avatar" src="/image/leaderboard/profile/default-avatar.png" :alt="`${firstPlace.nickname}头像`">
                <img class="podium-crown podium-crown--gold" src="/image/leaderboard/profile/crown-gold.png" alt="" aria-hidden="true">
              </div>
              <div class="podium-base podium-base--first">
                <strong class="podium-place">NO.1</strong>
                <p class="podium-nickname" :title="firstPlace.nickname">{{ firstPlace.nickname }}</p>
                <p class="podium-score">
                  <svg class="gift-icon" aria-hidden="true"><use href="#ranking-gift" /></svg>
                  <span>{{ fmtScore(firstPlace.score) }}</span>
                </p>
              </div>
            </article>

            <article v-if="thirdPlace" class="podium-player podium-player--third">
              <div class="podium-profile">
                <img class="podium-avatar" src="/image/leaderboard/profile/default-avatar.png" :alt="`${thirdPlace.nickname}头像`">
                <img class="podium-crown podium-crown--bronze" src="/image/leaderboard/profile/crown-bronze.png" alt="" aria-hidden="true">
              </div>
              <div class="podium-base podium-base--third">
                <strong class="podium-place">NO.3</strong>
                <p class="podium-nickname" :title="thirdPlace.nickname">{{ thirdPlace.nickname }}</p>
                <p class="podium-score">
                  <svg class="gift-icon" aria-hidden="true"><use href="#ranking-gift" /></svg>
                  <span>{{ fmtScore(thirdPlace.score) }}</span>
                </p>
              </div>
            </article>
          </div>

          <ol class="rank-list" start="4" aria-label="第4至20名">
            <li v-for="item in restList" :key="item.user_id" class="rank-item">
              <span class="rank-number" :aria-label="`第${item.rank}名`">
                <img src="/image/leaderboard/profile/rank-base.png" alt="" aria-hidden="true">
                <span>{{ item.rank }}</span>
              </span>
              <img class="rank-avatar" src="/image/leaderboard/profile/default-avatar.png" :alt="`${item.nickname}头像`">
              <p class="rank-nickname" :title="item.nickname">{{ item.nickname }}</p>
              <p class="rank-score">
                <img class="rank-emblem" src="/image/leaderboard/rank-emblem.png" alt="" aria-hidden="true">
                <svg class="gift-icon" aria-hidden="true"><use href="#ranking-gift" /></svg>
                <span>{{ fmtScore(item.score) }}</span>
              </p>
            </li>
          </ol>
        </template>
      </section>
    </div>

    <footer class="lb-footer" aria-label="我的排名" :inert="namingDialogOpen">
      <template v-if="loading">
        <span class="footer-loading"></span>
      </template>
      <template v-else>
        <span v-if="isCurrentUserRanked" class="footer-rank" :aria-label="currentRankLabel">
          <img src="/image/leaderboard/profile/rank-base.png" alt="" aria-hidden="true">
          <span>{{ myRank?.rank }}</span>
        </span>
        <img
          v-else
          class="footer-unranked"
          src="/image/leaderboard/unranked.png"
          alt="未上榜"
        >
        <img class="footer-avatar" src="/image/leaderboard/profile/default-avatar.png" :alt="`${currentNickname}头像`">
        <p class="footer-nickname" :title="currentNickname">{{ currentNickname }}</p>
        <p class="footer-score">
          <svg class="gift-icon" aria-hidden="true"><use href="#ranking-gift" /></svg>
          <span>{{ fmtScore(myRank?.score ?? 0) }}</span>
        </p>
      </template>
    </footer>

    <div
      v-if="namingDialogOpen"
      class="naming-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="naming-dialog-title"
      @click.self="closeNamingDialog"
      @keydown.esc="closeNamingDialog"
    >
      <section class="naming-dialog-panel popup-center-expand">
        <img
          class="naming-dialog-frame"
          src="/image/leaderboard/naming-guide-dialog.png"
          alt=""
        >
        <h2 id="naming-dialog-title" class="sr-only">定制称号说明</h2>
        <button
          ref="namingDialogCloseButton"
          class="naming-dialog-close"
          type="button"
          aria-label="关闭定制称号说明"
          @click="closeNamingDialog"
        >
          <span aria-hidden="true">×</span>
        </button>

        <div class="naming-dialog-copy" data-testid="naming-dialog-scroll">
          <section>
            <h3>【定制&amp;修改V8称号】说明：</h3>
            <ol>
              <li>定制称号的图标、背景色、文字均可定制，文字最多支持10个字符（称号最多5个字，内容不可包含特殊符号及emoji表情，字体不可更改样式及颜色）。1个中文字、1个符号为2个字符，1个英文字母为1个字符；</li>
              <li>称号文字为系统预设字体，不可更改样式及颜色；</li>
              <li>修改V8称号，仅支持在当前已拥有的V8称号基础上，修改原称号的“图标+文字”或“背景+文字”；</li>
              <li>定制称号的动效统一为：溜光动效加图标左右两个光点闪烁动效，不支持其它动效要求；</li>
              <li>定制/修改特权不能转让或售卖，不能涉及广告、谩骂等不良信息。否则，官方有权拒绝违规冠名信息。</li>
            </ol>
          </section>

          <section>
            <h3>【头像框冠名】说明：</h3>
            <ol>
              <li>冠名可在指定头像框的指定位置展示第一名玩家本人昵称，成为全服独有的专属款式；冠名头像框的样式、冠名文案颜色/字体/位置均按预览（尚未冠名）的版本进行制作。</li>
              <li>昵称需为近30天内至少使用10天以上的常用昵称，最多8个字符。1个中文字、1个符号为2个字符，1个英文字母为1个字符；</li>
              <li>冠名特权不能转让或售卖，不能涉及广告、谩骂等不良信息。否则，官方有权拒绝违规冠名信息；</li>
              <li>冠名特权需在活动结束后的30天内完成冠名，否则自动失效。</li>
            </ol>
          </section>
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import { fetchRanking, type RankingItem, type UserRank } from '@/api/leaderboard'
import {
  activityAudioEnabled,
  initializeActivityAudio,
  toggleActivityAudio,
} from '@/services/activityAudio'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const top20 = ref<RankingItem[]>([])
const myRank = ref<UserRank | null>(null)
const loading = ref(true)
const error = ref('')
const isRefreshing = ref(false)
const namingDialogOpen = ref(false)
const namingGuideButton = ref<HTMLButtonElement | null>(null)
const namingDialogCloseButton = ref<HTMLButtonElement | null>(null)

const firstPlace = computed(() => top20.value[0] ?? null)
const secondPlace = computed(() => top20.value[1] ?? null)
const thirdPlace = computed(() => top20.value[2] ?? null)
const restList = computed(() => top20.value.slice(3))
const currentNickname = computed(() => authStore.nickname || authStore.user?.username || '我')
const isCurrentUserRanked = computed(() => {
  const rank = myRank.value?.rank
  return typeof rank === 'number' && rank >= 1 && rank <= 20
})
const currentRankLabel = computed(() => isCurrentUserRanked.value ? `第${myRank.value!.rank}名` : '未上榜')

function fmtScore(score: number): string {
  if (score < 10_000) return score.toLocaleString('zh-CN')
  return `${(score / 10_000).toFixed(1).replace(/\.0$/, '')}W`
}

function applyRanking(ranking: Awaited<ReturnType<typeof fetchRanking>>) {
  top20.value = (ranking.top_20 ?? []).slice(0, 20)
  myRank.value = ranking.current_user
}

async function openNamingDialog() {
  namingDialogOpen.value = true
  await nextTick()
  namingDialogCloseButton.value?.focus()
}

async function closeNamingDialog() {
  namingDialogOpen.value = false
  await nextTick()
  namingGuideButton.value?.focus()
}

async function handleRefresh() {
  if (isRefreshing.value) return
  isRefreshing.value = true

  try {
    const [, ranking] = await Promise.all([
      authStore.initFromStorage(),
      fetchRanking(),
    ])
    applyRanking(ranking)
    error.value = ''
    showToast('排行榜已刷新')
  } catch (cause) {
    showToast(cause instanceof Error ? cause.message : '排行榜刷新失败，请稍后重试')
  } finally {
    isRefreshing.value = false
  }
}

async function load() {
  loading.value = true
  error.value = ''

  try {
    const profileTask = authStore.user
      ? Promise.resolve(true)
      : authStore.initFromStorage()
    const [, ranking] = await Promise.all([profileTask, fetchRanking()])
    applyRanking(ranking)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '加载排行榜失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  initializeActivityAudio()
  load()
})
</script>

<style scoped>
.leaderboard-page {
  --footer-height: clamp(75px, 20vw, 86px);
  position: relative;
  width: min(100%, 430px);
  height: 100vh;
  height: 100dvh;
  margin: 0 auto;
  overflow: hidden;
  color: #fff;
  background: #21466f;
  box-shadow: 0 0 32px rgba(8, 22, 58, 0.35);
}

.icon-sprite {
  position: absolute;
  width: 0;
  height: 0;
  overflow: hidden;
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

.lb-scroll {
  height: 100%;
  overflow-x: hidden;
  overflow-y: auto;
  padding-bottom: calc(var(--footer-height) + env(safe-area-inset-bottom) + 12px);
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.lb-scroll::-webkit-scrollbar {
  display: none;
}

.reward-scene {
  position: relative;
  width: 100%;
  aspect-ratio: 750 / 1857;
  overflow: hidden;
  background: url('/image/background2.png') center top / 100% 100% no-repeat;
}

.reward-scene::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 6%;
  content: '';
  pointer-events: none;
  background: linear-gradient(180deg, rgba(33, 70, 111, 0), #21466f 92%);
}

.lb-nav {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  z-index: 5;
  height: calc(44px + env(safe-area-inset-top));
  padding-top: env(safe-area-inset-top);
}

.lb-nav__backdrop {
  position: absolute;
  inset: env(safe-area-inset-top) 0 auto;
  width: 100%;
  height: 44px;
  pointer-events: none;
}

.lb-nav__button {
  position: relative;
  display: grid;
  width: 44px;
  height: 44px;
  padding: 4px;
  place-items: center;
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  transition: opacity 180ms ease, transform 180ms ease;
  -webkit-tap-highlight-color: transparent;
}

.lb-nav__button img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.lb-nav__button--back {
  position: absolute;
  top: env(safe-area-inset-top);
  left: 12px;
}

.lb-nav__actions {
  position: absolute;
  top: env(safe-area-inset-top);
  right: 10px;
  display: flex;
  gap: 4px;
}

.lb-nav__button:active {
  transform: scale(0.92);
}

.lb-nav__button:disabled {
  cursor: wait;
  opacity: 0.65;
}

.lb-nav__button--loading img {
  animation: spin 700ms linear infinite;
}

.lb-nav__button:focus-visible,
.naming-guide-button:focus-visible,
.naming-dialog-close:focus-visible,
.retry-button:focus-visible {
  outline: 2px solid #fff4c6;
  outline-offset: 2px;
}

.reward-panel,
.reward-content {
  position: absolute;
  z-index: 1;
  display: block;
  height: auto;
  pointer-events: none;
}

.naming-guide-button {
  position: absolute;
  top: 15.3%;
  right: 0.5%;
  z-index: 3;
  width: 18.8%;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.naming-guide {
  display: block;
  width: 100%;
  height: auto;
  pointer-events: none;
}

.reward-panel {
  top: 22.7%;
  left: 4%;
  width: 92%;
}

.reward-content {
  top: 31.5%;
  left: 50%;
  z-index: 2;
  width: 84%;
  transform: translateX(-50%);
}

.reward-assets {
  position: absolute;
  top: 31.5%;
  left: 50%;
  z-index: 3;
  width: 84%;
  aspect-ratio: 666 / 1247;
  pointer-events: none;
  transform: translateX(-50%);
}

.reward-asset {
  position: absolute;
  display: block;
  height: auto;
  object-fit: contain;
  filter: drop-shadow(0 3px 7px rgba(26, 20, 75, 0.38));
}

.reward-asset img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.reward-asset--first-naming { top: 14%; left: 58%; width: 24%; aspect-ratio: 1; transform: translate(5px, 10px); }
.reward-asset--second-title { top: 52.05%; left: 14.4%; width: 26%; }
.reward-asset--second-frame { top: 47.56%; left: 59.4%; width: 24%; }
.reward-asset--fourth-title { top: 84.17%; left: 4.2%; width: 26%; }
.reward-asset--fourth-frame { top: 79.68%; left: 35.07%; width: 24%; }
.reward-asset--eleventh-title { top: 84.17%; left: 67.26%; width: 26%; }

.ranking-section {
  position: relative;
  margin-top: -1px;
  padding-bottom: 20px;
  overflow: hidden;
  background:
    linear-gradient(116deg, transparent 0 42%, rgba(255, 255, 255, 0.035) 42% 53%, transparent 53%),
    linear-gradient(180deg, #21466f 0%, #1d416a 100%);
}

.ranking-section::before,
.ranking-section::after {
  display: none;
}

.podium {
  position: relative;
  z-index: 1;
  width: 100%;
  aspect-ratio: 750 / 408;
  overflow: hidden;
  background: url('/image/leaderboard/podium-bg.png') center / 100% 100% no-repeat;
}

.podium-player {
  position: absolute;
  top: 0;
  height: 100%;
}

.podium-player--second {
  left: 6%;
  width: 28.7%;
}

.podium-player--first {
  left: 30.4%;
  z-index: 2;
  width: 39.2%;
}

.podium-player--third {
  right: 6.3%;
  width: 28.7%;
}

.podium-crown {
  position: absolute;
  z-index: 2;
  top: -22%;
  left: -8%;
  display: block;
  width: 58%;
  height: auto;
  object-fit: contain;
  pointer-events: none;
  filter: drop-shadow(0 2px 3px rgba(28, 20, 61, 0.35));
}

.podium-profile {
  position: absolute;
  top: 19%;
  left: 50%;
  z-index: 3;
  width: 62%;
  aspect-ratio: 1;
  transform: translateX(-50%);
}

.podium-player--first .podium-profile {
  top: 7%;
  width: 58%;
}

.podium-player--first .podium-crown {
  top: -17%;
}

.podium-player--third .podium-crown {
  left: -8%;
  right: auto;
}

.podium-avatar {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
  filter: drop-shadow(0 4px 7px rgba(19, 25, 78, 0.42));
}

.podium-base {
  position: absolute;
  right: 0;
  bottom: 5%;
  left: 0;
  display: flex;
  width: 100%;
  height: 22%;
  min-width: 0;
  padding: 0 8px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  background: transparent;
  filter: none;
}

.podium-player--first .podium-base {
  height: 28%;
}

.podium-base--first {
  color: #ce6700;
}

.podium-base--second {
  color: #2d6cff;
}

.podium-base--third {
  color: #ff2b91;
}

.podium-place {
  display: none;
}

.podium-nickname {
  width: 100%;
  margin: 0;
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow: 0 1px 0 rgba(255, 255, 255, 0.45);
}

.podium-base--first .podium-nickname {
  font-size: 14px;
}

.podium-score,
.rank-score,
.footer-score {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.podium-score {
  margin-top: 3px;
  font-size: 13px;
  font-weight: 800;
}

.podium-base--first .podium-score {
  font-size: 15px;
}

.gift-icon {
  width: 19px;
  height: 19px;
  flex: 0 0 auto;
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.rank-list {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 22px 24px;
  list-style: none;
}

.rank-item {
  position: relative;
  display: grid;
  grid-template-columns: 40px 46px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  width: 100%;
  min-height: 0;
  aspect-ratio: 658 / 130;
  padding: 8px 18px 8px 9px;
  overflow: hidden;
  background: url('/image/leaderboard/rank-row-bg.png') center / 100% 100% no-repeat;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.rank-item::before,
.rank-item::after {
  display: none;
}

.rank-number,
.rank-nickname,
.rank-score {
  position: relative;
  z-index: 1;
}

.rank-number {
  position: relative;
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  color: #ffd0a7;
  font-size: 16px;
  font-weight: 800;
  text-shadow: 0 1px 2px rgba(43, 26, 83, 0.6);
}

.rank-number img,
.footer-rank img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.rank-number span,
.footer-rank span {
  position: relative;
  z-index: 1;
}

.rank-avatar {
  display: block;
  width: 46px;
  height: 46px;
  object-fit: cover;
  border-radius: 50%;
  filter: drop-shadow(0 2px 4px rgba(25, 25, 84, 0.38));
}

.rank-nickname {
  min-width: 0;
  overflow: hidden;
  color: #f7f8ff;
  font-family: 'Songti SC', 'STSong', 'SimSun', serif;
  font-size: 17px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(23, 32, 109, 0.45);
}

.rank-score {
  color: #ffe45c;
  font-size: 16px;
  font-weight: 800;
  text-shadow: 0 1px 2px rgba(62, 44, 109, 0.55);
}

.rank-emblem {
  display: block;
  width: 44px;
  height: 44px;
  margin-right: 1px;
  object-fit: contain;
  filter: drop-shadow(0 2px 3px rgba(40, 35, 110, 0.32));
}

.rank-score .gift-icon {
  color: #fff;
}

.ranking-loading,
.ranking-state {
  position: relative;
  z-index: 1;
  min-height: 620px;
  padding: 24px 22px;
}

.podium-skeleton,
.rank-skeleton {
  overflow: hidden;
  background: rgba(255, 255, 255, 0.09);
}

.podium-skeleton {
  width: 100%;
  aspect-ratio: 750 / 408;
  margin-bottom: 12px;
}

.rank-skeleton {
  width: 100%;
  height: auto;
  aspect-ratio: 658 / 130;
  margin-bottom: 12px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 8px;
}

.podium-skeleton::after,
.rank-skeleton::after {
  display: block;
  width: 44%;
  height: 100%;
  content: '';
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.12), transparent);
  animation: loading-shimmer 1.4s ease-in-out infinite;
}

@keyframes loading-shimmer {
  from { transform: translateX(-110%); }
  to { transform: translateX(340%); }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.ranking-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  padding-top: 90px;
  color: rgba(255, 255, 255, 0.76);
  text-align: center;
}

.ranking-state p {
  font-size: 16px;
}

.ranking-state span {
  color: rgba(255, 255, 255, 0.48);
  font-size: 13px;
}

.retry-button {
  min-width: 96px;
  height: 36px;
  margin-top: 4px;
  color: #fff6ca;
  font-size: 14px;
  background: rgba(106, 102, 215, 0.75);
  border: 1px solid rgba(255, 255, 255, 0.6);
  border-radius: 6px;
  cursor: pointer;
}

.lb-footer {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 20;
  display: grid;
  grid-template-columns: auto 44px minmax(0, 1fr) auto;
  gap: 9px;
  align-items: center;
  min-height: calc(var(--footer-height) + env(safe-area-inset-bottom));
  padding: 12px 18px calc(12px + env(safe-area-inset-bottom));
  overflow: hidden;
  background: #263f77 url('/image/leaderboard/current-user-bg.png') center top / 100% var(--footer-height) no-repeat;
  border-top: 0;
  box-shadow: 0 -6px 18px rgba(9, 25, 67, 0.35);
}

.lb-footer::before {
  display: none;
}

.footer-rank,
.footer-unranked,
.footer-nickname,
.footer-score,
.footer-loading {
  position: relative;
  z-index: 1;
}

.footer-rank {
  position: relative;
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  color: #fff0b1;
  font-family: 'Songti SC', 'STSong', 'SimSun', serif;
  font-size: 14px;
  font-weight: 800;
  white-space: nowrap;
}

.footer-unranked {
  display: block;
  width: 72px;
  height: auto;
  object-fit: contain;
}

.footer-avatar {
  display: block;
  width: 44px;
  height: 44px;
  object-fit: cover;
  border-radius: 50%;
  filter: drop-shadow(0 2px 4px rgba(22, 19, 64, 0.42));
}

.footer-nickname {
  min-width: 0;
  overflow: hidden;
  color: #f8f8ff;
  font-family: 'Songti SC', 'STSong', 'SimSun', serif;
  font-size: 17px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-shadow: 0 1px 2px rgba(18, 25, 72, 0.6);
}

.footer-score {
  color: #fff05a;
  font-size: 16px;
  font-weight: 800;
  text-shadow: 0 1px 2px rgba(45, 26, 73, 0.65);
}

.footer-score .gift-icon {
  color: #fff;
}

.footer-loading {
  grid-column: 1 / -1;
  width: 100%;
  height: 34px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 6px;
}

.naming-dialog {
  position: absolute;
  inset: 0;
  z-index: 50;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 10px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.naming-dialog-panel {
  position: relative;
  width: min(96%, 400px);
  aspect-ratio: 654 / 716;
  max-height: 100%;
}

.naming-dialog-frame {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.naming-dialog-close {
  position: absolute;
  top: 5.5%;
  right: 4.5%;
  z-index: 2;
  display: grid;
  width: 30px;
  height: 30px;
  padding: 0;
  place-items: center;
  color: #f8f4ff;
  font-family: Arial, sans-serif;
  font-size: 27px;
  line-height: 1;
  background: rgba(69, 69, 165, 0.62);
  border: 1px solid rgba(255, 255, 255, 0.72);
  border-radius: 50%;
  box-shadow: 0 0 8px rgba(237, 218, 255, 0.45);
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}

.naming-dialog-close span {
  transform: translateY(-1px);
}

.naming-dialog-copy {
  position: absolute;
  top: 18.5%;
  right: 8.5%;
  bottom: 8.5%;
  left: 10.5%;
  z-index: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 2px 7px 14px 1px;
  color: #fff;
  font-family: 'Songti SC', 'STSong', 'SimSun', serif;
  font-size: 14px;
  line-height: 1.65;
  text-shadow: 0 1px 2px rgba(39, 39, 103, 0.55);
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.naming-dialog-copy::-webkit-scrollbar {
  display: none;
}

.naming-dialog-copy section + section {
  margin-top: 20px;
}

.naming-dialog-copy h3 {
  margin-bottom: 10px;
  color: #fff;
  font-size: 15px;
  font-weight: 700;
}

.naming-dialog-copy ol {
  padding-left: 0;
  list-style: none;
  counter-reset: naming-rule;
}

.naming-dialog-copy li {
  position: relative;
  padding-left: 1.8em;
  counter-increment: naming-rule;
}

.naming-dialog-copy li::before {
  position: absolute;
  top: 0;
  left: 0;
  content: counter(naming-rule) '、';
}

.naming-dialog-copy li + li {
  margin-top: 7px;
}

@media (max-width: 360px) {
  .rank-list {
    padding-right: 16px;
    padding-left: 16px;
  }

  .rank-item {
    grid-template-columns: 36px 42px minmax(0, 1fr) auto;
    gap: 8px;
    padding-right: 12px;
  }

  .rank-number {
    width: 34px;
    height: 34px;
    font-size: 14px;
  }

  .rank-avatar {
    width: 42px;
    height: 42px;
  }

  .rank-nickname,
  .footer-nickname {
    font-size: 15px;
  }

  .rank-score,
  .footer-score {
    font-size: 14px;
  }

  .rank-emblem {
    width: 38px;
    height: 38px;
  }

  .lb-footer {
    gap: 9px;
    padding-right: 16px;
    padding-left: 16px;
  }

  .naming-dialog-copy {
    font-size: 13px;
    line-height: 1.6;
  }

  .naming-dialog-copy h3 {
    font-size: 14px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .lb-nav__button--loading img,
  .podium-skeleton::after,
  .rank-skeleton::after {
    animation: none;
  }
}
</style>
