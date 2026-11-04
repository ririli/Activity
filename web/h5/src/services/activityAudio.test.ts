class MockAudio {
  static instances: MockAudio[] = []

  src: string
  loop = false
  preload = ''
  volume = 1
  play = vi.fn().mockResolvedValue(undefined)
  pause = vi.fn()
  addEventListener = vi.fn()

  constructor(src: string) {
    this.src = src
    MockAudio.instances.push(this)
  }
}

describe('activityAudio', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    MockAudio.instances = []
    vi.stubGlobal('Audio', MockAudio)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('starts looping activity BGM and persists the music switch', async () => {
    const audio = await import('./activityAudio')

    audio.initializeActivityAudio()

    expect(MockAudio.instances).toHaveLength(1)
    expect(MockAudio.instances[0].src).toBe('/audio/activity/bgm.mp3')
    expect(MockAudio.instances[0].loop).toBe(true)
    expect(MockAudio.instances[0].play).toHaveBeenCalledTimes(1)

    expect(audio.toggleActivityAudio()).toBe(false)
    expect(MockAudio.instances[0].pause).toHaveBeenCalledTimes(1)
    expect(localStorage.getItem('lottery_activity_audio_enabled')).toBe('false')
  })

  test('uses the configured effect and stops the previous effect', async () => {
    const audio = await import('./activityAudio')

    audio.playActivityEffect('drawStart')
    audio.playActivityEffect('reward')

    expect(MockAudio.instances.map((item) => item.src)).toEqual([
      '/audio/activity/draw-start.mp3',
      '/audio/activity/reward.mp3',
    ])
    expect(MockAudio.instances[0].pause).toHaveBeenCalledTimes(1)
    expect(MockAudio.instances[1].play).toHaveBeenCalledTimes(1)
  })
})
