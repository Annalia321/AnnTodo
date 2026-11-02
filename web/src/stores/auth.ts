import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, type AuthResult, type SystemStatus, type User } from '@/api'
import { useSettingsStore } from './settings'
import { useTodoStore } from './todo'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const status = ref<SystemStatus | null>(null)
  const checked = ref(false)

  const isAdmin = computed(() => user.value?.role === 'admin')
  const displayName = computed(() => user.value?.nickname || user.value?.username || '')

  async function loadStatus() {
    status.value = await api.get<SystemStatus>('/system/status')
    return status.value
  }

  function apply(res: AuthResult) {
    user.value = res.user
    useSettingsStore().load(res.settings)
  }

  async function loadMe() {
    try {
      apply(await api.get<AuthResult>('/auth/me'))
    } catch {
      user.value = null
    } finally {
      checked.value = true
    }
    return user.value
  }

  async function login(username: string, password: string, remember: boolean) {
    apply(await api.post<AuthResult>('/auth/login', { username, password, remember }))
  }

  async function register(body: { username: string; password: string; nickname: string; email: string }) {
    apply(await api.post<AuthResult>('/auth/register', body))
  }

  async function setup(body: { username: string; password: string; nickname: string; email: string }) {
    apply(await api.post<AuthResult>('/auth/setup', body))
    if (status.value) status.value.initialized = true
  }

  function clear() {
    user.value = null
    useSettingsStore().reset()
    useTodoStore().reset()
  }

  async function logout() {
    try {
      await api.post('/auth/logout')
    } finally {
      clear()
    }
  }

  return { user, status, checked, isAdmin, displayName, loadStatus, loadMe, login, register, setup, logout, clear }
})
