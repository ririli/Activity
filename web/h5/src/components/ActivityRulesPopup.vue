<template>
  <Teleport to="body">
    <div class="rules-overlay" @click.self="emit('close')">
      <section
        class="rules-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="activity-rules-title"
        @keydown.esc="emit('close')"
      >
        <img
          class="rules-frame"
          src="/image/activity/rules/rules-popup-frame.png"
          alt=""
          aria-hidden="true"
          draggable="false"
        >
        <h2 id="activity-rules-title" class="sr-only">活动说明</h2>
        <div class="rules-copy" data-testid="activity-rules-scroll">
          <section>
            <h3>一、活动时间</h3>
            <p>活动上线至2026年4月14日23:59:59</p>
          </section>

          <section>
            <h3>二、活动玩法</h3>
            <ol>
              <li>活动期间可选择白昼许愿或星夜许愿两种许愿方式，<strong>星夜许愿必得稀有及以上等级奖励。</strong></li>
              <li>每次许愿必得奖励，有概率获得真爱无敌戒指。</li>
              <li>每次许愿还有概率点亮花盒中的花朵，<strong>白昼许愿每进行1次许愿有概率点亮花盒中的1朵花，星夜许愿每进行1次许愿有概率点亮花盒中的多朵花。</strong></li>
              <li>每次点亮花盒中的6朵花时，可获得1次开启戒指宝箱召唤戒指的机会，必定获得真爱无敌戒指自选权、闪闪心蝶戒指或烟花之恋戒指之一，通过该渠道获得的戒指不会重复获得，每开启3次宝箱后状态重置。</li>
              <li>真爱无敌戒指共有三种形态：经典款、真爱无敌铭文款、爱你万年铭文款，在本次活动中通过许愿或集花朵召唤戒指获得真爱无敌戒指选择权后可在以上三种形态中自选1枚领取。</li>
              <li>每点亮18朵花，召唤3次戒指宝箱中的戒指视为完成一轮，之后轮次状态重置。</li>
              <li>每一轮内累计点亮的花朵数达标可领取对应的爱意翩跹戒指、全新真爱降临主页特效或月亮游记戒指等丰厚奖励。</li>
            </ol>
          </section>

          <section>
            <h3>三、新戒福利</h3>
            <p>活动期间，使用真爱无敌戒指任意一款挂戒，CP双方可获得66天真爱无敌称号，15天挚爱一生戒指墙、15天挚爱一生戒指框，重复挂戒奖励天数可叠加。</p>
          </section>

          <section>
            <h3>四、活动须知</h3>
            <ol>
              <li>活动中获得的礼物卡有效期为30天，需尽快使用。</li>
              <li>本活动涉及的相关时间节点皆为：东八区（GMT+8）。</li>
              <li>本活动5折花瓣礼包限购1次，请按正常流程购买，勿用非正常手段多次支付。若出现重复支付，超出购买次数限制的订单将按 1 元 = 100 金币的比例发放对应金币。为保障您的购买体验，避免困扰，后续购买礼包时请注意并遵守购买上限，勿超限制支付。</li>
              <li>本次活动禁止未成年人参加，请用户理性消费、适度参与。用户通过本次活动获得的所有虚拟道具仅限于平台内使用，不得反向兑换为法定货币、现金或其他任何具有交换价值的物品，不得用于任何形式的盈利活动。用户通过本次活动获得的所有虚拟物品均不得被转让、赠送，仅供用户自身使用。若用户出现违反会玩平台规则的行为，我们将采取包括但不限于扣除收益、永久封号等处罚措施。</li>
              <li><span class="rules-probability">点击查看概率公示。</span></li>
            </ol>
          </section>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const emit = defineEmits<{ close: [] }>()
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

.rules-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 10px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.36);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}

.rules-panel {
  position: relative;
  width: min(96vw, 420px, calc((100dvh - 24px) * 0.913408));
  aspect-ratio: 654 / 716;
  transform-origin: center;
  animation: rules-panel-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.rules-frame {
  position: absolute;
  inset: 0;
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

.rules-copy {
  position: absolute;
  top: 18.5%;
  right: 8.5%;
  bottom: 8.5%;
  left: 10.5%;
  z-index: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 2px 7px 16px 1px;
  color: #fff;
  font-family: 'Songti SC', 'STSong', 'SimSun', serif;
  font-size: 14px;
  line-height: 1.65;
  text-shadow: 0 1px 2px rgba(39, 39, 103, 0.55);
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.rules-copy::-webkit-scrollbar {
  display: none;
}

.rules-copy section + section {
  margin-top: 20px;
}

.rules-copy h3 {
  margin-bottom: 7px;
  color: #fff;
  font-size: 16px;
  font-weight: 700;
}

.rules-copy p,
.rules-copy ol {
  margin: 0;
}

.rules-copy ol {
  padding-left: 0;
  list-style: none;
  counter-reset: activity-rule;
}

.rules-copy li {
  position: relative;
  padding-left: 1.8em;
  counter-increment: activity-rule;
}

.rules-copy li::before {
  position: absolute;
  top: 0;
  left: 0;
  content: counter(activity-rule) '、';
}

.rules-copy li + li {
  margin-top: 7px;
}

.rules-copy strong,
.rules-probability {
  color: #fff0a4;
  font-weight: 700;
}

@keyframes rules-panel-expand {
  0% {
    opacity: 0;
    transform: scale(0.3);
  }

  72% {
    opacity: 1;
    transform: scale(1.03);
  }

  100% {
    opacity: 1;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .rules-panel {
    animation: none;
  }
}

@media (max-width: 360px) {
  .rules-copy {
    font-size: 13px;
    line-height: 1.6;
  }

  .rules-copy h3 {
    font-size: 15px;
  }
}
</style>
