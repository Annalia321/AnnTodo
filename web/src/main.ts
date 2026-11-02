import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { i18n } from './locales'
import { vuetify } from './plugins/vuetify'
import { onUnauthorized } from './api'
import { useAuthStore } from './stores/auth'
import './styles/main.css'

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(vuetify)
app.use(router)

// A 401 from any request means the session ended (expired or revoked on another device).
onUnauthorized(() => {
  const auth = useAuthStore()
  if (!auth.user) return
  auth.clear()
  router.push({ name: 'login' })
})

app.mount('#app')

if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch((e) => console.warn('service worker registration failed', e))
  })
}
