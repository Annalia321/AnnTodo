import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi-svg'
import { en, zhHans, zhHant, ja, ko, fr, de, es, ru, pt } from 'vuetify/locale'

export const vuetifyLocaleMap: Record<string, string> = {
  'zh-CN': 'zhHans',
  'zh-TW': 'zhHant',
  en: 'en',
  ja: 'ja',
  ko: 'ko',
  fr: 'fr',
  de: 'de',
  es: 'es',
  ru: 'ru',
  pt: 'pt',
}

export const vuetify = createVuetify({
  icons: { defaultSet: 'mdi', aliases, sets: { mdi } },
  // Phones (< 600px) get the mobile layout: bottom navigation, bottom sheets, collapsed cards.
  display: { mobileBreakpoint: 'sm' },
  locale: { locale: 'en', fallback: 'en', messages: { en, zhHans, zhHant, ja, ko, fr, de, es, ru, pt } },
  theme: {
    defaultTheme: 'light',
    themes: {
      light: { dark: false, colors: { primary: '#1976D2', background: '#F4F6F9' } },
      dark: { dark: true, colors: { primary: '#64B5F6', background: '#121212' } },
    },
  },
})
