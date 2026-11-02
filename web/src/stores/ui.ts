import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ApiError } from '@/api'
import { i18n } from '@/locales'

export interface Toast {
  id: number
  text: string
  color?: string
  actionText?: string
  action?: () => void
  timeout?: number
}

let seq = 0

export function errorText(e: unknown): string {
  const t = i18n.global.t as (k: string, v?: Record<string, unknown>) => string
  const te = i18n.global.te as (k: string) => boolean
  if (e instanceof ApiError) {
    if (e.code === 'too_many_attempts') return t('errors.too_many_attempts', { s: e.data?.error?.retry_after ?? 60 })
    if (te('errors.' + e.code)) return t('errors.' + e.code)
    return e.message || t('errors.unknown')
  }
  return t('errors.unknown')
}

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  /** Item shown in the detail panel. */
  const detailId = ref<number | null>(null)

  function notify(text: string, opts: Partial<Omit<Toast, 'id' | 'text'>> = {}) {
    toasts.value.push({ id: ++seq, text, timeout: 4000, ...opts })
  }

  function error(e: unknown) {
    console.error(e)
    notify(errorText(e), { color: 'error' })
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter((x) => x.id !== id)
  }

  function openDetail(id: number) {
    detailId.value = id
  }

  function closeDetail() {
    detailId.value = null
  }

  return { toasts, detailId, notify, error, dismiss, openDetail, closeDetail }
})
