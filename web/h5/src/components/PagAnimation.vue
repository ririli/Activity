<template>
  <canvas
    ref="canvasRef"
    class="pag-animation"
    :data-pag-src="src"
    aria-hidden="true"
  ></canvas>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { getPagRuntime } from '@/services/pagRuntime'

const props = defineProps<{
  src: string
}>()

const emit = defineEmits<{
  complete: []
  error: []
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
let disposed = false
let pagFile: { destroy: () => void } | null = null
let pagView: {
  addListener: (eventName: 'onAnimationEnd', listener: () => void) => void
  destroy: () => void
  setRepeatCount: (count: number) => void
  play: () => Promise<void>
} | null = null

onMounted(async () => {
  if (import.meta.env.MODE === 'test') {
    await nextTick()
    setTimeout(() => emit('complete'), 80)
    return
  }
  if (!canvasRef.value) return

  try {
    const [PAG, buffer] = await Promise.all([
      getPagRuntime(),
      fetch(props.src).then((response) => {
        if (!response.ok) throw new Error(`PAG资源加载失败：${response.status}`)
        return response.arrayBuffer()
      }),
    ])
    if (disposed || !canvasRef.value) return

    const file = await PAG.PAGFile.load(buffer)
    if (disposed) {
      file.destroy()
      return
    }
    pagFile = file
    canvasRef.value.width = file.width()
    canvasRef.value.height = file.height()

    const view = await PAG.PAGView.init(file, canvasRef.value, { useScale: false })
    if (!view) throw new Error('PAG画布初始化失败')
    if (disposed) {
      view.destroy()
      return
    }
    pagView = view
    view.setRepeatCount(1)
    view.addListener('onAnimationEnd', () => emit('complete'))
    await view.play()
  } catch (error) {
    console.warn('许愿飞行动效加载失败，已直接展示奖励。', error)
    emit('error')
  }
})

onBeforeUnmount(() => {
  disposed = true
  pagView?.destroy()
  pagView = null
  pagFile?.destroy()
  pagFile = null
})
</script>

<style scoped>
.pag-animation {
  display: block;
  pointer-events: none;
}
</style>
