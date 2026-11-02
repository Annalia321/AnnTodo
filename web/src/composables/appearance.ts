import { computed, onBeforeUnmount, ref, watch, watchEffect } from 'vue'
import { useLocale, useTheme } from 'vuetify'
import { i18n, rememberLanguage, resolveLanguage, storedLanguage, type LangCode } from '@/locales'
import { vuetifyLocaleMap } from '@/plugins/vuetify'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { dayjsLocale, setTimeConfig } from '@/utils/time'

/** Language chosen on the login page (before any user settings are available). */
export const guestLanguage = ref(storedLanguage())

export function setGuestLanguage(code: string) {
  guestLanguage.value = code
  rememberLanguage(code)
}

const fontStacks: Record<string, string> = {
  system:
    'system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", "PingFang SC", "Microsoft YaHei", "Noto Sans CJK SC", sans-serif',
  sans: '"Source Han Sans SC", "Source Han Sans", "Noto Sans CJK SC", "Noto Sans SC", "Microsoft YaHei", sans-serif',
  serif: '"Source Han Serif SC", "Source Han Serif", "Noto Serif CJK SC", "Noto Serif SC", "SimSun", serif',
}

const fontSizes: Record<string, string> = { s: '14px', m: '16px', l: '18px', xl: '20px' }

/** Applies the user's appearance, language and date settings to the whole app. */
export function useAppearance() {
  const auth = useAuthStore()
  const settings = useSettingsStore()
  const theme = useTheme()
  const vLocale = useLocale()

  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const prefersDark = ref(media.matches)
  const onMedia = (e: MediaQueryListEvent) => (prefersDark.value = e.matches)
  media.addEventListener('change', onMedia)
  onBeforeUnmount(() => media.removeEventListener('change', onMedia))

  const language = computed<LangCode>(() =>
    auth.user ? resolveLanguage(settings.s.language) : resolveLanguage(guestLanguage.value),
  )

  watch(
    language,
    (code) => {
      i18n.global.locale.value = code
      vLocale.current.value = vuetifyLocaleMap[code] ?? 'en'
      dayjsLocale(code)
      document.documentElement.lang = code
    },
    { immediate: true },
  )

  // Remember the logged-in user's choice so the login page uses it next time.
  watch(
    () => (auth.user ? settings.s.language : null),
    (v) => {
      if (v !== null) {
        rememberLanguage(v)
        guestLanguage.value = v
      }
    },
  )

  watchEffect(() => {
    const s = settings.s
    const dark = s.theme === 'dark' || (s.theme === 'system' && prefersDark.value)
    const accent = /^#[0-9a-f]{6}$/i.test(s.accent) ? s.accent : '#1976D2'
    theme.themes.value.light.colors.primary = accent
    theme.themes.value.dark.colors.primary = accent
    theme.change(dark ? 'dark' : 'light')
    document.querySelector('meta[name="theme-color"]')?.setAttribute('content', accent)
  })

  watchEffect(() => {
    const s = settings.s
    const root = document.documentElement
    let font = fontStacks[s.font] ?? fontStacks.system
    if (s.font === 'custom' && s.custom_font.trim()) {
      font = `"${s.custom_font.trim().replace(/["\\]/g, '')}", ${fontStacks.system}`
    }
    root.style.setProperty('--app-font', font)
    root.style.fontSize = fontSizes[s.font_size] ?? '16px'
    root.classList.toggle('density-compact', s.density === 'compact')
  })

  watchEffect(() => {
    setTimeConfig({ tz: settings.timezone, dateFormat: settings.s.date_format, timeFormat: settings.s.time_format })
  })

  watchEffect(() => {
    document.title = auth.status?.site_name || 'AnnTodo'
  })

  /** Vuetify component defaults for the selected density. */
  const defaults = computed(() => {
    const density = settings.s.density === 'compact' ? 'compact' : 'default'
    const d = { density }
    return {
      VBtn: d,
      VTextField: d,
      VTextarea: d,
      VSelect: d,
      VAutocomplete: d,
      VList: d,
      VListItem: d,
      VSwitch: d,
      VCheckbox: d,
      VBtnToggle: d,
      VTable: d,
    }
  })

  return { defaults, language }
}
