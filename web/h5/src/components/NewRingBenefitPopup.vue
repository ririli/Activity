<template>
  <Teleport to="body">
    <div class="benefit-overlay" @click.self="emit('close')">
      <section
        ref="dialogRef"
        class="benefit-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="benefit-title"
        tabindex="-1"
        @keydown.esc.stop.prevent="emit('close')"
      >
        <h2 id="benefit-title" class="sr-only">新戒福利</h2>
        <div class="benefit-scroll" data-testid="new-ring-benefit-scroll">
          <div class="benefit-artwork">
            <img
              class="benefit-artwork__background"
              src="/image/activity/benefit/new-ring-benefit-popup.png"
              alt="真爱无敌戒指新戒福利详情"
              draggable="false"
            >
            <picture class="benefit-artwork__ring" aria-hidden="true">
              <source srcset="/image/activity/rings/ring-1207751-true-love-classic.svg" type="image/svg+xml">
              <img
                src="/image/activity/rings/ring-1207751-true-love-classic.png"
                alt=""
                draggable="false"
              >
            </picture>
          </div>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from 'vue'

const emit = defineEmits<{ close: [] }>()
const dialogRef = ref<HTMLElement | null>(null)

onMounted(async () => {
  await nextTick()
  dialogRef.value?.focus()
})
</script>

<style scoped>
.benefit-overlay {
  position: fixed;
  inset: 0;
  z-index: 1600;
  display: grid;
  padding: max(12px, env(safe-area-inset-top)) 10px max(12px, env(safe-area-inset-bottom));
  place-items: center;
  background: rgba(10, 6, 30, 0.72);
  backdrop-filter: blur(3px);
  -webkit-backdrop-filter: blur(3px);
}

.benefit-dialog {
  position: relative;
  width: min(84.6vw, 387px);
  max-height: calc(100dvh - 24px);
  outline: 0;
  transform-origin: center;
  animation: benefit-dialog-expand 280ms cubic-bezier(0.2, 0.82, 0.3, 1) both;
}

.benefit-scroll {
  max-height: calc(100dvh - 24px);
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}

.benefit-scroll::-webkit-scrollbar {
  display: none;
}

.benefit-artwork {
  position: relative;
}

.benefit-artwork__background {
  display: block;
  width: 100%;
  height: auto;
  user-select: none;
}

.benefit-artwork__ring {
  position: absolute;
  z-index: 1;
  top: -2%;
  left: -17%;
  width: 99%;
  aspect-ratio: 1;
  pointer-events: none;
  transform: translate(-5px, -30px);
}

.benefit-artwork__ring img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  user-select: none;
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

@keyframes benefit-dialog-expand {
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
  .benefit-dialog {
    animation: none;
  }
}
</style>
