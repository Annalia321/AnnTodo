import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'

declare module 'vue-router' {
  interface RouteMeta {
    /** Pages for logged-out users (login, register, setup). */
    guest?: boolean
    admin?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'list', component: () => import('@/views/ListView.vue') },
    { path: '/cards', name: 'cards', component: () => import('@/views/CardsView.vue') },
    { path: '/calendar', name: 'calendar', component: () => import('@/views/CalendarView.vue') },
    { path: '/quadrant', name: 'quadrant', component: () => import('@/views/QuadrantView.vue') },
    { path: '/settings', name: 'settings', component: () => import('@/views/SettingsView.vue') },
    { path: '/admin', name: 'admin', component: () => import('@/views/AdminView.vue'), meta: { admin: true } },
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { guest: true } },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      props: { mode: 'register' },
      meta: { guest: true },
    },
    {
      path: '/setup',
      name: 'setup',
      component: () => import('@/views/RegisterView.vue'),
      props: { mode: 'setup' },
      meta: { guest: true },
    },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior(to, _from, saved) {
    if (saved) return saved
    if (to.hash) return { el: to.hash, top: 16, behavior: 'smooth' }
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.status) {
    try {
      await auth.loadStatus()
    } catch {
      /* server unreachable: fall through to the login page */
    }
  }
  if (auth.status && !auth.status.initialized) return to.name === 'setup' ? true : { name: 'setup' }
  if (to.name === 'setup') return { name: 'login' }
  if (!auth.checked) await auth.loadMe()

  if (to.meta.guest) {
    if (auth.user) return { name: 'list' }
    if (to.name === 'register' && !auth.status?.allow_register) return { name: 'login' }
    return true
  }
  if (!auth.user) return { name: 'login', query: to.fullPath !== '/' ? { redirect: to.fullPath } : {} }
  if (to.meta.admin && !auth.isAdmin) return { name: 'list' }

  const todo = useTodoStore()
  if (!todo.loaded) {
    try {
      await todo.load()
    } catch (e) {
      useUiStore().error(e)
    }
  }
  return true
})
