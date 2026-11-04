<template>
  <div class="falling-petals" aria-hidden="true">
    <img
      v-for="petal in petals"
      :key="petal.id"
      class="falling-petals__item"
      :class="`falling-petals__item--${petal.path}`"
      :src="petal.src"
      alt=""
      draggable="false"
      :style="petal.style"
    >
  </div>
</template>

<script setup lang="ts">
const petals = [
  { id: 1, src: '/image/activity/effects/falling-petal-1.png', path: 'right', style: { '--left': '4%', '--size': '21px', '--duration': '7.8s', '--delay': '-1.8s', '--drift': '48px', '--sway': '-14px', '--turn': '310deg', '--petal-opacity': '0.86' } },
  { id: 2, src: '/image/activity/effects/falling-petal-2.png', path: 'left', style: { '--left': '13%', '--size': '29px', '--duration': '9.2s', '--delay': '-6.4s', '--drift': '-36px', '--sway': '16px', '--turn': '-250deg', '--petal-opacity': '0.72' } },
  { id: 3, src: '/image/activity/effects/falling-petal-1.png', path: 'left', style: { '--left': '24%', '--size': '17px', '--duration': '6.9s', '--delay': '-4.7s', '--drift': '-42px', '--sway': '13px', '--turn': '-340deg', '--petal-opacity': '0.8' } },
  { id: 4, src: '/image/activity/effects/falling-petal-2.png', path: 'right', style: { '--left': '34%', '--size': '25px', '--duration': '8.5s', '--delay': '-0.9s', '--drift': '54px', '--sway': '-19px', '--turn': '270deg', '--petal-opacity': '0.78' } },
  { id: 5, src: '/image/activity/effects/falling-petal-1.png', path: 'right', style: { '--left': '45%', '--size': '19px', '--duration': '7.3s', '--delay': '-5.5s', '--drift': '39px', '--sway': '-11px', '--turn': '360deg', '--petal-opacity': '0.9' } },
  { id: 6, src: '/image/activity/effects/falling-petal-2.png', path: 'left', style: { '--left': '55%', '--size': '31px', '--duration': '10.1s', '--delay': '-8.2s', '--drift': '-55px', '--sway': '22px', '--turn': '-290deg', '--petal-opacity': '0.66' } },
  { id: 7, src: '/image/activity/effects/falling-petal-1.png', path: 'left', style: { '--left': '64%', '--size': '16px', '--duration': '6.6s', '--delay': '-2.9s', '--drift': '-31px', '--sway': '12px', '--turn': '-380deg', '--petal-opacity': '0.84' } },
  { id: 8, src: '/image/activity/effects/falling-petal-2.png', path: 'right', style: { '--left': '72%', '--size': '27px', '--duration': '8.9s', '--delay': '-7.1s', '--drift': '43px', '--sway': '-17px', '--turn': '240deg', '--petal-opacity': '0.75' } },
  { id: 9, src: '/image/activity/effects/falling-petal-1.png', path: 'right', style: { '--left': '81%', '--size': '20px', '--duration': '7.6s', '--delay': '-3.6s', '--drift': '35px', '--sway': '-10px', '--turn': '330deg', '--petal-opacity': '0.88' } },
  { id: 10, src: '/image/activity/effects/falling-petal-2.png', path: 'left', style: { '--left': '90%', '--size': '24px', '--duration': '9.6s', '--delay': '-5.9s', '--drift': '-47px', '--sway': '19px', '--turn': '-260deg', '--petal-opacity': '0.7' } },
  { id: 11, src: '/image/activity/effects/falling-petal-1.png', path: 'right', style: { '--left': '18%', '--size': '15px', '--duration': '8.1s', '--delay': '-7.7s', '--drift': '62px', '--sway': '-21px', '--turn': '410deg', '--petal-opacity': '0.65' } },
  { id: 12, src: '/image/activity/effects/falling-petal-2.png', path: 'left', style: { '--left': '77%', '--size': '22px', '--duration': '7.1s', '--delay': '-1.2s', '--drift': '-38px', '--sway': '14px', '--turn': '-320deg', '--petal-opacity': '0.82' } },
] as const
</script>

<style scoped>
.falling-petals {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  z-index: 3;
  height: 33.333%;
  overflow: hidden;
  pointer-events: none;
  contain: layout paint;
}

.falling-petals__item {
  position: absolute;
  top: -48px;
  left: var(--left);
  width: var(--size);
  height: auto;
  opacity: 0;
  object-fit: contain;
  filter: drop-shadow(0 2px 2px rgba(87, 9, 23, 0.16));
  will-change: transform, opacity;
  animation-duration: var(--duration);
  animation-delay: var(--delay);
  animation-timing-function: linear;
  animation-iteration-count: infinite;
}

.falling-petals__item--right {
  animation-name: petal-fall-right;
}

.falling-petals__item--left {
  animation-name: petal-fall-left;
}

@keyframes petal-fall-right {
  0% {
    opacity: var(--petal-opacity);
    transform: translate3d(0, -30px, 0) rotate(-18deg) scale(0.82);
  }
  34% {
    opacity: var(--petal-opacity);
    transform: translate3d(var(--sway), 11dvh, 0) rotate(82deg) scale(1);
  }
  68% {
    opacity: var(--petal-opacity);
    transform: translate3d(var(--sway), 23dvh, 0) rotate(205deg) scale(0.94);
  }
  82% {
    opacity: 0.62;
  }
  92% {
    opacity: 0.26;
  }
  100% {
    opacity: 0;
    transform: translate3d(var(--drift), 36dvh, 0) rotate(var(--turn)) scale(0.78);
  }
}

@keyframes petal-fall-left {
  0% {
    opacity: var(--petal-opacity);
    transform: translate3d(0, -34px, 0) rotate(22deg) scale(0.78);
  }
  38% {
    opacity: var(--petal-opacity);
    transform: translate3d(var(--sway), 12dvh, 0) rotate(-95deg) scale(0.98);
  }
  70% {
    opacity: var(--petal-opacity);
    transform: translate3d(var(--sway), 24dvh, 0) rotate(-210deg) scale(0.9);
  }
  83% {
    opacity: 0.6;
  }
  93% {
    opacity: 0.24;
  }
  100% {
    opacity: 0;
    transform: translate3d(var(--drift), 36dvh, 0) rotate(var(--turn)) scale(0.74);
  }
}

@media (prefers-reduced-motion: reduce) {
  .falling-petals {
    display: none;
  }
}
</style>
