import { createI18n } from 'vue-i18n'
import en from './en'
import zhCN from './zh-CN'
import zhTW from './zh-TW'
import ja from './ja'
import ko from './ko'
import fr from './fr'
import de from './de'
import es from './es'
import ru from './ru'
import pt from './pt'
import type { Messages } from './en'

export const languages = [
  { code: 'zh-CN', label: '简体中文' },
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'en', label: 'English' },
  { code: 'ja', label: '日本語' },
  { code: 'ko', label: '한국어' },
  { code: 'fr', label: 'Français' },
  { code: 'de', label: 'Deutsch' },
  { code: 'es', label: 'Español' },
  { code: 'ru', label: 'Русский' },
  { code: 'pt', label: 'Português' },
] as const

export type LangCode = (typeof languages)[number]['code']

const codes = languages.map((l) => l.code) as string[]

/** Picks the best supported language from the browser, falling back to English. */
export function detectLanguage(): LangCode {
  const prefs = navigator.languages?.length ? navigator.languages : [navigator.language || 'en']
  for (const raw of prefs) {
    const p = raw.toLowerCase()
    if (p.startsWith('zh')) {
      return /(tw|hk|mo|hant)/.test(p) ? 'zh-TW' : 'zh-CN'
    }
    const base = p.split('-')[0]
    if (codes.includes(base)) return base as LangCode
  }
  return 'en'
}

export function resolveLanguage(setting: string | null | undefined): LangCode {
  if (setting && codes.includes(setting)) return setting as LangCode
  return detectLanguage()
}

const LS_KEY = 'anntodo.lang'

export function storedLanguage(): string {
  try {
    return localStorage.getItem(LS_KEY) || ''
  } catch {
    return ''
  }
}

export function rememberLanguage(code: string) {
  try {
    if (code) localStorage.setItem(LS_KEY, code)
    else localStorage.removeItem(LS_KEY)
  } catch {
    /* ignore */
  }
}

export const i18n = createI18n<[Messages], LangCode, false>({
  legacy: false,
  locale: resolveLanguage(storedLanguage()),
  fallbackLocale: 'en',
  messages: { en, 'zh-CN': zhCN, 'zh-TW': zhTW, ja, ko, fr, de, es, ru, pt },
})
