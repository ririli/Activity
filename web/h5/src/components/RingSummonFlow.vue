<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="summon-overlay"
      @keydown.esc.stop.prevent
    >
      <section
        ref="dialogRef"
        class="summon-dialog"
        :class="{
          'summon-dialog--ready': status?.flow_state === 'ready',
          'summon-dialog--result': status?.flow_state === 'result',
          'summon-dialog--choice': status?.flow_state === 'choice',
          'popup-center-expand': loadState === 'loading' || loadState === 'error',
        }"
        role="dialog"
        aria-modal="true"
        aria-labelledby="summon-title"
        tabindex="-1"
      >
        <div v-if="loadState === 'loading'" class="summon-state summon-state--compact" role="status">
          <span class="summon-spinner" aria-hidden="true"></span>
          <h2 id="summon-title">同步召唤进度</h2>
          <p>正在恢复尚未完成的戒指召唤。</p>
        </div>

        <div v-else-if="loadState === 'error'" class="summon-state summon-state--compact" role="alert">
          <span class="summon-mark">!</span>
          <h2 id="summon-title">召唤进度同步失败</h2>
          <p>需要先恢复宝箱状态，才能继续本次活动。</p>
          <button class="summon-button" type="button" @click="$emit('retry')">重新同步</button>
        </div>

        <div v-else-if="status?.flow_state === 'ready'" class="summon-ready popup-center-expand">
          <h2 id="summon-title" class="sr-only">六朵花已全部点亮</h2>
          <img
            class="summon-ready__frame"
            src="/image/activity/ring-summon/ready-frame.png"
            alt=""
            aria-hidden="true"
            draggable="false"
          >
          <p class="summon-ready__copy">
            恭喜您已点亮6朵花！<br>
            已点亮全部花朵，<br>
            获得开启第{{ status.next_chance_index }}次戒指宝箱的机会
          </p>
          <button
            class="summon-ready__button"
            type="button"
            :disabled="opening"
            aria-label="召唤戒指"
            @click="$emit('summon')"
          >
            <img src="/image/activity/ring-summon/summon-button.png" alt="" aria-hidden="true" draggable="false">
            <span v-if="opening">召唤中…</span>
          </button>
        </div>

        <div v-else-if="status?.flow_state === 'result' && status.pending_reward" class="summon-result popup-center-expand">
          <h2 id="summon-title" class="sr-only">戒指已回应召唤</h2>
          <img class="summon-result__frame" src="/image/activity/ring-summon/result/frame.png" alt="" aria-hidden="true" draggable="false">
          <div class="summon-result__reward" aria-label="本次召唤结果">
            <img class="summon-result__card" src="/image/activity/ring-summon/result/reward-card.png" alt="" aria-hidden="true" draggable="false">
            <picture class="summon-result__ring">
              <source v-if="resultRing?.animatedImage" :srcset="resultRing.animatedImage" type="image/svg+xml">
              <img :src="resultRing?.image" :alt="status.pending_reward.item_name" draggable="false">
            </picture>
            <strong>{{ status.pending_reward.item_name }}<span>×1</span></strong>
          </div>
          <button
            class="summon-result__confirm"
            type="button"
            :disabled="confirming"
            :aria-label="confirming ? '正在确认收下' : '确认收下'"
            @click="$emit('confirm', 0)"
          >
            <img src="/image/activity/ring-summon/result/confirm.png" alt="" aria-hidden="true" draggable="false">
          </button>
        </div>

        <div
          v-else-if="status?.flow_state === 'choice' && status.pending_reward"
          :key="choiceStep"
          class="true-love-choice popup-center-expand"
          :class="{ 'true-love-choice--confirm': choiceStep === 'confirm' }"
        >
          <template v-if="choiceStep === 'select'">
            <h2 id="summon-title" class="sr-only">选择一枚戒指</h2>
            <img class="true-love-choice__frame" src="/image/activity/true-love-choice/select-frame.png" alt="" aria-hidden="true" draggable="false">
            <p class="true-love-choice__intro">恭喜您获得真爱无敌戒指选择权<br>请您在以下三款样式中选择一枚领取</p>
            <div class="true-love-choice__list" role="radiogroup" aria-label="真爱无敌戒指三选一">
              <button
                v-for="item in choiceItems"
                :key="item.id"
                class="true-love-choice__item"
                type="button"
                role="radio"
                :aria-checked="selectedItemId === item.id"
                :aria-label="`${item.name}，道具ID ${item.id}`"
                @click="selectedItemId = item.id"
              >
                <img class="true-love-choice__card" :src="item.card" :alt="item.name">
                <img v-if="item.id === 1207751" class="true-love-choice__new" src="/image/activity/true-love-choice/new.png" alt="" aria-hidden="true">
                <img
                  class="true-love-choice__radio"
                  :src="selectedItemId === item.id ? '/image/activity/true-love-choice/radio-selected.png' : '/image/activity/true-love-choice/radio-empty.png'"
                  alt=""
                  aria-hidden="true"
                >
              </button>
            </div>
            <button
              class="true-love-choice__select-confirm"
              type="button"
              :disabled="selectedItemId === null"
              aria-label="确定所选戒指"
              @click="selectedItemId !== null && (choiceStep = 'confirm')"
            >
              <img src="/image/activity/true-love-choice/select-confirm.png" alt="" aria-hidden="true">
            </button>
          </template>

          <template v-else>
            <h2 id="summon-title" class="sr-only">确认选择戒指</h2>
            <img class="true-love-choice__frame" src="/image/activity/true-love-choice/confirm-frame.png" alt="" aria-hidden="true" draggable="false">
            <p class="true-love-choice__confirm-copy">确认选择「{{ selectedChoice?.name }}」吗？<br>*请谨慎选择哦！</p>
            <div class="true-love-choice__confirm-actions">
              <button type="button" aria-label="再想想" :disabled="confirming" @click="choiceStep = 'select'">
                <img src="/image/activity/true-love-choice/rethink.png" alt="" aria-hidden="true">
              </button>
              <button type="button" aria-label="确认选择戒指" :disabled="confirming" @click="selectedItemId !== null && $emit('confirm', selectedItemId)">
                <img src="/image/activity/true-love-choice/final-confirm.png" alt="" aria-hidden="true">
              </button>
            </div>
          </template>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { BoxStatusResponse } from '@/api/treasureBox'
import { getRingCatalogItem } from '@/constants/ringCatalog'
import { playActivityEffect } from '@/services/activityAudio'

const props = defineProps<{
  status: BoxStatusResponse | null
  loadState: 'idle' | 'loading' | 'ready' | 'error'
  opening: boolean
  confirming: boolean
}>()

defineEmits<{
  retry: []
  summon: []
  confirm: [selectedItemId: number]
}>()

const dialogRef = ref<HTMLElement | null>(null)
const selectedItemId = ref<number | null>(null)
const choiceStep = ref<'select' | 'confirm'>('select')

const visible = computed(() => {
  if (props.loadState === 'loading' || props.loadState === 'error') return true
  if (props.loadState !== 'ready' || !props.status) return false

  if (props.status.flow_state === 'ready') {
    return typeof props.status.next_chance_index === 'number'
  }

  if (props.status.flow_state === 'result' || props.status.flow_state === 'choice') {
    return Boolean(props.status.pending_reward)
  }

  return false
})

const choiceItems = computed(() => (props.status?.pending_reward?.choice_ids ?? []).map((id) => ({
  id,
  name: getRingCatalogItem(id)?.name ?? '真爱无敌戒指',
  shortName: getRingCatalogItem(id)?.shortName ?? '真爱无敌',
  card: choiceCardAssets[id] ?? getRingCatalogItem(id)?.image,
})))

const choiceCardAssets: Record<number, string> = {
  1207751: '/image/activity/true-love-choice/classic.png',
  1207752: '/image/activity/true-love-choice/true-love-inscription.png',
  1207753: '/image/activity/true-love-choice/love-forever-inscription.png',
}

const selectedChoice = computed(() => choiceItems.value.find((item) => item.id === selectedItemId.value))
const resultRing = computed(() => {
  const itemId = props.status?.flow_state === 'result' ? props.status.pending_reward?.item_id : null
  return itemId ? getRingCatalogItem(itemId) : undefined
})

watch(() => props.status?.pending_reward?.chance_index, () => {
  selectedItemId.value = null
  choiceStep.value = 'select'
})

watch(
  () => props.loadState === 'ready'
    && props.status?.flow_state === 'ready'
    && typeof props.status.next_chance_index === 'number'
      ? props.status.next_chance_index
      : null,
  (chanceIndex, previousChanceIndex) => {
    if (chanceIndex !== null && chanceIndex !== previousChanceIndex) {
      playActivityEffect('ringBoxReady')
    }
  },
  { immediate: true },
)

watch(visible, async (isVisible) => {
  if (!isVisible) return
  await nextTick()
  dialogRef.value?.focus()
}, { immediate: true })
</script>

<style scoped>
.summon-overlay {
  position: fixed;
  inset: 0;
  z-index: 1600;
  display: grid;
  padding: 20px;
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.summon-dialog {
  position: relative;
  width: min(100%, 370px);
  max-height: min(88dvh, 700px);
  overflow: auto;
  color: #59243e;
  background: #fff8f7;
  border: 1px solid rgba(255, 228, 215, 0.92);
  border-radius: 8px;
  box-shadow: 0 24px 64px rgba(12, 5, 21, 0.56), inset 0 0 0 5px rgba(255, 255, 255, 0.72);
  outline: 0;
}

.summon-dialog::before {
  position: absolute;
  inset: 8px;
  pointer-events: none;
  content: '';
  border: 1px solid rgba(206, 111, 132, 0.2);
  border-radius: 4px;
}

.summon-dialog--ready {
  width: min(94vw, 470px);
  max-height: 94dvh;
  overflow: visible;
  color: #fff;
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.summon-dialog--ready::before { display: none; }

.summon-dialog--result,
.summon-dialog--choice {
  width: min(94vw, 470px);
  max-height: 94dvh;
  overflow: visible;
  color: #fff;
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.summon-dialog--result::before,
.summon-dialog--choice::before { display: none; }

.summon-ready {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
}

.summon-ready__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.summon-ready__copy {
  position: absolute;
  z-index: 1;
  top: 22.5%;
  right: 15%;
  left: 15%;
  margin: 0;
  color: #fff;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: clamp(14px, 4vw, 20px);
  font-weight: 700;
  line-height: 1.45;
  text-align: center;
  text-shadow: 0 2px 5px rgba(55, 25, 105, 0.72);
}

.summon-ready__button {
  position: absolute;
  z-index: 2;
  right: 24.3%;
  bottom: 3.8%;
  width: 51.4%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 160ms ease, transform 160ms ease, opacity 160ms ease;
}

.summon-ready__button img {
  display: block;
  width: 100%;
  height: auto;
}

.summon-ready__button span {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  color: #8d4d38;
  font-size: 14px;
  font-weight: 700;
  text-shadow: 0 1px rgba(255, 255, 255, 0.7);
}

.summon-ready__button:active:not(:disabled) { transform: scale(0.97); filter: brightness(1.06); }
.summon-ready__button:disabled { cursor: wait; opacity: 0.68; }

.summon-result {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
}

.summon-result__frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.summon-result__reward {
  position: absolute;
  z-index: 1;
  top: 28%;
  right: 31%;
  left: 31%;
  aspect-ratio: 1;
}

.summon-result__card {
  position: absolute;
  inset: 8%;
  width: 84%;
  height: 84%;
  object-fit: contain;
}

.summon-result__ring {
  position: absolute;
  z-index: 1;
  inset: -8%;
  width: 116%;
  height: 116%;
  filter: drop-shadow(0 7px 10px rgba(38, 16, 91, 0.42));
}

.summon-result__ring img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.summon-result__reward strong {
  position: absolute;
  top: calc(100% + 4px);
  left: 50%;
  width: 190%;
  color: #fff;
  font-family: 'Songti SC', 'STSong', serif;
  font-size: clamp(13px, 3.8vw, 18px);
  line-height: 1.3;
  text-align: center;
  text-shadow: 0 2px 4px rgba(43, 20, 85, 0.85);
  transform: translateX(-50%);
}

.summon-result__reward strong span {
  margin-left: 3px;
  font-family: Arial, sans-serif;
}

.summon-result__confirm {
  position: absolute;
  z-index: 2;
  right: 24.3%;
  bottom: 10.2%;
  width: 51.4%;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: filter 160ms ease, opacity 160ms ease, transform 160ms ease;
}

.summon-result__confirm img {
  display: block;
  width: 100%;
  height: auto;
}

.summon-result__confirm:active:not(:disabled) { transform: scale(0.97); filter: brightness(1.06); }
.summon-result__confirm:disabled { cursor: wait; opacity: 0.58; }

.true-love-choice { position: relative; width: 100%; aspect-ratio: 1; }
.true-love-choice--confirm { aspect-ratio: 654 / 480; }
.true-love-choice__frame { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; pointer-events: none; }
.true-love-choice__intro { position: absolute; z-index: 1; top: 21%; right: 12%; left: 12%; margin: 0; color: #fff; font-family: 'Songti SC', 'STSong', serif; font-size: clamp(14px, 4vw, 20px); font-weight: 700; line-height: 1.55; text-align: center; text-shadow: 0 2px 5px rgba(55, 25, 105, 0.72); }
.true-love-choice__list { position: absolute; z-index: 1; top: 36%; right: 9%; left: 9%; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); align-items: start; gap: 2%; }
.true-love-choice__item { position: relative; min-width: 0; padding: 0; background: transparent; border: 0; cursor: pointer; }
.true-love-choice__card { display: block; width: 100%; height: auto; }
.true-love-choice__new { position: absolute; top: 2%; left: -2%; width: 50%; height: auto; pointer-events: none; }
.true-love-choice__radio { position: absolute; top: 1%; right: -3%; width: 31%; height: auto; pointer-events: none; }
.true-love-choice__select-confirm { position: absolute; z-index: 2; right: 24.3%; bottom: 10.2%; width: 51.4%; padding: 0; background: transparent; border: 0; cursor: pointer; }
.true-love-choice__select-confirm img { display: block; width: 100%; height: auto; }
.true-love-choice__select-confirm:disabled { cursor: default; opacity: 0.5; filter: grayscale(0.35); }
.true-love-choice__confirm-copy { position: absolute; z-index: 1; top: 35%; right: 12%; left: 12%; margin: 0; color: #fff; font-family: 'Songti SC', 'STSong', serif; font-size: clamp(16px, 4.5vw, 22px); font-weight: 700; line-height: 1.55; text-align: center; text-shadow: 0 2px 5px rgba(55, 25, 105, 0.72); transform: translateY(-50%); }
.true-love-choice__confirm-actions { position: absolute; z-index: 2; right: 12%; bottom: 12%; left: 12%; display: grid; grid-template-columns: 1fr 1fr; gap: 7%; }
.true-love-choice__confirm-actions button { padding: 0; background: transparent; border: 0; cursor: pointer; }
.true-love-choice__confirm-actions img { display: block; width: 100%; height: auto; }
.true-love-choice__confirm-actions button:active:not(:disabled), .true-love-choice__select-confirm:active:not(:disabled) { transform: scale(0.97); }
.true-love-choice__confirm-actions button:disabled { cursor: wait; opacity: 0.58; }

.summon-state {
  position: relative;
  z-index: 1;
  display: flex;
  min-height: 480px;
  flex-direction: column;
  align-items: center;
  padding: 30px 22px 24px;
  text-align: center;
}

.summon-state--compact {
  min-height: 300px;
  justify-content: center;
}

.summon-state--choice {
  min-height: 510px;
}

.summon-kicker {
  color: #b66379;
  font-size: 12px;
  font-weight: 700;
}

.summon-state h2 {
  margin: 6px 0 0;
  color: #7b2949;
  font-family: 'Source Han Serif CN', 'Songti SC', serif;
  font-size: 24px;
  line-height: 1.25;
}

.summon-state p {
  max-width: 280px;
  margin: 10px auto 18px;
  color: #8b5b6b;
  font-size: 13px;
  line-height: 1.6;
}

.summon-button {
  width: min(100%, 240px);
  min-height: 48px;
  padding: 0 18px;
  color: #7e3148;
  font-size: 15px;
  font-weight: 800;
  background: #ffe2dc;
  border: 1px solid #e9a8a4;
  border-radius: 8px;
  cursor: pointer;
  transition: filter 160ms ease, transform 160ms ease, opacity 160ms ease;
}

.summon-button--primary {
  color: #6f273c;
  background: linear-gradient(180deg, #ffe7b4, #f3b66f);
  border-color: #e6a45f;
  box-shadow: 0 5px 14px rgba(151, 69, 73, 0.2), inset 0 1px rgba(255, 255, 255, 0.8);
}

.summon-button:active:not(:disabled) { transform: translateY(1px) scale(0.98); }
.summon-button:disabled { cursor: wait; opacity: 0.55; }

.summon-reward {
  display: flex;
  width: min(100%, 246px);
  min-height: 258px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin: 24px auto;
  background: #fff0ee;
  border: 1px solid #efc6c4;
  border-radius: 8px;
}

.summon-reward__icon {
  display: block;
  width: 112px;
  height: 112px;
  margin-bottom: 16px;
  filter: drop-shadow(0 8px 10px rgba(126, 62, 103, 0.2));
}

.summon-reward strong { color: #7e314d; font-size: 17px; }
.summon-reward small { margin-top: 6px; color: #a47a87; font-size: 11px; }

.summon-choice-list {
  display: grid;
  width: 100%;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 24px 0;
}

.summon-choice {
  display: flex;
  min-width: 0;
  min-height: 208px;
  flex-direction: column;
  align-items: center;
  padding: 10px 5px;
  color: #7e314d;
  background: #fff0ee;
  border: 1px solid #ebc4c4;
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease;
}

.summon-choice img,
.summon-choice__fallback {
  display: block;
  width: 92px;
  max-width: 100%;
  height: 112px;
  object-fit: contain;
}

.summon-choice__fallback { padding: 10px; }
.summon-choice strong { min-height: 40px; margin-top: 5px; font-size: 12px; line-height: 1.4; }
.summon-choice small { margin-top: auto; color: #a07885; font-size: 10px; }

.summon-choice--selected {
  border-color: #c65f78;
  box-shadow: 0 0 0 2px rgba(198, 95, 120, 0.2), 0 8px 18px rgba(118, 48, 76, 0.14);
  transform: translateY(-2px);
}

.summon-spinner {
  width: 42px;
  height: 42px;
  margin-bottom: 18px;
  border: 4px solid #f0c7cc;
  border-top-color: #a94867;
  border-radius: 50%;
  animation: summon-spin 760ms linear infinite;
}

.summon-mark {
  display: grid;
  width: 42px;
  height: 42px;
  margin-bottom: 16px;
  place-items: center;
  color: #fff;
  font-size: 25px;
  font-weight: 800;
  background: #a94867;
  border-radius: 50%;
}

@keyframes summon-spin { to { transform: rotate(360deg); } }

@media (max-width: 350px) {
  .summon-state { padding-inline: 16px; }
  .summon-choice-list { gap: 5px; }
  .summon-choice { min-height: 192px; }
  .summon-choice img,
  .summon-choice__fallback { height: 96px; }
}

@media (prefers-reduced-motion: reduce) {
  .summon-spinner { animation-duration: 0.01ms; }
  .summon-button,
  .summon-ready__button,
  .summon-result__confirm,
  .summon-choice { transition-duration: 0.01ms; }
}
</style>
