import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '@/api'
import { browserTimezone } from '@/utils/time'

export interface UserSettings {
  language: string // '' = follow browser
  theme: 'light' | 'dark' | 'system'
  accent: string
  font: 'system' | 'sans' | 'serif' | 'custom'
  custom_font: string
  font_size: 's' | 'm' | 'l' | 'xl'
  density: 'comfortable' | 'compact'
  timezone: string // '' = browser timezone
  week_start: 0 | 1
  date_format: 'YYYY-MM-DD' | 'YYYY/MM/DD' | 'MM/DD/YYYY'
  time_format: '24h' | '12h'
  new_item_position: 'top' | 'bottom'
  expand_completed: boolean
  auto_urgent: boolean
  auto_urgent_days: number
  current_card_id: number | null
  quadrant_show_completed: boolean
  calendar_show_completed: boolean
  calendar_hidden_cards: number[]
  onboarding_done: boolean
}

export const defaultSettings: UserSettings = {
  language: '',
  theme: 'system',
  accent: '#1976D2',
  font: 'system',
  custom_font: '',
  font_size: 'm',
  density: 'comfortable',
  timezone: '',
  week_start: 0,
  date_format: 'YYYY-MM-DD',
  time_format: '24h',
  new_item_position: 'bottom',
  expand_completed: false,
  auto_urgent: false,
  auto_urgent_days: 2,
  current_card_id: null,
  quadrant_show_completed: false,
  calendar_show_completed: true,
  calendar_hidden_cards: [],
  onboarding_done: false,
}

export const accentPresets = [
  '#1976D2', // blue
  '#0097A7', // cyan
  '#00897B', // teal
  '#43A047', // green
  '#7CB342', // light green
  '#F9A825', // amber
  '#FB8C00', // orange
  '#E53935', // red
  '#D81B60', // pink
  '#8E24AA', // purple
]

export const useSettingsStore = defineStore('settings', () => {
  const raw = ref<Record<string, unknown>>({})

  const s = computed<UserSettings>(() => ({ ...defaultSettings, ...(raw.value as Partial<UserSettings>) }))
  const timezone = computed(() => s.value.timezone || browserTimezone)

  function load(values: Record<string, unknown> | null | undefined) {
    raw.value = { ...(values ?? {}) }
  }

  function reset() {
    raw.value = {}
  }

  /** Updates one or more settings locally and on the server. */
  async function set(patch: Partial<UserSettings>) {
    raw.value = { ...raw.value, ...patch }
    try {
      const saved = await api.put<Record<string, unknown>>('/settings', patch)
      if (saved) raw.value = saved
    } catch (e) {
      console.error('saving settings failed', e)
    }
  }

  return { raw, s, timezone, load, reset, set }
})
