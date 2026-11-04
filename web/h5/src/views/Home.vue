<template>
  <main class="activity-page" :inert="activityPageBlocking ? true : undefined">
    <section class="activity-scene" :class="`activity-scene--${activeTab}`">
      <FallingPetals />
      <div ref="activityScrollRef" class="activity-scroll" :style="bgStyle">
        <h1 class="sr-only">仙境奇旅·花开几许</h1>

        <header class="scene-nav" aria-label="页面导航">
          <img class="scene-nav__backdrop" src="/image/activity/nav-bar.svg" alt="" aria-hidden="true">
          <button class="scene-nav__button scene-nav__button--back" type="button" aria-label="退出活动" @click="handleLogout">
            <img src="/image/activity/nav-back.svg" alt="">
          </button>
          <div class="scene-nav__actions">
            <button
              class="scene-nav__button"
              type="button"
              :aria-label="activityAudioEnabled ? '关闭背景音乐' : '开启背景音乐'"
              @click="toggleActivityAudio"
            >
              <img :src="activityAudioEnabled ? '/image/activity/nav-music.png' : '/image/activity/nav-music-off.png'" alt="">
            </button>
            <button
              class="scene-nav__button"
              :class="{ 'scene-nav__button--loading': isRefreshing }"
              type="button"
              aria-label="刷新活动数据"
              :disabled="isRefreshing"
              @click="handleRefresh"
            >
              <img src="/image/activity/nav-refresh.svg" alt="">
            </button>
            <button
              v-if="authStore.isAdmin"
              class="scene-nav__button scene-nav__button--admin"
              type="button"
              aria-label="管理后台"
              @click="goToAdmin"
            >
              <span class="scene-nav__admin-label">管理</span>
            </button>
          </div>
        </header>

        <div class="hero-space" aria-hidden="true"></div>

        <div class="marquee-slot">
          <section
            v-if="currentMarquee"
            class="marquee"
            :class="`marquee--${activeTab}`"
            aria-label="中奖播报"
            aria-live="polite"
          >
            <div
              :key="currentMarquee.id"
              class="marquee__track"
              @animationend="handleMarqueeAnimationEnd"
            >
              <span class="marquee__item">
                <span class="marquee__spark" aria-hidden="true"></span>
                <span class="marquee__text">恭喜 {{ currentMarquee.nickname }} 获得 {{ currentMarquee.item_name }}</span>
              </span>
            </div>
          </section>
          <p
            v-if="marqueeSSEDegraded"
            class="marquee__degraded"
            role="alert"
          >
            实时播报连接中断，
            <button type="button" class="marquee__reconnect-btn" @click="connectMarqueeSSE()">点击重连</button>
          </p>
        </div>

        <div class="wish-tabs" role="tablist" aria-label="选择许愿场景">
          <img
            class="wish-tabs__art"
            :src="wishTabAssets[activeTab]"
            alt=""
            aria-hidden="true"
            draggable="false"
          >
          <button
            class="wish-tab wish-tab--day"
            :class="{ 'wish-tab--active': activeTab === 'day' }"
            type="button"
            role="tab"
            :aria-selected="activeTab === 'day'"
            @click="activeTab = 'day'"
          >
            <span class="sr-only">白昼许愿</span>
          </button>
          <button
            class="wish-tab wish-tab--night"
            :class="{ 'wish-tab--active': activeTab === 'night' }"
            type="button"
            role="tab"
            :aria-selected="activeTab === 'night'"
            @click="activeTab = 'night'"
          >
            <span class="sr-only">星夜许愿</span>
          </button>
        </div>

        <section class="guide-section" aria-labelledby="guide-title">
          <h2 id="guide-title" class="sr-only">玩法攻略</h2>
          <img
            class="guide-section__art"
            :src="guideAssets[activeTab]"
            alt=""
            aria-hidden="true"
            draggable="false"
          >
          <p class="sr-only">
            {{ activeTab === 'day'
              ? '每次许愿获得真爱无敌戒指、烟花之恋戒指等奖励之一；每次许愿概率点亮1朵花，每点亮花盒6朵花可开启戒指宝箱。'
              : '每次许愿获得真爱无敌戒指、爱意翩跹戒指、闪闪心蝶戒指等奖励之一；每次许愿概率点亮多朵花，每点亮花盒6朵花可开启戒指宝箱。' }}
          </p>

          <button
            class="scene-entry guide-section__all-rewards"
            type="button"
            aria-label="查看全部奖励"
            @click="showAllRewards = true"
          >
            <img :src="allRewardsAssets[activeTab]" alt="查看全部奖励">
          </button>

          <button class="scene-entry scene-entry--rules" type="button" aria-label="活动说明" @click="showActivityRules = true">
            <img src="/image/activity/rules.png" alt="活动说明">
          </button>
          <button class="scene-entry scene-entry--rank" type="button" aria-label="排行榜" @click="router.push('/leaderboard')">
            <img src="/image/activity/leaderboard.png" alt="排行榜">
          </button>
          <button class="scene-entry scene-entry--benefit" type="button" aria-label="新戒福利" @click="showNewRingBenefit = true">
            <img src="/image/activity/new-ring-benefit.png" alt="新戒福利">
          </button>
        </section>

        <section class="showcase" aria-label="戒指奖励展示">
          <button v-if="petalGiftAvailable && activityActive" class="scene-entry showcase__gift" type="button" aria-label="打开许愿花瓣礼包" @click="openPetalGift">
            <img src="/image/activity/petal-gift.png" alt="五折许愿花瓣礼包">
          </button>

          <FlowerBox
            ref="flowerBoxRef"
            class="showcase__box"
            :lit-count="litFlowersInCurrentBox"
            :mode="activeTab"
          />

          <button
            class="scene-entry showcase__ring-record"
            type="button"
            aria-label="戒指召唤记录"
            @click="showRingRecords = true"
          >
            <img src="/image/activity/ring-record.png" alt="戒指召唤记录">
          </button>

          <article
            v-for="ring in ringItems"
            :key="ring.slot"
            class="ring-card"
            :class="`ring-card--${ring.slot}`"
            :aria-label="ring.name"
          >
            <img class="ring-card__orb" :src="ringOrbAsset" alt="">
            <picture class="ring-card__reward">
              <source :srcset="ring.animatedImage" type="image/svg+xml">
              <img :src="ring.image" :alt="ring.name">
            </picture>
            <strong class="ring-card__name">{{ ring.name }}</strong>
            <img v-if="ring.isNew" class="ring-card__new" src="/image/activity/ring-orbs/new.png" alt="新奖励">
            <img
              v-if="isRingObtained(ring.itemIds)"
              class="ring-card__obtained"
              src="/image/activity/ring-orbs/obtained.png"
              alt="已获得"
            >
          </article>

          <div class="petal-balance" aria-label="当前花瓣余额">
            <img class="petal-balance__icon" src="/image/activity/petal.png" alt="">
            <strong>{{ petals.toLocaleString() }}</strong>
            <button type="button" aria-label="购买花瓣" :disabled="!activityActive" @click="openPetalShop">
              <img src="/image/activity/plus.svg" alt="">
            </button>
          </div>

          <button
            class="scene-entry showcase__my-rewards"
            type="button"
            aria-label="我的奖励"
            @click="showMyRewards = true"
          >
            <img src="/image/activity/my-rewards/button-bg.png" alt="" aria-hidden="true">
          </button>
        </section>

        <section class="progress-panel" aria-labelledby="progress-title">
          <img
            class="progress-panel__art"
            :src="activeStagePanelAsset"
            alt=""
            aria-hidden="true"
            draggable="false"
          >

          <div class="progress-panel__header">
            <h2 id="progress-title" class="sr-only">累计点亮花朵领戒指</h2>
            <div class="progress-panel__count">
              <img
                :src="activeTab === 'night'
                  ? '/image/activity/flower-count-bg-night.png'
                  : '/image/activity/flower-count-bg-day.png'"
                alt=""
                aria-hidden="true"
              >
              <span>当前已点亮：<strong>{{ currentFlowers }}</strong>朵</span>
            </div>
          </div>

          <div v-if="stageLoadState === 'loading'" class="progress-panel__state" role="status">奖励加载中…</div>
          <div v-else-if="stageLoadState === 'error'" class="progress-panel__state" role="alert">
            <span>奖励加载失败</span>
            <button type="button" @click="loadStageRewards">重新加载</button>
          </div>
          <div v-else-if="progressNodes.length === 0" class="progress-panel__state">奖励即将揭晓</div>
          <ol v-else class="reward-list">
            <li v-for="node in progressNodes" :key="node.id" class="reward-item" :class="`reward-item--${node.status}`">
              <button
                type="button"
                :aria-label="rewardAriaLabel(node)"
                :disabled="node.status !== 'available' || !activityActive"
                @click="handleClaimStageReward(node.id - 1)"
              >
                <span class="reward-item__visual">
                  <picture
                    v-if="getStageRewardAnimatedAsset(node.item_id)"
                    class="reward-item__picture"
                  >
                    <source :srcset="getStageRewardAnimatedAsset(node.item_id)" type="image/svg+xml">
                    <img
                      class="reward-item__image"
                      :src="getStageRewardAsset(node.item_id)"
                      :alt="node.name"
                      draggable="false"
                    >
                  </picture>
                  <img
                    v-else-if="getStageRewardAsset(node.item_id)"
                    class="reward-item__image"
                    :src="getStageRewardAsset(node.item_id)"
                    :alt="node.name"
                    draggable="false"
                  >
                  <RewardIcon v-else class="reward-item__icon" :kind="node.icon" />
                  <img
                    v-if="node.status === 'claimed'"
                    class="reward-item__claimed"
                    src="/image/activity/stage/claimed.png"
                    alt="已领取"
                    draggable="false"
                  >
                </span>
                <span class="reward-item__name">{{ node.name }}</span>
                <img
                  v-if="getStageThresholdAsset(node.lit_threshold)"
                  class="reward-item__threshold"
                  :src="getStageThresholdAsset(node.lit_threshold)"
                  :alt="`点亮${node.lit_threshold}朵花`"
                >
                <span v-else class="reward-item__threshold-fallback">{{ node.lit_threshold }}朵</span>
              </button>
            </li>
          </ol>

        </section>
      </div>

      <section
        class="draw-dock"
        :class="[
          `draw-dock--${activeTab}`,
          {
            'draw-dock--tbp': showTryBeforePayButton,
            'draw-dock--round-complete': roundComplete,
          },
        ]"
        :style="drawDockStyle"
        aria-labelledby="draw-title"
      >
        <h2 id="draw-title" class="sr-only">选择许愿次数</h2>
        <span class="sr-only">{{ poolNames[activePoolType] || '奖池加载中' }}</span>

        <button class="draw-dock__record" type="button" aria-label="抽奖记录" @click="showRecords = true">
          <img src="/image/activity/draw-record.png" alt="抽奖记录">
        </button>

        <div class="draw-dock__buttons" :class="{ 'draw-dock__buttons--compact': !showTryBeforePayButton }">
          <button
            v-for="count in drawCounts"
            :key="count"
            :ref="(element) => setDrawButtonRef(count, element)"
            class="draw-dock__button"
            type="button"
            :aria-label="getDrawButtonAriaLabel(count)"
            :disabled="!activityActive || isDrawing || currentCostPerDraw === 0 || boxFlowBlocking || roundComplete"
            @click="handleDraw(count)"
          >
            <img class="draw-dock__button-image" :src="getDrawButtonImage(count)" alt="" aria-hidden="true" draggable="false">
            <span class="sr-only">{{ isDrawing ? `许愿${count}次，许愿中` : `许愿${count}次` }}</span>
            <small class="sr-only" aria-hidden="true">{{ (currentCostPerDraw * count).toLocaleString() }}</small>
          </button>
          <button
            v-if="showTryBeforePayButton"
            class="draw-dock__button draw-dock__button--tbp"
            type="button"
            :aria-label="getTryBeforePayAriaLabel()"
            :disabled="!activityActive || isDrawing || isTryBeforePayLoading || isTryBeforePayConfirming || boxFlowBlocking || roundComplete"
            @click="handleTryBeforePay"
          >
            <img class="draw-dock__button-image" src="/image/activity/draw/tbp.png" alt="" aria-hidden="true" draggable="false">
            <span class="sr-only">{{ isTryBeforePayLoading ? '先抽后付，生成中' : '先抽后付' }}</span>
          </button>
        </div>

        <div class="draw-dock__settings" aria-label="许愿设置">
          <label>
            <input v-model="showConfirm" type="checkbox" :disabled="roundComplete || !activityActive">
            <span aria-hidden="true"></span>显示购买确认
          </label>
          <label>
            <input v-model="skipAnim" type="checkbox" :disabled="roundComplete || !activityActive">
            <span aria-hidden="true"></span>跳过动画
          </label>
        </div>

        <div
          v-if="roundComplete"
          class="draw-dock__new-round"
        >
          <img src="/image/activity/draw/start-new-round.png" alt="" aria-hidden="true" draggable="false">
          <button
            class="draw-dock__new-round-action"
            type="button"
            :disabled="isStartingNewRound || !activityActive"
            :aria-label="isStartingNewRound ? '正在开始新一轮' : '开始新一轮'"
            @click="handleStartNewRound"
          >
            <span class="sr-only">开始新一轮</span>
          </button>
        </div>

        <div v-if="activityUnavailable" class="draw-dock__ended" role="status">
          {{ activityStatusText }}
        </div>
      </section>
      <PagAnimation
        v-if="drawAnimation"
        :key="drawAnimation.key"
        class="draw-flight-animation"
        :src="drawAnimation.src"
        :style="{ transform: drawAnimation.transform }"
        @complete="finishDrawAnimation"
        @error="finishDrawAnimation"
      />
    </section>

    <ActivityRulesPopup v-if="showActivityRules" @close="showActivityRules = false" />
    <PetalShopPopup v-if="showPetalShop" @close="handlePetalShopClose" @balance-change="fetchAssets" />
    <PetalGiftOfferPopup
      v-if="showPetalGift"
      :show-result="showPetalGiftResult"
      :purchasing="isPurchasingPetalGift"
      @close="showPetalGift = false"
      @purchase="handlePurchasePetalGift"
      @confirm="closePetalGiftResult"
    />
    <AllRewardsPopup
      v-if="showAllRewards"
      :pools="pools"
      :initial-pool-type="activePoolType"
      @close="showAllRewards = false"
    />
    <MyRewardsPopup v-if="showMyRewards" @close="showMyRewards = false" />
    <BuyRemainingPopup
      v-if="showBuyRemaining"
      :pool-name="poolNames[activePoolType] ?? ''"
      :count="pendingDrawCount"
      :cost-per-draw="currentCostPerDraw"
      :shortage="currentShortage"
      :petals="petals"
      :gold="gold"
      @confirm="onBuyRemainingConfirm"
      @close="showBuyRemaining = false"
      @go-shop="onBuyRemainingGoShop"
    />
    <DrawResultPopup
      v-if="drawResult"
      :result="drawResult"
      :confirming="isConfirmingDrawResult"
      :pool-type="activePoolType"
      @confirm="onDrawResultConfirm"
      @redraw="onDrawResultRedraw"
    />
    <RoundRefundPopup
      v-if="refundResult"
      :result="refundResult"
      @confirm="onRefundConfirm"
    />
    <FlowerLitResultPopup
      v-if="flowerLitNotice"
      :lit-flowers="flowerLitNotice.litFlowers"
      :lit-count-after="flowerLitNotice.litCountAfter"
      @confirm="onFlowerLitConfirm"
    />
    <TryBeforePayPopup
      v-if="showTryBeforePay && tryBeforePayPreview"
      :preview="tryBeforePayPreview"
      :confirming="isTryBeforePayConfirming"
      @confirm="onTryBeforePayConfirm"
      @close="requestTryBeforePayAbandon"
    />
    <TryBeforePayAbandonPopup
      v-if="showTryBeforePayAbandonConfirm"
      :loading="isTryBeforePayLoading"
      @confirm="confirmTryBeforePayAbandon"
      @keep="showTryBeforePayAbandonConfirm = false"
    />
    <TryBeforePayOfferPopup
      v-if="showTryBeforePayOffer"
      :loading="isTryBeforePayLoading"
      @close="dismissTryBeforePayOffer"
      @use="useTryBeforePayOffer"
    />
    <RecordsPopup
      v-if="showRecords"
      :initial-pool-type="activePoolType"
      @close="showRecords = false"
    />
    <RingRecordsPopup v-if="showRingRecords" @close="showRingRecords = false" />
    <NewRingBenefitPopup v-if="showNewRingBenefit" @close="showNewRingBenefit = false" />
    <StageRewardResultPopup
      v-if="stageRewardResult"
      :reward="stageRewardResult"
      @confirm="stageRewardResult = null"
    />
    <TrueLoveAcquiredPopup
      v-if="trueLoveAcquired"
      :item-id="trueLoveAcquired.itemId"
      :item-name="trueLoveAcquired.itemName"
      :nickname="authStore.nickname"
      @confirm="handleTrueLoveAcquiredConfirm"
    />
    <RingSummonFlow
      :status="activeSummonStatus"
      :load-state="activeSummonLoadState"
      :opening="isOpeningBox"
      :confirming="isConfirmingBox || isConfirmingStageChoice || isConfirmingDrawChoice"
      @retry="loadBoxStatus(true)"
      @summon="handleOpenBox"
      @confirm="handleConfirmSummon"
    />
  </main>

  <Teleport to="body">
    <div v-if="newRoundNotice" class="new-round-notice" role="status" aria-live="polite">
      {{ newRoundNotice }}
    </div>
    <div v-if="goldInsufficientNotice" class="gold-insufficient-notice" role="status" aria-live="polite">
      {{ goldInsufficientNotice }}
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from 'vant'
import ActivityRulesPopup from '@/components/ActivityRulesPopup.vue'
import AllRewardsPopup from '@/components/AllRewardsPopup.vue'
import BuyRemainingPopup from '@/components/BuyRemainingPopup.vue'
import DrawResultPopup from '@/components/DrawResultPopup.vue'
import FallingPetals from '@/components/FallingPetals.vue'
import FlowerBox from '@/components/FlowerBox.vue'
import FlowerLitResultPopup from '@/components/FlowerLitResultPopup.vue'
import MyRewardsPopup from '@/components/MyRewardsPopup.vue'
import NewRingBenefitPopup from '@/components/NewRingBenefitPopup.vue'
import PagAnimation from '@/components/PagAnimation.vue'
import PetalShopPopup from '@/components/PetalShopPopup.vue'
import PetalGiftOfferPopup from '@/components/PetalGiftOfferPopup.vue'
import RecordsPopup from '@/components/RecordsPopup.vue'
import RewardIcon from '@/components/RewardIcon.vue'
import RoundRefundPopup from '@/components/RoundRefundPopup.vue'
import RingRecordsPopup from '@/components/RingRecordsPopup.vue'
import RingSummonFlow from '@/components/RingSummonFlow.vue'
import StageRewardResultPopup from '@/components/StageRewardResultPopup.vue'
import TrueLoveAcquiredPopup from '@/components/TrueLoveAcquiredPopup.vue'
import TryBeforePayPopup from '@/components/TryBeforePayPopup.vue'
import TryBeforePayAbandonPopup from '@/components/TryBeforePayAbandonPopup.vue'
import TryBeforePayOfferPopup from '@/components/TryBeforePayOfferPopup.vue'
import { exchangePetals, getAssets, getPetalGiftStatus, purchasePetalGift } from '@/api/asset'
import {
  fetchActivityConfig,
  type ActivityConfig,
  type RingMarqueeItem,
} from '@/api/activity'
import { startNewRound } from '@/api/flower'
import { clearRecordCaches } from '@/api/records'
import {
  confirmDrawChoice,
  draw,
  fetchPendingDrawChoices,
  fetchPools,
  type DrawPendingChoice,
  type DrawResponse,
  type PoolConfig,
} from '@/api/lottery'
import {
  claimStageReward,
  confirmStageRewardChoice,
  fetchStageRewards,
  type ClaimStageRewardResult,
  type StageRewardChoice,
  type StageRewardItem,
} from '@/api/stage'
import {
  confirmBox,
  fetchBoxStatus,
  openBox,
  type BoxStatusResponse,
} from '@/api/treasureBox'
import {
  abandonTryBeforePay,
  confirmTryBeforePay,
  fetchTryBeforePayStatus,
  previewTryBeforePay,
  type TryBeforePayPreview,
} from '@/api/tbp'
import { createAuthSSE } from '@/api/sse-client'
import { getToken } from '@/api/request'
import { useAuthStore } from '@/stores/auth'
import {
  activityAudioEnabled,
  initializeActivityAudio,
  pauseActivityAudio,
  playActivityEffect,
  toggleActivityAudio,
} from '@/services/activityAudio'
import { getRingCatalogItem } from '@/constants/ringCatalog'
import type { ProgressNode, ProgressStatus, RewardIconKind, WishMode } from '@/types/activity'
import {
  createDrawAnimationTransform,
  DRAW_ANIMATION_CANVAS_SIZE,
  toCssMatrix,
} from '@/utils/drawAnimationTransform'

const router = useRouter()
const authStore = useAuthStore()
const drawCounts = [1, 10, 30] as const
const MARQUEE_MAX_QUEUE = 100
const MARQUEE_PLAYED_STORAGE_KEY = 'lottery_marquee_played'
type DrawCount = (typeof drawCounts)[number]
type FlowerLitNotice = {
  litFlowers: number
  litCountAfter: number
  drawResult: DrawResponse | null
}
type FlowerBoxExpose = {
  getDrawAnimationTargetRect: () => DOMRect | null
}

const drawButtonAssets: Record<WishMode, Record<DrawCount, string>> = {
  day: {
    1: '/image/activity/draw/day-draw-1.png',
    10: '/image/activity/draw/day-draw-10.png',
    30: '/image/activity/draw/day-draw-30.png',
  },
  night: {
    1: '/image/activity/draw/night-draw-1.png',
    10: '/image/activity/draw/night-draw-10.png',
    30: '/image/activity/draw/night-draw-30.png',
  },
}

const drawButtonAnimations: Record<DrawCount, string> = {
  1: '/image/activity/draw/animations/wish-left.pag',
  10: '/image/activity/draw/animations/wish-center.pag',
  30: '/image/activity/draw/animations/wish-right.pag',
}

const drawDockBackgrounds: Record<WishMode, string> = {
  day: '/image/activity/draw/day-draw-base.png',
  night: '/image/activity/draw/night-draw-base.png',
}

const wishTabAssets: Record<WishMode, string> = {
  day: '/image/activity/wish-tab-day-active.png',
  night: '/image/activity/wish-tab-night-active.png',
}

const guideAssets: Record<WishMode, string> = {
  day: '/image/activity/guide/guide-day.png',
  night: '/image/activity/guide/guide-night.png',
}

const allRewardsAssets: Record<WishMode, string> = {
  day: '/image/activity/guide/all-rewards-day.png',
  night: '/image/activity/guide/all-rewards-night.png',
}

const ringItems = [
  {
    slot: 'moon',
    name: '月亮游记戒指',
    image: '/image/activity/rings/ring-1205251-moon.png',
    animatedImage: '/image/activity/rings/ring-1205251-moon.svg',
    itemIds: [1205251],
    isNew: false,
  },
  {
    slot: 'true-love',
    name: '真爱无敌戒指',
    image: '/image/activity/rings/ring-1207751-true-love-classic.png',
    animatedImage: '/image/activity/rings/ring-1207751-true-love-classic.svg',
    itemIds: [1207751, 1207752, 1207753],
    isNew: true,
  },
  {
    slot: 'love',
    name: '爱意翩跹戒指',
    image: '/image/activity/rings/ring-1205470-love.png',
    animatedImage: '/image/activity/rings/ring-1205470-love.svg',
    itemIds: [1205470],
    isNew: false,
  },
] as const

const stagePanelAssets: Record<WishMode, string> = {
  day: '/image/activity/stage/stage-panel-day.png',
  night: '/image/activity/stage/stage-panel-night.png',
}

const stageThresholdAssets: Partial<Record<number, string>> = {
  3: '/image/activity/stage/stage-threshold-3.png',
  5: '/image/activity/stage/stage-threshold-5.png',
  8: '/image/activity/stage/stage-threshold-8.png',
  11: '/image/activity/stage/stage-threshold-11.png',
  15: '/image/activity/stage/stage-threshold-15.png',
  17: '/image/activity/stage/stage-threshold-17.png',
}

const activeTab = ref<WishMode>('day')
const activityConfig = ref<ActivityConfig | null>(null)
const marqueeQueue = ref<RingMarqueeItem[]>([])
const currentMarquee = ref<RingMarqueeItem | null>(null)
const marqueeEventSource = ref<{ close: () => void } | null>(null)
const marqueeSSEReconnectCount = ref(0)
const marqueeSSEDegraded = ref(false)
const queuedMarqueeIds = new Set<string>()
let playedMarqueeIds = new Set<string>()
const petals = ref(0)
const gold = ref(0)
const progressNodes = ref<ProgressNode[]>([])
const currentFlowers = ref(0)
const stageLoadState = ref<'loading' | 'ready' | 'error'>('loading')
const poolCosts = ref<Record<number, number>>({})
const poolNames = ref<Record<number, string>>({})
const pools = ref<PoolConfig[]>([])
const isDrawing = ref(false)
const activityScrollRef = ref<HTMLElement | null>(null)
const flowerBoxRef = ref<FlowerBoxExpose | null>(null)
const drawButtonRefs = new Map<DrawCount, HTMLButtonElement>()
const drawResult = ref<DrawResponse | null>(null)
const refundResult = ref<DrawResponse | null>(null)
const isConfirmingDrawResult = ref(false)
const pendingRepeatDrawCount = ref<number | null>(null)
const isRefreshing = ref(false)
const showActivityRules = ref(false)
const showPetalShop = ref(false)
const showPetalGift = ref(false)
const showPetalGiftResult = ref(false)
const isPurchasingPetalGift = ref(false)
const petalGiftAvailable = ref(false)
const showAllRewards = ref(false)
const showMyRewards = ref(false)
const showBuyRemaining = ref(false)
const showTryBeforePay = ref(false)
const showTryBeforePayAbandonConfirm = ref(false)
const showTryBeforePayOffer = ref(false)
const tryBeforePayOfferHandled = ref(false)
const showRecords = ref(false)
const showRingRecords = ref(false)
const showNewRingBenefit = ref(false)
const pendingDrawCount = ref(0)
const currentShortage = ref(0)
const showConfirm = ref(true)
const skipAnim = ref(false)
const tryBeforePayAvailable = ref(false)
const tryBeforePaySessionId = ref<string | null>(null)
const tryBeforePayPreview = ref<TryBeforePayPreview | null>(null)
const isTryBeforePayLoading = ref(false)
const isTryBeforePayConfirming = ref(false)
const isTryBeforePayStatusLoading = ref(true)
const boxStatus = ref<BoxStatusResponse | null>(null)
const boxLoadState = ref<'idle' | 'loading' | 'ready' | 'error'>('idle')
const isOpeningBox = ref(false)
const isConfirmingBox = ref(false)
const isStartingNewRound = ref(false)
const stageRewardResult = ref<ClaimStageRewardResult | null>(null)
const stagePendingChoice = ref<StageRewardChoice | null>(null)
const isConfirmingStageChoice = ref(false)
const drawPendingChoices = ref<DrawPendingChoice[]>([])
const isConfirmingDrawChoice = ref(false)
const drawFlowAfterChoices = ref<DrawResponse | null>(null)
const tbpFlowAfterChoices = ref<{ litFlowers: number; litCountAfter: number } | null>(null)
const flowerLitNotice = ref<FlowerLitNotice | null>(null)
const drawAnimation = ref<{ key: number; src: string; transform: string } | null>(null)
const trueLoveAcquired = ref<{ itemId: number; itemName: string } | null>(null)
let trueLoveAcquiredContinuation: (() => Promise<void> | void) | null = null
let drawAnimationSequence = 0
let resolveDrawAnimation: (() => void) | null = null

const activePoolType = computed(() => activeTab.value === 'day' ? 1 : 2)
const activityActive = computed(() => activityConfig.value?.status === 1)
const activityUnavailable = computed(() => activityConfig.value !== null && !activityActive.value)
const activityStatusText = computed(() => {
  if (!activityConfig.value) return '活动状态加载中'
  return activityConfig.value.status === 2 ? '活动已结束' : '活动尚未开始'
})
const activeSummonStatus = computed<BoxStatusResponse | null>(() => {
  if (activityConfig.value && !activityActive.value) return null
  const drawChoice = drawPendingChoices.value[0]
  if (drawChoice) {
    return {
      round_number: 0,
      box_chances: 1,
      box_opened: 0,
      flow_state: 'choice',
      pending_reward: {
        box_id: drawChoice.reward_record_id,
        chance_index: 1,
        item_id: drawChoice.item_id,
        item_name: drawChoice.item_name,
        is_choice: true,
        choice_ids: drawChoice.choice_ids,
      },
      opened_rewards: [],
    }
  }
  const pending = stagePendingChoice.value
  if (!pending) return boxStatus.value
  return {
    round_number: 0,
    box_chances: 1,
    box_opened: 0,
    flow_state: 'choice',
    pending_reward: {
      box_id: pending.reward_record_id,
      chance_index: pending.slot_index + 1,
      item_id: pending.item_id,
      item_name: pending.item_name,
      is_choice: true,
      choice_ids: pending.choice_ids,
    },
    opened_rewards: [],
  }
})
const activeSummonLoadState = computed(() => (
  activityConfig.value && !activityActive.value
    ? 'idle'
    : drawPendingChoices.value.length > 0 || stagePendingChoice.value
      ? 'ready'
      : boxLoadState.value
))
const ringOrbAsset = computed(() => activeTab.value === 'night'
  ? '/image/activity/ring-orbs/night.png'
  : '/image/activity/ring-orbs/day.png')
const currentCostPerDraw = computed(() => poolCosts.value[activePoolType.value] ?? 0)
const showTryBeforePayButton = computed(() => activityActive.value && activeTab.value === 'day' && tryBeforePayAvailable.value)
const activeStagePanelAsset = computed(() => stagePanelAssets[activeTab.value])
const roundComplete = computed(() => currentFlowers.value >= 18)
const newRoundNotice = ref('')
const goldInsufficientNotice = ref('')
let newRoundNoticeTimer: ReturnType<typeof setTimeout> | null = null
let goldInsufficientNoticeTimer: ReturnType<typeof setTimeout> | null = null
const hasUnclaimedStageReward = computed(() => progressNodes.value.some((node) => (
  node.lit_threshold <= currentFlowers.value && node.status !== 'claimed'
)))
const boxFlowBlocking = computed(() => {
  if (activityConfig.value && !activityActive.value) return false
  if (drawPendingChoices.value.length > 0) return true
  if (stagePendingChoice.value) return true
  if (boxLoadState.value === 'loading' || boxLoadState.value === 'error') return true
  const status = boxStatus.value
  if (!status) return false
  if (status.flow_state === 'ready') return typeof status.next_chance_index === 'number'
  if (status.flow_state === 'result' || status.flow_state === 'choice') return Boolean(status.pending_reward)
  return false
})
const activityPageBlocking = computed(() => (
  isTryBeforePayStatusLoading.value
  || boxFlowBlocking.value
  || drawResult.value !== null
  || refundResult.value !== null
  || flowerLitNotice.value !== null
  || showTryBeforePay.value
  || showTryBeforePayOffer.value
  || showAllRewards.value
  || showMyRewards.value
  || showActivityRules.value
  || showNewRingBenefit.value
  || showPetalGift.value
  || trueLoveAcquired.value !== null
))
const litFlowersInCurrentBox = computed(() => {
  const status = boxStatus.value
  if (roundComplete.value) return 6
  if (!status) return Math.min(6, currentFlowers.value)
  if (status.flow_state !== 'none' || status.box_chances > status.box_opened) return 6

  const completedFlowers = status.box_opened * 6
  return Math.max(0, Math.min(6, currentFlowers.value - completedFlowers))
})
const bgStyle = computed(() => ({
  '--activity-background': `url('/image/background${activeTab.value === 'day' ? '1' : '2'}.png')`,
}))
const drawDockStyle = computed(() => ({
  '--draw-dock-bg': `url('${drawDockBackgrounds[activeTab.value]}')`,
}))

function getRewardIconKind(item: StageRewardItem): RewardIconKind {
  if (item.item_id === 1001 || item.item_name.includes('金币')) return 'coin'
  if (item.item_id === 1002 || item.item_name.includes('花瓣')) return 'petal'
  if (item.item_name.includes('戒指')) return 'ring'
  return 'gift'
}

function isRingObtained(itemIds: readonly number[]) {
  return boxStatus.value?.opened_rewards.some((reward) => itemIds.includes(reward.item_id)) ?? false
}

function showNewRingBenefitOnFirstVisit() {
  const userId = authStore.user?.user_id
  if (!userId) return

  const storageKey = `lottery_new_ring_benefit_seen:${userId}`
  if (localStorage.getItem(storageKey)) return

  localStorage.setItem(storageKey, 'true')
  showNewRingBenefit.value = true
}

function mapStageRewardsToNodes(rewards: StageRewardItem[]): ProgressNode[] {
  return rewards.map((reward) => {
    let status: ProgressStatus = 'locked'
    if (reward.claimed) status = 'claimed'
    else if (reward.reachable) status = 'available'

    return {
      id: reward.slot_index + 1,
      item_id: reward.item_id,
      name: reward.item_name,
      lit_threshold: reward.lit_threshold,
      icon: getRewardIconKind(reward),
      status,
    }
  })
}

function rewardAriaLabel(node: ProgressNode) {
  const statusText = node.status === 'available' ? '可领取' : node.status === 'claimed' ? '已领取' : '未解锁'
  return `${node.name}，需要点亮${node.lit_threshold}朵花，${statusText}`
}

function getStageThresholdAsset(litThreshold: number) {
  return stageThresholdAssets[litThreshold]
}

function getStageRewardAsset(itemId: number) {
  if (itemId === 1002) return '/image/activity/petal.png'
  return getRingCatalogItem(itemId)?.image
}

function getStageRewardAnimatedAsset(itemId: number) {
  const assets: Partial<Record<number, string>> = {
    1207244: '/image/activity/rings/ring-1207244-firework.svg',
    1205251: '/image/activity/rings/ring-1205251-moon.svg',
    1207751: '/image/activity/rings/ring-1207751-true-love-classic.svg',
  }
  return assets[itemId]
}

function getDrawButtonImage(count: DrawCount) {
  return drawButtonAssets[activeTab.value][count]
}

function setDrawButtonRef(count: DrawCount, element: unknown) {
  if (element instanceof HTMLButtonElement) drawButtonRefs.set(count, element)
  else drawButtonRefs.delete(count)
}

function getDrawButtonAnimation(count: DrawCount) {
  return drawButtonAnimations[count]
}

function getRectCenter(rect: DOMRect) {
  return {
    x: rect.left + rect.width / 2,
    y: rect.top + rect.height / 2,
  }
}

function getFallbackDrawAnimationTransform() {
  const sceneRect = activityScrollRef.value?.getBoundingClientRect()
  const viewportWidth = sceneRect && sceneRect.width > 0 ? sceneRect.width : window.innerWidth
  const viewportHeight = sceneRect && sceneRect.height > 0 ? sceneRect.height : window.innerHeight
  const viewportLeft = sceneRect && sceneRect.width > 0 ? sceneRect.left : 0
  const viewportTop = sceneRect && sceneRect.height > 0 ? sceneRect.top : 0
  const scale = Math.max(0.01, Math.min(
    viewportWidth / DRAW_ANIMATION_CANVAS_SIZE.width,
    Math.max(1, viewportHeight - 20) / DRAW_ANIMATION_CANVAS_SIZE.height,
  ))

  return toCssMatrix({
    a: scale,
    b: 0,
    c: 0,
    d: scale,
    e: viewportLeft + (viewportWidth - DRAW_ANIMATION_CANVAS_SIZE.width * scale) / 2,
    f: viewportTop + viewportHeight - 20 - DRAW_ANIMATION_CANVAS_SIZE.height * scale,
  })
}

function getDrawAnimationTransform(count: DrawCount) {
  const buttonRect = drawButtonRefs.get(count)?.getBoundingClientRect()
  const targetRect = flowerBoxRef.value?.getDrawAnimationTargetRect()
  if (!buttonRect || !targetRect) return getFallbackDrawAnimationTransform()

  const transform = createDrawAnimationTransform(
    count,
    getRectCenter(buttonRect),
    getRectCenter(targetRect),
  )
  return transform ? toCssMatrix(transform) : getFallbackDrawAnimationTransform()
}

function getDrawButtonAriaLabel(count: DrawCount) {
  if (currentCostPerDraw.value === 0) return `许愿${count}次，奖池加载中`

  const totalCost = currentCostPerDraw.value * count
  const drawingState = isDrawing.value ? '，许愿中' : ''
  return `许愿${count}次${drawingState}，消耗${totalCost.toLocaleString()}花瓣`
}

function getTryBeforePayAriaLabel() {
  if (isTryBeforePayLoading.value) return '先抽后付，预览生成中'
  if (isTryBeforePayConfirming.value) return '先抽后付，确认中'
  return '先抽后付，180抽，消耗900花瓣，必出稀有及以上奖励'
}

async function loadActivityStatus(showEntryPrompt = false) {
  try {
    activityConfig.value = await fetchActivityConfig()
    if (showEntryPrompt && activityConfig.value.status === 2) {
      showToast('活动已结束')
    } else if (showEntryPrompt && activityConfig.value.status === 0) {
      showToast('活动尚未开始')
    }
  } catch {
    activityConfig.value = null
    showToast('活动状态加载失败，请刷新重试')
  }
}

function ensureActivityActive() {
  if (activityActive.value) return true
  showToast(activityStatusText.value)
  return false
}

function openPetalShop() {
  if (!ensureActivityActive()) return
  showPetalShop.value = true
}

function openPetalGift() {
  if (!ensureActivityActive()) return
  showPetalGift.value = true
}

async function fetchAssets() {
  try {
    const result = await getAssets()
    petals.value = result.petals
    gold.value = result.gold
  } catch {
    showToast('资产加载失败，请刷新重试')
  }
}

async function loadPetalGiftStatus() {
  try {
    const result = await getPetalGiftStatus()
    petalGiftAvailable.value = result.available
  } catch {
    petalGiftAvailable.value = false
  }
}

async function handlePurchasePetalGift() {
	if (isPurchasingPetalGift.value || !ensureActivityActive()) return
  isPurchasingPetalGift.value = true
  try {
    await purchasePetalGift()
    petalGiftAvailable.value = false
    showPetalGiftResult.value = true
    await fetchAssets()
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '礼包购买失败，请重试')
    await loadPetalGiftStatus()
  } finally {
    isPurchasingPetalGift.value = false
  }
}

function closePetalGiftResult() {
  showPetalGiftResult.value = false
  showPetalGift.value = false
}

function handlePetalShopClose() {
  showPetalShop.value = false
  if (tryBeforePaySessionId.value === null || tryBeforePayPreview.value === null) return

  showTryBeforePay.value = true
  void loadTryBeforePayStatus(true)
}

async function loadPools() {
  try {
    const result = await fetchPools()
    pools.value = result
    result.forEach((pool) => {
      poolCosts.value[pool.pool_type] = pool.cost_per_draw
      poolNames.value[pool.pool_type] = pool.pool_name
    })
  } catch {
    showToast('奖池加载失败，请刷新重试')
  }
}

async function loadTryBeforePayStatus(showError = false) {
  isTryBeforePayStatusLoading.value = true
  if (!activityActive.value) {
    tryBeforePayAvailable.value = false
    showTryBeforePayOffer.value = false
    isTryBeforePayStatusLoading.value = false
    return
  }
  try {
    const result = await fetchTryBeforePayStatus()
    tryBeforePayAvailable.value = result.available
    tryBeforePaySessionId.value = result.session?.session_id ?? null

    if (result.session?.status === 1) {
      const preview = result.preview ?? await previewTryBeforePay()
      if (preview.session_id !== result.session.session_id) {
        throw new Error('先抽后付预览会话不一致')
      }
      tryBeforePayPreview.value = preview
      showTryBeforePay.value = true
      showTryBeforePayAbandonConfirm.value = false
      showTryBeforePayOffer.value = false
      showNewRingBenefit.value = false
      tryBeforePayOfferHandled.value = true
      return
    }

    showTryBeforePay.value = false
    showTryBeforePayAbandonConfirm.value = false
    tryBeforePayPreview.value = null

    if (result.available && !tryBeforePayOfferHandled.value) {
      showTryBeforePayOffer.value = true
      tryBeforePayOfferHandled.value = true
    }
  } catch {
    if (tryBeforePayPreview.value === null) {
      tryBeforePayAvailable.value = false
      tryBeforePaySessionId.value = null
    }
    if (showError) showToast('先抽后付状态加载失败，请刷新重试')
  } finally {
    isTryBeforePayStatusLoading.value = false
  }
}

async function loadBoxStatus(showError = false, showLoading = true) {
  if (showLoading) boxLoadState.value = 'loading'
  try {
    boxStatus.value = await fetchBoxStatus()
    boxLoadState.value = 'ready'
  } catch {
    boxLoadState.value = 'error'
    if (showError) showToast('召唤进度加载失败，请重新同步')
  }
}

async function loadPendingDrawChoices(showError = false) {
  try {
    drawPendingChoices.value = await fetchPendingDrawChoices()
  } catch {
    if (showError) showToast('待选择的抽奖奖励加载失败，请刷新重试')
  }
}

async function loadStageRewards() {
  stageLoadState.value = 'loading'
  try {
    const result = await fetchStageRewards()
    currentFlowers.value = result.current_lit_count
    progressNodes.value = mapStageRewardsToNodes(result.rewards)
    stagePendingChoice.value = result.pending_choice ?? null
    stageLoadState.value = 'ready'
  } catch {
    stageLoadState.value = 'error'
  }
}

async function handleClaimStageReward(slotIndex: number) {
	if (!ensureActivityActive()) return
  try {
    const result = await claimStageReward(slotIndex)
    clearRecordCaches()
    await Promise.all([loadStageRewards(), fetchAssets()])
    if (result.pending_choice) stagePendingChoice.value = result.pending_choice
    else stageRewardResult.value = result
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '领取失败')
  }
}

async function handleRefresh() {
  if (isRefreshing.value) return
  isRefreshing.value = true
  await loadActivityStatus()
  // Load pool costs before showing draw/TBP entry points; the buttons stay
  // disabled while currentCostPerDraw is zero.
  await loadPools()
  await fetchAssets()
  await loadBoxStatus(true)
  await loadStageRewards()
  await loadPendingDrawChoices(true)
  await loadTryBeforePayStatus(true)
  // 重连 SSE 以获取最新弹幕
  disconnectMarqueeSSE()
  connectMarqueeSSE()
  isRefreshing.value = false
  showToast('活动数据已刷新')
}

// 使用 @microsoft/fetch-event-source 替代 EventSource，支持 Authorization 自定义头。
// 网关统一鉴权后，SSE 端点需要通过 Bearer Token 验证用户身份。
function connectMarqueeSSE() {
  if (marqueeEventSource.value) return

  const token = getToken()
  if (!token) return

  marqueeSSEReconnectCount.value = 0
  marqueeSSEDegraded.value = false

  const client = createAuthSSE(
    '/api/notification/v1/barrage/stream',
    getToken,
    {
      onOpen: () => {
        marqueeSSEReconnectCount.value = 0
        marqueeSSEDegraded.value = false
      },
      onEvent: (event: string, data: string) => {
        if (event !== 'barrage') return
        try {
          const payload = JSON.parse(data) as {
            message_id?: string
            display_name?: string
            item_name?: string
            quantity?: number
          }
          if (!payload.message_id) return
          const item: RingMarqueeItem = {
            id: payload.message_id,
            nickname: payload.display_name || '用户',
            item_name: payload.item_name || '',
            obtained_at: new Date().toISOString(),
          }
          // 去重：已播放或已在队列中的忽略
          if (playedMarqueeIds.has(item.id) || queuedMarqueeIds.has(item.id)) return
          // 队列过载保护
          if (marqueeQueue.value.length + (currentMarquee.value ? 1 : 0) >= MARQUEE_MAX_QUEUE) return

          queuedMarqueeIds.add(item.id)
          marqueeQueue.value.push(item)
          playNextMarquee()
        } catch {
          // 解析失败，跳过
        }
      },
      onAuthRequired: () => {
        marqueeEventSource.value?.close()
        marqueeEventSource.value = null
        void (async () => {
          if (await authStore.refreshSession()) connectMarqueeSSE()
          else router.replace('/login')
        })()
      },
      onUnavailable: () => {
        marqueeSSEDegraded.value = true
        marqueeEventSource.value?.close()
        marqueeEventSource.value = null
      },
      onError: (reconnectCount: number) => {
        marqueeSSEReconnectCount.value = reconnectCount
      },
      onReconnectMax: () => {
        marqueeSSEDegraded.value = true
        marqueeEventSource.value = null
      },
    },
    5 // 最大重连次数，与原 EventSource 行为一致
  )

  marqueeEventSource.value = client
}

function disconnectMarqueeSSE() {
  marqueeEventSource.value?.close()
  marqueeEventSource.value = null
  marqueeSSEReconnectCount.value = 0
  marqueeSSEDegraded.value = false
}

function restorePlayedMarqueeIds(): Set<string> {
  const sessionID = authStore.user?.user_id ?? ''
  if (!sessionID) return new Set<string>()

  try {
    const stored = JSON.parse(localStorage.getItem(MARQUEE_PLAYED_STORAGE_KEY) ?? '{}') as {
      session_id?: string
      ids?: string[]
    }
    if (stored.session_id !== sessionID || !Array.isArray(stored.ids)) return new Set<string>()
    return new Set(stored.ids.filter((id) => typeof id === 'string'))
  } catch {
    return new Set<string>()
  }
}

function persistPlayedMarqueeIds() {
  const sessionID = authStore.user?.user_id ?? ''
  if (!sessionID) return

  try {
    localStorage.setItem(MARQUEE_PLAYED_STORAGE_KEY, JSON.stringify({
      session_id: sessionID,
      ids: [...playedMarqueeIds].slice(-200),
    }))
  } catch {
    // 存储不可用时仍保持当前页面内不重复播放。
  }
}

function playNextMarquee() {
  if (currentMarquee.value || marqueeQueue.value.length === 0) return
  currentMarquee.value = marqueeQueue.value.shift() ?? null
}

function handleMarqueeAnimationEnd() {
  const played = currentMarquee.value
  if (!played) return

  playedMarqueeIds.add(played.id)
  queuedMarqueeIds.delete(played.id)
  persistPlayedMarqueeIds()
  currentMarquee.value = null
  playNextMarquee()
}

function handleVisibilityChange() {
  // 页面隐藏时保持 SSE 连接（sse-client 已配置 openWhenHidden: true），
  // 确保切回前台时弹幕进度与其他窗口同步，不会出现"延迟回放"。
  if (document.hidden) return
  // 页面恢复可见时，若 SSE 已因其他原因断开则重连
  if (!marqueeEventSource.value) connectMarqueeSSE()
}

async function handleLogout() {
  pauseActivityAudio()
  await authStore.logout()
  router.replace('/login')
}

function goToAdmin() {
  window.open('/admin/', '_blank')
}

async function executeDraw(poolType: number, count: number) {
	if (!ensureActivityActive()) return
  isDrawing.value = true
  await scrollActivityToBottom()
  playActivityEffect('drawStart')
  const animation = playDrawAnimation(count as DrawCount)
  try {
    const [result] = await Promise.all([
      draw({ pool_type: poolType, count }),
      animation,
    ])
    drawResult.value = result
    clearRecordCaches()
    playActivityEffect('reward')
    await Promise.all([fetchAssets(), loadStageRewards(), loadTryBeforePayStatus()])
  } catch (error: unknown) {
    finishDrawAnimation()
    showToast(error instanceof Error ? error.message : '许愿失败，请重试')
  } finally {
    isDrawing.value = false
  }
}

async function scrollActivityToBottom() {
  await nextTick()
  const scroller = activityScrollRef.value
  if (!scroller) return

  const target = Math.max(0, scroller.scrollHeight - scroller.clientHeight)
  if (target - scroller.scrollTop <= 1) return

  const reducedMotion = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  if (typeof scroller.scrollTo === 'function') {
    scroller.scrollTo({ top: target, behavior: reducedMotion ? 'auto' : 'smooth' })
  } else {
    scroller.scrollTop = target
  }
  if (reducedMotion || Math.abs(target - scroller.scrollTop) <= 1) return

  await new Promise<void>((resolve) => {
    const startedAt = performance.now()
    const checkPosition = (time: number) => {
      if (Math.abs(target - scroller.scrollTop) <= 1 || time - startedAt >= 700) {
        if (Math.abs(target - scroller.scrollTop) > 1) scroller.scrollTop = target
        resolve()
        return
      }
      requestAnimationFrame(checkPosition)
    }
    requestAnimationFrame(checkPosition)
  })
  await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()))
}

function playDrawAnimation(count: DrawCount) {
  if (skipAnim.value) return Promise.resolve()

  finishDrawAnimation()
  return new Promise<void>((resolve) => {
    resolveDrawAnimation = resolve
    drawAnimation.value = {
      key: ++drawAnimationSequence,
      src: getDrawButtonAnimation(count),
      transform: getDrawAnimationTransform(count),
    }
  })
}

function finishDrawAnimation() {
  drawAnimation.value = null
  resolveDrawAnimation?.()
  resolveDrawAnimation = null
}

async function settleDrawResult() {
  if (drawResult.value === null || isConfirmingDrawResult.value) return false

  isConfirmingDrawResult.value = true
  try {
    const result = drawResult.value
    drawResult.value = null
    const choices = result.rewards
      .filter((reward) => reward.is_choice && reward.reward_record_id && reward.choice_ids?.length)
      .map((reward) => ({
        reward_record_id: reward.reward_record_id!,
        item_id: reward.item_id,
        item_name: reward.item_name,
        quantity: reward.quantity,
        choice_ids: reward.choice_ids!,
      }))
    if (choices.length > 0) {
      drawPendingChoices.value = choices
      drawFlowAfterChoices.value = result
    } else {
      await continueDrawResultFlow(result)
    }
  } finally {
    isConfirmingDrawResult.value = false
  }
  return true
}

async function onDrawResultConfirm() {
  await settleDrawResult()
}

async function onDrawResultRedraw(count: number) {
  pendingRepeatDrawCount.value = count
  const settled = await settleDrawResult()
  if (!settled) {
    pendingRepeatDrawCount.value = null
    return
  }
  await continuePendingRepeatDraw()
}

async function continuePendingRepeatDraw() {
  const count = pendingRepeatDrawCount.value
  if (count === null) return
  if (
    drawResult.value !== null
    || flowerLitNotice.value !== null
    || refundResult.value !== null
    || drawPendingChoices.value.length > 0
    || trueLoveAcquired.value !== null
    || boxFlowBlocking.value
  ) return

  pendingRepeatDrawCount.value = null
  await handleDraw(count)
}

async function continueDrawResultFlow(result: DrawResponse) {
  if (result.lit_flowers > 0) {
    flowerLitNotice.value = {
      litFlowers: result.lit_flowers,
      litCountAfter: result.lit_count_after,
      drawResult: result,
    }
  } else if (result.refund_petals > 0) {
    refundResult.value = result
  } else {
    await loadBoxStatus(true, false)
  }
}

async function onFlowerLitConfirm() {
  const notice = flowerLitNotice.value
  if (!notice) return

  flowerLitNotice.value = null
  if (notice.drawResult?.refund_petals) {
    refundResult.value = notice.drawResult
    return
  }
  await loadBoxStatus(true, false)
}

async function onRefundConfirm() {
  await loadBoxStatus(true, false)
  refundResult.value = null
  await continuePendingRepeatDraw()
}

async function autoBuyAndDraw(count: number, shortage: number) {
	if (!ensureActivityActive()) return
  const goldCost = shortage * 60
  if (gold.value < goldCost) {
    showGoldInsufficientThenShop()
    return
  }

  try {
    await exchangePetals(shortage)
    await fetchAssets()
    await executeDraw(activePoolType.value, count)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '购买花瓣失败')
  }
}

async function handleDraw(count: number) {
	if (isDrawing.value || boxFlowBlocking.value || !ensureActivityActive()) return
  const costPerDraw = currentCostPerDraw.value
  if (costPerDraw === 0) {
    showToast('奖池加载中，请稍后')
    return
  }

  const totalCost = costPerDraw * count
  if (petals.value >= totalCost) {
    await executeDraw(activePoolType.value, count)
    return
  }

  pendingDrawCount.value = count
  currentShortage.value = totalCost - petals.value
  const goldCost = currentShortage.value * 60
  if (gold.value < goldCost) {
    showGoldInsufficientThenShop()
    return
  }
  if (showConfirm.value) showBuyRemaining.value = true
  else await autoBuyAndDraw(count, currentShortage.value)
}

async function handleOpenBox() {
	if (!ensureActivityActive()) return
  const chanceIndex = boxStatus.value?.next_chance_index
  if (!chanceIndex || isOpeningBox.value) return

  isOpeningBox.value = true
  try {
    await openBox(chanceIndex)
    await loadBoxStatus(true)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '戒指召唤失败，请重试')
    await loadBoxStatus()
  } finally {
    isOpeningBox.value = false
  }
}

async function handleConfirmBox(selectedItemId: number) {
  const chanceIndex = boxStatus.value?.pending_reward?.chance_index
  if (!chanceIndex || isConfirmingBox.value) return

  isConfirmingBox.value = true
  try {
    const result = await confirmBox(chanceIndex, selectedItemId)
    clearRecordCaches()
    await Promise.all([loadBoxStatus(true), loadStageRewards(), fetchAssets()])
    if (isTrueLoveRing(result.item_id)) showTrueLoveAcquired(result.item_id, result.item_name)
    else showToast(`已获得 ${result.item_name}`)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '召唤结果确认失败，请重试')
    await loadBoxStatus()
  } finally {
    isConfirmingBox.value = false
  }
}

async function handleConfirmStageChoice(selectedItemId: number) {
  const pending = stagePendingChoice.value
  if (!pending || isConfirmingStageChoice.value) return

  isConfirmingStageChoice.value = true
  try {
    const result = await confirmStageRewardChoice(pending.reward_record_id, selectedItemId)
    clearRecordCaches()
    stagePendingChoice.value = null
    await loadStageRewards()
    showTrueLoveAcquired(result.item_id, result.item_name)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '阶段奖励戒指选择失败，请重试')
    await loadStageRewards()
  } finally {
    isConfirmingStageChoice.value = false
  }
}

async function handleConfirmDrawChoice(selectedItemId: number) {
  const pending = drawPendingChoices.value[0]
  if (!pending || isConfirmingDrawChoice.value) return

  isConfirmingDrawChoice.value = true
  try {
    const result = await confirmDrawChoice(pending.reward_record_id, selectedItemId)
    clearRecordCaches()
    drawPendingChoices.value.shift()
    if (drawPendingChoices.value.length === 0 && drawFlowAfterChoices.value) {
      const drawResultToContinue = drawFlowAfterChoices.value
      drawFlowAfterChoices.value = null
      showTrueLoveAcquired(result.item_id, result.item_name, () => continueDrawResultFlow(drawResultToContinue))
    } else if (drawPendingChoices.value.length === 0 && tbpFlowAfterChoices.value) {
      const tbpResultToContinue = tbpFlowAfterChoices.value
      tbpFlowAfterChoices.value = null
      showTrueLoveAcquired(result.item_id, result.item_name, () => continueTryBeforePayResultFlow(tbpResultToContinue))
    } else {
      showTrueLoveAcquired(result.item_id, result.item_name)
    }
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '抽奖奖励戒指选择失败，请重试')
  } finally {
    isConfirmingDrawChoice.value = false
  }
}

function showTrueLoveAcquired(
  itemId: number,
  itemName: string,
  continuation: (() => Promise<void> | void) | null = null,
) {
  trueLoveAcquiredContinuation = continuation
  trueLoveAcquired.value = { itemId, itemName }
}

function isTrueLoveRing(itemId: number) {
  return itemId === 1207751 || itemId === 1207752 || itemId === 1207753
}

async function handleTrueLoveAcquiredConfirm() {
  const continuation = trueLoveAcquiredContinuation
  trueLoveAcquired.value = null
  trueLoveAcquiredContinuation = null
  await continuation?.()
  await continuePendingRepeatDraw()
}

function handleConfirmSummon(selectedItemId: number) {
  if (drawPendingChoices.value.length > 0) {
    void handleConfirmDrawChoice(selectedItemId)
    return
  }
  if (stagePendingChoice.value) {
    void handleConfirmStageChoice(selectedItemId)
    return
  }
  void handleConfirmBox(selectedItemId)
}

async function handleStartNewRound() {
	if (!roundComplete.value || isStartingNewRound.value || !ensureActivityActive()) return
  if (hasUnclaimedStageReward.value) {
    showNewRoundNotice('请领取花朵累计任务奖励～')
    return
  }
  if (stagePendingChoice.value) {
    showNewRoundNotice('请领取花朵累计任务奖励～')
    return
  }
  if (boxFlowBlocking.value) {
    showToast('请先完成本轮戒指召唤')
    return
  }

  isStartingNewRound.value = true
  try {
    const result = await startNewRound()
    clearRecordCaches()
    await Promise.all([loadStageRewards(), loadBoxStatus(true, false)])
    showToast(`已开始第${result.round_number}轮`)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '开始新一轮失败，请重试')
  } finally {
    isStartingNewRound.value = false
  }
}

function showNewRoundNotice(message: string) {
  newRoundNotice.value = message
  if (newRoundNoticeTimer) clearTimeout(newRoundNoticeTimer)
  newRoundNoticeTimer = setTimeout(() => {
    newRoundNotice.value = ''
    newRoundNoticeTimer = null
  }, 2200)
}

onUnmounted(() => {
  if (newRoundNoticeTimer) clearTimeout(newRoundNoticeTimer)
  if (goldInsufficientNoticeTimer) clearTimeout(goldInsufficientNoticeTimer)
  disconnectMarqueeSSE()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

function showGoldInsufficientThenShop() {
  goldInsufficientNotice.value = '金币不足～'
  if (goldInsufficientNoticeTimer) clearTimeout(goldInsufficientNoticeTimer)
  goldInsufficientNoticeTimer = setTimeout(() => {
    goldInsufficientNotice.value = ''
    if (activityActive.value) showPetalShop.value = true
    goldInsufficientNoticeTimer = null
  }, 1200)
}

async function onBuyRemainingConfirm(dontShowAgain = false) {
	if (!ensureActivityActive()) {
		showBuyRemaining.value = false
		return
	}
  if (dontShowAgain) showConfirm.value = false
  showBuyRemaining.value = false
  const goldCost = currentShortage.value * 60
  if (gold.value < goldCost) {
    showGoldInsufficientThenShop()
    return
  }

  try {
    await exchangePetals(currentShortage.value)
    await fetchAssets()
    await executeDraw(activePoolType.value, pendingDrawCount.value)
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '购买花瓣失败')
  }
}

function onBuyRemainingGoShop(dontShowAgain = false) {
  if (dontShowAgain) showConfirm.value = false
  showBuyRemaining.value = false
	if (ensureActivityActive()) showGoldInsufficientThenShop()
}

async function handleTryBeforePay() {
	if (isTryBeforePayLoading.value || isTryBeforePayConfirming.value || !ensureActivityActive()) return

  isTryBeforePayLoading.value = true
  try {
    const preview = await previewTryBeforePay()
    tryBeforePayPreview.value = preview
    tryBeforePaySessionId.value = preview.session_id
    showTryBeforePay.value = true
    playActivityEffect('tryBeforePayPreview')
  } catch (error: unknown) {
    showToast(error instanceof Error ? error.message : '先抽后付预览生成失败')
    await loadTryBeforePayStatus()
  } finally {
    isTryBeforePayLoading.value = false
  }
}

function dismissTryBeforePayOffer() {
  showTryBeforePayOffer.value = false
}

async function useTryBeforePayOffer() {
  showTryBeforePayOffer.value = false
  await handleTryBeforePay()
}

function requestTryBeforePayAbandon() {
  if (isTryBeforePayConfirming.value || isTryBeforePayLoading.value) return
  showTryBeforePayAbandonConfirm.value = true
}

async function confirmTryBeforePayAbandon() {
	if (isTryBeforePayConfirming.value || isTryBeforePayLoading.value || !ensureActivityActive()) return

  if (tryBeforePaySessionId.value === null) return
  isTryBeforePayLoading.value = true
  try {
    await abandonTryBeforePay()
    showTryBeforePayAbandonConfirm.value = false
    showTryBeforePay.value = false
    tryBeforePayPreview.value = null
    tryBeforePayAvailable.value = false
  } catch {
    showToast('先抽后付预览放弃失败，请稍后重试')
  } finally {
    tryBeforePaySessionId.value = null
    await loadTryBeforePayStatus()
    isTryBeforePayLoading.value = false
  }
}

async function onTryBeforePayConfirm() {
	if (isTryBeforePayConfirming.value || tryBeforePayPreview.value === null || !ensureActivityActive()) return

  isTryBeforePayConfirming.value = true
  try {
    const preview = tryBeforePayPreview.value
    await confirmTryBeforePay()
    clearRecordCaches()
    await Promise.all([
      fetchAssets(),
      loadStageRewards(),
      loadTryBeforePayStatus(),
      loadPendingDrawChoices(true),
    ])
    showTryBeforePay.value = false
    tryBeforePayPreview.value = null
    tryBeforePaySessionId.value = null
    const resultFlow = {
      litFlowers: preview.estimated_lit_flowers,
      litCountAfter: currentFlowers.value,
    }
    if (drawPendingChoices.value.length > 0) {
      tbpFlowAfterChoices.value = resultFlow
    } else {
      await continueTryBeforePayResultFlow(resultFlow)
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message : '先抽后付确认失败'
    if (message.includes('金币不足')) {
      showTryBeforePay.value = false
      showGoldInsufficientThenShop()
    } else {
      showToast(message)
    }
  } finally {
    isTryBeforePayConfirming.value = false
  }
}

async function continueTryBeforePayResultFlow(result: { litFlowers: number; litCountAfter: number }) {
  if (result.litFlowers > 0) {
    flowerLitNotice.value = {
      litFlowers: result.litFlowers,
      litCountAfter: result.litCountAfter,
      drawResult: null,
    }
    return
  }
  await loadBoxStatus(true, false)
  showToast('支付成功，奖励已发放')
}

onMounted(async () => {
  initializeActivityAudio()
  ;[
    '/image/background1.png',
    '/image/background2.png',
    ...Object.values(wishTabAssets),
    ...Object.values(guideAssets),
    ...Object.values(allRewardsAssets),
  ].forEach((source) => {
    const image = new Image()
    image.src = source
  })

  if (!authStore.user) {
    const authenticated = await authStore.initFromStorage()
    if (!authenticated) {
      router.replace('/login')
      return
    }
  }
  playedMarqueeIds = restorePlayedMarqueeIds()
  queuedMarqueeIds.clear()

  await loadActivityStatus(true)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  // 建立 SSE 长连接，代替轮询
  connectMarqueeSSE()
  await loadPools()
  await fetchAssets()
  await loadBoxStatus(true)
  await loadStageRewards()
  await loadPendingDrawChoices(true)
  await loadTryBeforePayStatus()
  await loadPetalGiftStatus()
  if (activityActive.value && !showTryBeforePay.value) showNewRingBenefitOnFirstVisit()
})
</script>

<style scoped>
:global(body) {
  background: #160c25;
  overscroll-behavior-y: contain;
}

:global(html),
:global(body),
:global(#app) {
  height: 100%;
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

.activity-page {
  --draw-dock-base-height: clamp(132px, 36.8vw, 158px);
  --draw-dock-height: calc(var(--draw-dock-base-height) + env(safe-area-inset-bottom));
  --rose-50: #fff7fb;
  --rose-100: #ffe9f2;
  --rose-300: #f6b5cc;
  --rose-500: #d45882;
  --rose-700: #8f2f53;
  --plum-900: #2e173d;
  --gold: #f7d793;
  height: 100dvh;
  overflow: hidden;
  background: radial-gradient(circle at 50% 0%, #4a2b63, #160c25 72%);
  color: var(--rose-50);
}

.activity-scene {
  position: relative;
  width: min(100%, 430px);
  height: 100dvh;
  margin: 0 auto;
  overflow: hidden;
  background-color: #24142f;
  box-shadow: 0 0 44px rgba(0, 0, 0, 0.46);
  isolation: isolate;
}

.activity-scroll {
  height: 100%;
  overflow-x: hidden;
  overflow-y: auto;
  padding-bottom: calc(var(--draw-dock-height) + 12px);
  background-color: #24142f;
  background-image:
    linear-gradient(180deg, transparent 0 41%, rgba(61, 23, 58, 0.08) 53%, rgba(29, 12, 40, 0.72) 77%, #170c25 100%),
    var(--activity-background);
  background-repeat: no-repeat;
  background-position: top center;
  background-size: 100% 100%, 100% auto;
  background-attachment: local, local;
  overscroll-behavior-y: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
  transition: background-image 320ms ease-out;
}

.activity-scroll::-webkit-scrollbar {
  display: none;
}

button {
  font: inherit;
  touch-action: manipulation;
  -webkit-tap-highlight-color: transparent;
}

button:focus-visible,
input:focus-visible + span {
  outline: 3px solid #fff;
  outline-offset: 2px;
}

.scene-nav {
  position: relative;
  z-index: 30;
  height: calc(44px + env(safe-area-inset-top));
  padding-top: env(safe-area-inset-top);
}

.scene-nav__backdrop {
  position: absolute;
  inset: env(safe-area-inset-top) 0 auto;
  width: 100%;
  height: 44px;
}

.scene-nav__button {
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
}

.scene-nav__button img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.scene-nav__button--back {
  position: absolute;
  top: env(safe-area-inset-top);
  left: 12px;
}

.scene-nav__actions {
  position: absolute;
  top: env(safe-area-inset-top);
  right: 10px;
  display: flex;
  isolation: isolate;
  gap: 4px;
}

.scene-nav__button:active { transform: scale(0.92); }
.scene-nav__button:disabled { cursor: wait; opacity: 0.65; }
.scene-nav__button--loading img { animation: spin 700ms linear infinite; }

.hero-space { height: clamp(76px, 24.8vw, 106px); }

.marquee-slot {
  position: relative;
  height: 25px;
  z-index: 25;
}

.marquee {
  --marquee-background: linear-gradient(90deg, transparent, rgba(103, 42, 65, 0.78) 14%, rgba(103, 42, 65, 0.78) 86%, transparent);
  --marquee-border: rgba(255, 237, 245, 0.36);
  --marquee-color: #fffbe7;
  --marquee-spark: #f7d793;
  position: relative;
  width: 100%;
  height: 25px;
  overflow: hidden;
  background: var(--marquee-background);
  border-block: 1px solid var(--marquee-border);
  pointer-events: none;
  transform: translateY(clamp(-74px, calc(-52px - 4.7vw), -66px));
}

.marquee--night {
  --marquee-background: linear-gradient(90deg, transparent, rgba(24, 47, 112, 0.82) 14%, rgba(24, 47, 112, 0.82) 86%, transparent);
  --marquee-border: rgba(195, 218, 255, 0.48);
  --marquee-color: #f6f8ff;
  --marquee-spark: #ffe79b;
}

.marquee__track {
  display: flex;
  width: 100%;
  height: 100%;
  align-items: center;
  justify-content: center;
  animation: marquee-once 7s linear forwards;
}

.marquee__item {
  display: inline-flex;
  max-width: calc(100% - 32px);
  flex-shrink: 0;
  align-items: center;
  gap: 7px;
  color: var(--marquee-color);
  font-size: 12px;
  white-space: nowrap;
  text-shadow: 0 1px 4px rgba(84, 6, 121, 0.75);
}

.marquee__spark {
  flex: 0 0 auto;
  width: 8px;
  height: 8px;
  background: var(--marquee-spark);
  clip-path: polygon(50% 0, 61% 38%, 100% 50%, 61% 62%, 50% 100%, 39% 62%, 0 50%, 39% 38%);
  filter: drop-shadow(0 0 4px #fff);
}

.marquee__text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.wish-tabs {
  position: relative;
  z-index: 8;
  display: grid;
  width: min(75%, 304px);
  height: auto;
  aspect-ratio: 1072 / 244;
  margin: -6px auto -5px;
  grid-template-columns: 1fr 1fr;
  align-items: stretch;
  isolation: isolate;
  transform: translateY(24px);
}

.wish-tabs__art {
  position: absolute;
  inset: 0;
  z-index: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.wish-tab {
  position: relative;
  z-index: 1;
  min-width: 0;
  height: 100%;
  padding: 0;
  color: transparent;
  background: transparent;
  border: 0;
  border-radius: 8px;
  cursor: pointer;
  transition: transform 180ms ease;
}

.wish-tab:active { transform: scale(0.97); }

.guide-section {
  position: relative;
  z-index: 35;
  height: clamp(118px, 30.9vw, 133px);
  margin-top: -5px;
  pointer-events: none;
}

.guide-section__art {
  position: absolute;
  top: 0;
  left: 0;
  width: 71.8%;
  height: auto;
  object-fit: contain;
  pointer-events: none;
  user-select: none;
}

.guide-section__all-rewards {
  position: absolute;
  z-index: 12;
  top: clamp(18px, 5.5vw, 24px);
  right: clamp(110px, calc(16vw + 50px), 119px);
  width: clamp(43px, 11.45vw, 50px);
  aspect-ratio: 96 / 102;
  pointer-events: auto;
}

.scene-entry {
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 180ms ease, transform 180ms ease;
}

.scene-entry img { display: block; width: 100%; height: 100%; object-fit: contain; pointer-events: none; user-select: none; }
.scene-entry:active { transform: scale(0.93); filter: brightness(1.12); }

.scene-entry--rules {
  position: absolute;
  z-index: 2;
  top: -114px;
  right: 0;
  display: block;
  width: 72px;
  height: auto;
  aspect-ratio: 144 / 136;
  pointer-events: auto;
}

.scene-entry--rank,
.scene-entry--benefit {
  position: absolute;
  z-index: 12;
  right: 4px;
  pointer-events: auto;
}

.scene-entry--rank {
  top: 14px;
  width: 48px;
  height: 51px;
}

.scene-entry--benefit {
  top: 92px;
  width: 52px;
  height: 55px;
}

.showcase {
  position: relative;
  height: clamp(244px, 66vw, 284px);
  margin-top: -15px;
}

.showcase__gift {
  position: absolute;
  z-index: 8;
  top: 6px;
  left: 7px;
  width: 68px;
  height: 60px;
}

.showcase__box {
  position: absolute;
  z-index: 2;
  top: 64px;
  left: 27%;
  width: 72%;
  height: auto;
  pointer-events: none;
  filter: drop-shadow(0 8px 10px rgba(70, 27, 74, 0.34));
}

.showcase__ring-record {
  position: absolute;
  z-index: 10;
  top: calc(clamp(132px, 38vw, 164px) + 90px);
  right: -10px;
  display: grid;
  width: 88px;
  height: 44px;
  place-items: center;
}

.showcase__ring-record img {
  width: 62px;
  height: 13px;
  object-fit: contain;
}

.ring-card {
  position: absolute;
  z-index: 5;
  width: clamp(108px, 29vw, 126px);
  height: clamp(108px, 29vw, 126px);
  pointer-events: none;
}

.ring-card--moon { top: -10px; right: 14%; }
.ring-card--true-love { top: 12px; left: calc(22% + 10px); }
.ring-card--love { top: 112px; left: 7%; }

.ring-card__orb {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.ring-card__new {
  position: absolute;
  z-index: 2;
  top: -3%;
  left: -3%;
  width: 46%;
  height: auto;
}

.ring-card__reward {
  position: absolute;
  z-index: 1;
  top: 19%;
  left: 19%;
  width: 62%;
  height: 62%;
  filter: drop-shadow(0 3px 6px rgba(54, 17, 80, 0.48));
  transform: scale(3);
  transform-origin: center;
}

.ring-card__reward img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.ring-card--true-love .ring-card__reward {
  transform: scale(3.15);
}

.ring-card__name {
  position: absolute;
  right: 4%;
  bottom: 2%;
  left: 4%;
  overflow: hidden;
  color: #fff;
  font-size: clamp(9px, 2.45vw, 11px);
  font-weight: 700;
  line-height: 1.15;
  text-align: center;
  text-overflow: ellipsis;
  text-shadow: -1px -1px 0 #773477, 1px -1px 0 #773477, -1px 1px 0 #773477, 1px 1px 0 #773477, 0 2px 4px rgba(55, 17, 78, 0.8);
  white-space: nowrap;
}

.ring-card__obtained {
  position: absolute;
  z-index: 4;
  top: 24%;
  left: 24%;
  width: 52%;
  height: 52%;
  object-fit: contain;
}

.petal-balance {
  --petal-balance-bg: url('/image/activity/petal-balance-day.png');
  position: absolute;
  z-index: 10;
  bottom: 0;
  left: 0;
  display: flex;
  width: clamp(132px, 34vw, 146px);
  height: 26px;
  align-items: center;
  padding: 0 4px 0 2px;
  color: #9c3b3b;
  background: transparent;
  border: 0;
  box-shadow: none;
}

.activity-scene--night .petal-balance {
  --petal-balance-bg: url('/image/activity/petal-balance-night.png');
}

.petal-balance::before {
  position: absolute;
  inset: 0;
  z-index: 0;
  content: '';
  background: var(--petal-balance-bg) center / 100% 100% no-repeat;
  opacity: 0.72;
}

.petal-balance__icon { position: relative; z-index: 1; width: 28px; height: 28px; object-fit: contain; }
.petal-balance strong { position: relative; z-index: 1; flex: 1; font-size: 14px; font-variant-numeric: tabular-nums; text-align: center; }
.petal-balance button { position: relative; z-index: 1; display: grid; width: 32px; height: 32px; margin-right: -4px; padding: 4px; place-items: center; background: transparent; border: 0; cursor: pointer; }
.petal-balance button img { width: 22px; height: 22px; }

.showcase__my-rewards {
  position: absolute;
  z-index: 11;
  top: calc(100% + 3px);
  left: 0;
  width: clamp(66px, 17vw, 73px);
  aspect-ratio: 154 / 32;
}

.showcase__my-rewards img {
  display: block;
  width: 100%;
  height: 100%;
}

.progress-panel {
  position: relative;
  z-index: 6;
  width: 100%;
  min-height: 0;
  aspect-ratio: 750 / 477;
  margin-top: -3px;
  margin-bottom: calc(-1 * clamp(62px, 16.3vw, 70px));
  padding: 0;
  overflow: visible;
}

.progress-panel__art {
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

.progress-panel__header {
  position: absolute;
  inset: 0;
  z-index: 3;
  pointer-events: none;
}

.progress-panel__count {
  position: absolute;
  top: 21%;
  right: calc(2.5% - 7px);
  width: clamp(100px, 27vw, 116px);
  height: 24px;
  pointer-events: auto;
}

.progress-panel__count img { position: absolute; inset: 0; width: 100%; height: 100%; }
.progress-panel__count span { position: relative; display: flex; height: 100%; align-items: center; justify-content: center; color: #ffebf5; font-size: 10px; }
.progress-panel__count strong { margin-inline: 2px; color: #fff; font-size: 12px; font-variant-numeric: tabular-nums; }

.reward-list {
  position: absolute;
  top: 38.2%;
  right: 4%;
  bottom: 1.5%;
  left: 4%;
  z-index: 2;
  display: grid;
  margin: 0;
  padding: 0;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  list-style: none;
}

.reward-item button {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  align-items: center;
  padding: 0 1px;
  color: #fff;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.reward-item button:disabled { cursor: default; }
.reward-item__visual { position: relative; display: block; flex: 0 0 auto; width: 54px; height: 56px; }
.reward-item__icon { position: absolute; top: 9px; left: 50%; width: 37px; height: 37px; transform: translateX(-50%); filter: drop-shadow(0 3px 4px rgba(61, 23, 62, 0.42)); }
.reward-item__image { position: absolute; top: 3px; left: 50%; width: 48px; height: 48px; object-fit: contain; transform: translateX(-50%); filter: drop-shadow(0 3px 4px rgba(61, 23, 62, 0.42)); }
.reward-item__picture { position: absolute; top: 50%; left: 50%; z-index: 1; display: block; width: 144px; height: 144px; pointer-events: none; transform: translate(-50%, -50%); filter: drop-shadow(0 3px 4px rgba(61, 23, 62, 0.42)); }
.reward-item__picture .reward-item__image { position: static; display: block; width: 100%; height: 100%; transform: none; filter: none; }
.reward-item__name { display: -webkit-box; min-height: 25px; overflow: hidden; color: #fffdf7; font-size: 9px; line-height: 1.25; text-align: center; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.reward-item__threshold { display: block; width: min(52px, 100%); height: auto; margin-top: -1px; object-fit: contain; }
.reward-item__threshold-fallback { display: grid; width: 52px; min-height: 31px; place-items: center; color: #9c6549; font-size: 10px; }

.reward-item__claimed {
  position: absolute;
  z-index: 3;
  top: 5px;
  left: 50%;
  width: 44px;
  height: 44px;
  object-fit: contain;
  opacity: 0.82;
  pointer-events: none;
  transform: translateX(-50%);
}

.reward-item--locked .reward-item__threshold { filter: grayscale(1) brightness(0.8) contrast(1.08); }
.reward-item--locked .reward-item__threshold-fallback { color: #4f4f4f; background: rgba(225, 225, 225, 0.78); }
.reward-item--available .reward-item__visual { animation: reward-pulse 1.8s ease-in-out infinite; }

.progress-panel__state {
  position: absolute;
  top: 38.2%;
  right: 6%;
  bottom: 9%;
  left: 6%;
  z-index: 3;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: rgba(255, 255, 255, 0.82);
  font-size: 13px;
}

.progress-panel__state button {
  min-width: 88px;
  min-height: 44px;
  color: #722746;
  background: linear-gradient(#fff9e8, #f3c68c);
  border: 1px solid #fff;
  border-radius: 22px;
}

.draw-dock {
  position: fixed;
  z-index: 80;
  right: 50%;
  bottom: 0;
  width: min(100%, 430px);
  height: var(--draw-dock-height);
  padding-bottom: env(safe-area-inset-bottom);
  overflow: visible;
  background-image: var(--draw-dock-bg);
  background-repeat: no-repeat;
  background-position: top center;
  background-size: 100% var(--draw-dock-base-height);
  pointer-events: none;
  transform: translateX(50%);
}

.draw-dock__ended {
  position: absolute;
  z-index: 8;
  top: 34px;
  left: 50%;
  min-width: 132px;
  padding: 7px 18px;
  color: #a4495d;
  font-size: 14px;
  font-weight: 700;
  text-align: center;
  background: rgba(255, 246, 248, 0.94);
  border: 1px solid rgba(190, 92, 113, 0.32);
  border-radius: 999px;
  box-shadow: 0 5px 14px rgba(100, 35, 50, 0.12);
  transform: translateX(-50%);
  pointer-events: none;
}

.draw-dock--night .draw-dock__ended {
  color: #f2dff7;
  background: rgba(48, 28, 68, 0.94);
  border-color: rgba(225, 190, 236, 0.32);
  box-shadow: 0 5px 14px rgba(18, 8, 30, 0.28);
}

.draw-dock__record {
  position: absolute;
  z-index: 7;
  top: clamp(-2px, 0.8vw, 3px);
  left: calc(clamp(14px, 4.7vw, 20px) - 15px);
  display: grid;
  min-width: 84px;
  min-height: 32px;
  padding: 0;
  place-items: center;
  background: transparent;
  border: 0;
  cursor: pointer;
  pointer-events: auto;
  transition: filter 180ms ease, transform 180ms ease;
}

.draw-dock__record img {
  width: 77px;
  height: 16px;
  object-fit: contain;
  filter: drop-shadow(0 2px 4px rgba(70, 34, 94, 0.42));
}

.draw-dock__record:active {
  transform: scale(0.96);
  filter: brightness(1.08);
}

.draw-dock__buttons {
  position: absolute;
  z-index: 1;
  display: grid;
  top: clamp(22px, 6.4vw, 27px);
  right: 0;
  left: 0;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  align-items: start;
  gap: 0;
}

.draw-dock__buttons--compact {
  right: clamp(30px, 10.5vw, 45px);
  left: clamp(30px, 10.5vw, 45px);
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: clamp(8px, 2.2vw, 11px);
}

.draw-dock__buttons button {
  position: relative;
  display: grid;
  min-width: 44px;
  min-height: clamp(68px, 18.9vw, 82px);
  padding: 0;
  place-items: center;
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  pointer-events: auto;
  transition: opacity 180ms ease, filter 180ms ease, transform 180ms ease;
}

.draw-dock__button-image {
  width: 100%;
  aspect-ratio: 192 / 142;
  object-fit: contain;
  pointer-events: none;
  filter: drop-shadow(0 3px 6px rgba(55, 19, 61, 0.28));
}

.draw-flight-animation {
  position: fixed;
  z-index: 1750;
  top: 0;
  left: 0;
  width: 750px;
  max-width: none;
  height: 1820px;
  max-height: none;
  pointer-events: none;
  transform-origin: 0 0;
  will-change: transform;
}

.draw-dock__button--tbp:not(:disabled) .draw-dock__button-image {
  transform-origin: center;
  animation: tbp-breathe 1.4s ease-in-out infinite;
}

.draw-dock__buttons button:active {
  transform: translateY(1px) scale(0.96);
  filter: brightness(1.06);
}

.draw-dock__buttons button:disabled {
  cursor: not-allowed;
  opacity: 0.52;
  filter: grayscale(0.22) saturate(0.8);
}

.draw-dock__settings {
  position: absolute;
  z-index: 1;
  top: calc(var(--draw-dock-base-height) - 52px);
  right: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
  padding: 0 clamp(34px, 10.5vw, 48px);
}

.draw-dock__settings label {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 8px;
  color: rgba(255, 255, 255, 0.92);
  font-family: 'Source Han Serif CN', 'Songti SC', serif;
  font-size: clamp(11px, 3.4vw, 14px);
  font-weight: 700;
  text-shadow: 0 2px 5px rgba(70, 34, 94, 0.72);
  cursor: pointer;
  pointer-events: auto;
}

.draw-dock__settings input { position: absolute; width: 1px; height: 1px; opacity: 0; }

.draw-dock__settings label > span {
  display: grid;
  flex: 0 0 auto;
  width: 22px;
  height: 22px;
  place-items: center;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.96);
  border-radius: 5px;
  box-shadow: 0 3px 7px rgba(73, 44, 103, 0.48);
}

.draw-dock__settings input:checked + span::after {
  content: '';
  width: 12px;
  height: 7px;
  border: solid #8d5d99;
  border-width: 0 0 4px 4px;
  transform: rotate(-45deg) translateY(-1px);
}

.draw-dock--round-complete .draw-dock__buttons,
.draw-dock--round-complete .draw-dock__settings {
  pointer-events: none;
  opacity: 0.3;
  filter: grayscale(0.45) brightness(0.72);
}

.draw-dock__new-round {
  position: absolute;
  inset: 0 0 auto;
  z-index: 6;
  width: 100%;
  height: var(--draw-dock-base-height);
  pointer-events: none;
}

.draw-dock__new-round img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: fill;
  pointer-events: none;
  filter: drop-shadow(0 3px 8px rgba(48, 8, 64, 0.56));
}

.draw-dock__new-round-action {
  position: absolute;
  z-index: 1;
  top: 36%;
  left: 29%;
  width: 42%;
  height: 34%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  pointer-events: auto;
}

.draw-dock__new-round-action:disabled {
  cursor: wait;
  opacity: 0.62;
}

.new-round-notice,
.gold-insufficient-notice {
  position: fixed;
  top: 50%;
  left: 50%;
  z-index: 3000;
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

@keyframes marquee-once {
  from { transform: translateX(100%); }
  to { transform: translateX(-100%); }
}
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes reward-pulse { 50% { transform: translateY(-2px); filter: brightness(1.12); } }
@keyframes tbp-breathe {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.08); }
}

@media (max-width: 350px) {
  .ring-card { width: 104px; height: 104px; }
  .reward-item__visual { width: 48px; height: 50px; }
  .reward-item__icon { top: 7px; width: 34px; height: 34px; }
  .reward-item__image { top: 2px; width: 44px; height: 44px; }
  .reward-item__picture { top: 50%; width: 132px; height: 132px; }
  .reward-item__name { min-height: 22px; font-size: 8px; }
  .reward-item__threshold { width: 48px; }
  .draw-dock__buttons { top: 20px; }
  .draw-dock__buttons--compact { right: 28px; left: 28px; gap: 6px; }
  .draw-dock__settings { padding-inline: 28px; }
  .draw-dock__settings label { gap: 6px; font-size: 11px; }
  .draw-dock__settings label > span { width: 20px; height: 20px; }
}

@media (orientation: landscape) and (max-height: 540px) {
  .activity-scene { width: min(100%, 560px); }
  .hero-space { height: 72px; }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
