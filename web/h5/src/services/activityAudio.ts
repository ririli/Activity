import { ref } from 'vue'

export type ActivityEffect = 'drawStart' | 'reward' | 'ringBoxReady' | 'tryBeforePayPreview'

const STORAGE_KEY = 'lottery_activity_audio_enabled'
const BGM_SOURCE = '/audio/activity/bgm.mp3'
const EFFECT_SOURCES: Record<ActivityEffect, string> = {
  drawStart: '/audio/activity/draw-start.mp3',
  reward: '/audio/activity/reward.mp3',
  ringBoxReady: '/audio/activity/ring-box-ready.mp3',
  tryBeforePayPreview: '/audio/activity/tbp-preview.mp3',
}
const EFFECT_VOLUMES: Record<ActivityEffect, number> = {
  drawStart: 0.78,
  reward: 0.86,
  ringBoxReady: 0.84,
  tryBeforePayPreview: 0.82,
}

function readInitialEnabled() {
  if (typeof window === 'undefined') return true
  try {
    return window.localStorage.getItem(STORAGE_KEY) !== 'false'
  } catch {
    return true
  }
}

export const activityAudioEnabled = ref(readInitialEnabled())

let bgmAudio: HTMLAudioElement | null = null
let effectAudio: HTMLAudioElement | null = null
let initialized = false

function canPlayAudio() {
  return typeof Audio !== 'undefined'
}

function ensureBgm() {
  if (!canPlayAudio()) return null
  if (bgmAudio) return bgmAudio

  bgmAudio = new Audio(BGM_SOURCE)
  bgmAudio.loop = true
  bgmAudio.preload = 'auto'
  bgmAudio.volume = 0.38
  return bgmAudio
}

function safePlay(audio: HTMLAudioElement) {
  try {
    const task = audio.play()
    task?.catch(() => undefined)
  } catch {
    // 浏览器未获得自动播放权限时，等待下一次用户交互重试。
  }
}

export function startActivityBgm() {
  if (!activityAudioEnabled.value) return
  const audio = ensureBgm()
  if (audio) safePlay(audio)
}

export function pauseActivityAudio() {
  bgmAudio?.pause()
  effectAudio?.pause()
  effectAudio = null
}

export function initializeActivityAudio() {
  if (typeof window === 'undefined') return
  startActivityBgm()
  if (initialized) return
  initialized = true

  const unlockAudio = () => {
    startActivityBgm()
    window.removeEventListener('click', unlockAudio)
    window.removeEventListener('keydown', unlockAudio)
  }

  window.addEventListener('click', unlockAudio)
  window.addEventListener('keydown', unlockAudio)
}

export function toggleActivityAudio() {
  activityAudioEnabled.value = !activityAudioEnabled.value
  try {
    window.localStorage.setItem(STORAGE_KEY, String(activityAudioEnabled.value))
  } catch {
    // 无持久化权限时仍保持当前页面内的开关状态。
  }

  if (activityAudioEnabled.value) {
    startActivityBgm()
  } else {
    pauseActivityAudio()
  }

  return activityAudioEnabled.value
}

export function playActivityEffect(effect: ActivityEffect) {
  if (!activityAudioEnabled.value || !canPlayAudio()) return

  effectAudio?.pause()
  const audio = new Audio(EFFECT_SOURCES[effect])
  audio.preload = 'auto'
  audio.volume = EFFECT_VOLUMES[effect]
  audio.addEventListener('ended', () => {
    if (effectAudio === audio) effectAudio = null
  }, { once: true })
  effectAudio = audio
  safePlay(audio)
}
