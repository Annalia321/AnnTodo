import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/zh-tw'
import 'dayjs/locale/ja'
import 'dayjs/locale/ko'
import 'dayjs/locale/fr'
import 'dayjs/locale/de'
import 'dayjs/locale/es'
import 'dayjs/locale/ru'
import 'dayjs/locale/pt'
import { reactive, ref } from 'vue'
import type { Item } from '@/api'

dayjs.extend(utc)
dayjs.extend(timezone)

export { dayjs }

export const browserTimezone = (() => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
})()

export interface TimeConfig {
  tz: string
  dateFormat: string
  timeFormat: '24h' | '12h'
}

// Reactive so that templates using the formatters re-render when settings change.
const cfg = reactive<TimeConfig>({ tz: browserTimezone, dateFormat: 'YYYY-MM-DD', timeFormat: '24h' })

/** Bumped every minute so that overdue markers refresh without user interaction. */
const tick = ref(0)
setInterval(() => tick.value++, 60_000)

export function setTimeConfig(c: TimeConfig) {
  Object.assign(cfg, c)
}

export function getTz() {
  return cfg.tz
}

export function timePattern() {
  return cfg.timeFormat === '12h' ? 'h:mm A' : 'HH:mm'
}

export function fmtDate(d: dayjs.Dayjs) {
  return d.format(cfg.dateFormat)
}

export function fmtDateTime(iso: string | null | undefined) {
  if (!iso) return ''
  const d = dayjs(iso).tz(cfg.tz)
  return d.format(cfg.dateFormat + ' ' + timePattern())
}

/** Current time in the user's timezone. */
export function nowTz() {
  void tick.value
  return dayjs().tz(cfg.tz)
}

export function todayStr() {
  return nowTz().format('YYYY-MM-DD')
}

/** Local (user timezone) view of an item's due date and optional time. */
export function localDue(it: Pick<Item, 'due_date' | 'due_time'>): { date: string; time: string | null } | null {
  if (!it.due_date) return null
  if (!it.due_time) return { date: it.due_date, time: null }
  const d = dayjs.utc(`${it.due_date}T${it.due_time}:00Z`).tz(cfg.tz)
  return { date: d.format('YYYY-MM-DD'), time: d.format('HH:mm') }
}

/** Converts a local date / time in the user's timezone to the stored form. */
export function toStoredDue(date: string | null, time: string | null): { due_date: string | null; due_time: string | null } {
  if (!date) return { due_date: null, due_time: null }
  if (!time) return { due_date: date, due_time: null }
  const d = dayjs.tz(`${date}T${time}:00`, cfg.tz).utc()
  return { due_date: d.format('YYYY-MM-DD'), due_time: d.format('HH:mm') }
}

/** The moment an item becomes overdue: its due time, or the end of its due day. */
export function dueMoment(it: Pick<Item, 'due_date' | 'due_time'>): dayjs.Dayjs | null {
  if (!it.due_date) return null
  if (it.due_time) return dayjs.utc(`${it.due_date}T${it.due_time}:00Z`)
  return dayjs.tz(`${it.due_date}T00:00:00`, cfg.tz).endOf('day')
}

export function isOverdue(it: Item) {
  if (it.completed) return false
  const m = dueMoment(it)
  void tick.value
  return !!m && m.isBefore(dayjs())
}

/** True when the item is due within `days` days (or already overdue). */
export function dueWithin(it: Item, days: number) {
  if (it.completed) return false
  const local = localDue(it)
  if (!local) return false
  const limit = nowTz().startOf('day').add(days, 'day').format('YYYY-MM-DD')
  return local.date <= limit
}

export function formatDue(it: Item): string {
  const local = localDue(it)
  if (!local) return ''
  const d = dayjs(local.date)
  const now = nowTz()
  let text: string
  if (d.year() === now.year() && cfg.dateFormat === 'YYYY-MM-DD') text = d.format('MM-DD')
  else if (d.year() === now.year() && cfg.dateFormat === 'YYYY/MM/DD') text = d.format('MM/DD')
  else if (d.year() === now.year() && cfg.dateFormat === 'MM/DD/YYYY') text = d.format('MM/DD')
  else text = d.format(cfg.dateFormat)
  if (local.time) text += ' ' + dayjs(`2000-01-01T${local.time}`).format(timePattern())
  return text
}

export function dayjsLocale(lang: string) {
  const map: Record<string, string> = {
    'zh-CN': 'zh-cn',
    'zh-TW': 'zh-tw',
    ja: 'ja',
    ko: 'ko',
    fr: 'fr',
    de: 'de',
    es: 'es',
    ru: 'ru',
    pt: 'pt',
  }
  dayjs.locale(map[lang] ?? 'en')
}
