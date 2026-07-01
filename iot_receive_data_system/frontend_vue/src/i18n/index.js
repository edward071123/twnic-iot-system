import { createI18n } from 'vue-i18n'
import zhTW from './locales/zh-TW'
import en from './locales/en'
import vi from './locales/vi'

const supportedLocales = ['zh-TW', 'en', 'vi']
const savedLocale = localStorage.getItem('locale')
const browserLocale = navigator.language === 'zh-TW' ? 'zh-TW' : navigator.language.split('-')[0]
const initialLocale = supportedLocales.includes(savedLocale)
  ? savedLocale
  : supportedLocales.includes(browserLocale)
    ? browserLocale
    : 'zh-TW'

document.documentElement.lang = initialLocale

export const i18n = createI18n({
  legacy: false,
  locale: initialLocale,
  fallbackLocale: 'zh-TW',
  messages: {
    'zh-TW': zhTW,
    en,
    vi,
  },
})

export { supportedLocales }
