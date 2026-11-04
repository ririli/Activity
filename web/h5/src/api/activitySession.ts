import request from './request'

export interface ActivityConfig {
  edition_id: string
  name: string
  status: string
  starts_at: string
  ends_at: string
  settlement_status: string
}

export interface Prize {
  code: string
  item_id: string
  name: string
  quantity: number
  weight?: number
  choices?: Prize[]
}

export interface PoolRule {
  cost: number
  prizes: Prize[]
  flowers?: unknown[]
}

export interface ActivityRules {
  pools: Record<string, PoolRule>
  stages: Record<string, Prize>
  chests: Prize[]
}

export interface JoinSummary {
  edition_id: string
  gold: number
  petals: number
  round_no: number
  flowers: number
  unlocked_chests: number
  completed_chests: number
  claimed_stage_mask: number
  score: number
  rules: ActivityRules
}

export interface ActivityState {
  gold: number
  petals: number
  roundNo: number
  formalDrawCount: number
  flowers: number
  unlockedChests: number
  completedChests: number
  claimedStageMask: number
  status: string
  rules: ActivityRules | null
}

let cachedConfig: ActivityConfig | null = null
let cachedEdition: string | null = null
let cachedRules: ActivityRules | null = null
let cachedSummary: JoinSummary | null = null
let loadPromise: Promise<ActivityConfig> | null = null

function isActive(config: ActivityConfig): boolean {
  if (config.status !== 'ACTIVE') return false
  const now = Date.now()
  const start = Date.parse(config.starts_at)
  const end = Date.parse(config.ends_at)
  return (!Number.isNaN(start) ? start <= now : true) && (!Number.isNaN(end) ? now < end : true)
}

function pickConfig(list: ActivityConfig[]): ActivityConfig {
  if (!list.length) throw new Error('活动配置不存在')
  const active = list.find(isActive)
  if (active) return active
  return [...list].sort((a, b) => Date.parse(b.starts_at) - Date.parse(a.starts_at))[0]
}

function parseRules(value: unknown): ActivityRules | null {
  if (!value) return null
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as ActivityRules
    } catch {
      return null
    }
  }
  return value as ActivityRules
}

async function loadActivity(): Promise<ActivityConfig> {
  const list = await request.get('/activity/v1/records/config').then((res) => res.data as ActivityConfig[])
  const config = pickConfig(list)
  cachedConfig = config
  cachedEdition = config.edition_id
  cachedRules = null
  cachedSummary = null
  if (isActive(config)) {
    try {
      const summary = await request.post('/activity/v1/join', {
        edition_id: config.edition_id,
      }).then((res) => res.data as JoinSummary)
      cachedSummary = summary
      cachedRules = summary.rules ?? null
    } catch (error) {
      // Account initialization is eventual; let the caller retry its page load.
      if (error instanceof Error && error.message.includes('初始化')) throw error
    }
  }
  return config
}

// Loads the current edition once and, while active, idempotently joins so the
// activity wallet/progress exists. All activity/ranking adapters call this;
// concurrent page-load calls share one request chain.
export function ensureActivity(force = false): Promise<ActivityConfig> {
  // Always join an in-flight load first. loadActivity publishes the config
  // before the idempotent join completes; falling through to the cache in that
  // window would expose a config with null rules.
  if (loadPromise) return loadPromise
  if (!force && cachedConfig && cachedEdition) return Promise.resolve(cachedConfig)
  loadPromise = loadActivity().finally(() => {
    loadPromise = null
  })
  return loadPromise
}

export function activityEditionId(): string {
  if (!cachedEdition) throw new Error('活动尚未加载')
  return cachedEdition
}

export function activityRules(): ActivityRules | null {
  return cachedRules
}

export function activitySummary(): JoinSummary | null {
  return cachedSummary
}

export function activityIsActive(): boolean {
  return cachedConfig ? isActive(cachedConfig) : false
}

export async function loadState(): Promise<ActivityState> {
  const config = await ensureActivity()
  const rows = await request.get('/activity/v1/records/state', {
    params: { edition_id: config.edition_id },
  }).then((res) => res.data as Array<Record<string, unknown>>)
  const row = rows[0] ?? {}
  const rules = cachedRules ?? parseRules(row.rule_json)
  if (!cachedRules && rules) cachedRules = rules
  return {
    gold: Number(row.gold_balance ?? 0),
    petals: Number(row.petal_balance ?? 0),
    roundNo: Number(row.current_round_no ?? 1),
    formalDrawCount: Number(row.formal_draw_count ?? 0),
    flowers: Number(row.flower_count ?? 0),
    unlockedChests: Number(row.unlocked_chests ?? 0),
    completedChests: Number(row.completed_chests ?? 0),
    claimedStageMask: Number(row.claimed_stage_mask ?? 0),
    status: String(row.status ?? ''),
    rules,
  }
}

// Re-reads the authoritative summary after a mutation. Calling join again is
// idempotent and returns the current wallet/progress plus the rule snapshot.
export async function refreshSummary(): Promise<ActivityState | null> {
  const config = await ensureActivity()
  if (isActive(config)) {
    const summary = await request.post('/activity/v1/join', {
      edition_id: config.edition_id,
    }).then((res) => res.data as JoinSummary)
    cachedSummary = summary
    cachedRules = summary.rules ?? cachedRules
    return {
      gold: summary.gold,
      petals: summary.petals,
      roundNo: summary.round_no,
      formalDrawCount: 0,
      flowers: summary.flowers,
      unlockedChests: summary.unlocked_chests,
      completedChests: summary.completed_chests,
      claimedStageMask: summary.claimed_stage_mask,
      status: 'ACTIVE',
      rules: cachedRules,
    }
  }
  return loadState()
}

export function resetActivitySession(): void {
  cachedConfig = null
  cachedEdition = null
  cachedRules = null
  cachedSummary = null
  loadPromise = null
}
