export interface User {
  id: number
  username: string
  email: string | null
  nickname: string
  role: 'admin' | 'user'
  status: 'active' | 'disabled'
  has_avatar: boolean
  avatar_url: string
  last_login_at: string | null
  created_at: string
}

export interface Card {
  id: number
  name: string
  color: string
  sort_order: number
}

export interface Item {
  id: number
  card_id: number
  title: string
  note: string
  due_date: string | null // local date when due_time is null, otherwise UTC date
  due_time: string | null // HH:MM in UTC
  important: boolean
  urgent: boolean
  completed: boolean
  completed_at: string | null
  sort_order: number
  quadrant_sort_order: number
  created_at: string
  updated_at: string
}

export interface SystemStatus {
  initialized: boolean
  allow_register: boolean
  site_name: string
  version: string
}

export interface AuthResult {
  user: User
  settings: Record<string, unknown>
}

export interface SessionInfo {
  id: string
  device: string
  ip: string
  created_at: string
  last_active_at: string
  expires_at: string
  current: boolean
}

export interface TrashData {
  cards: { id: number; name: string; color: string; item_count: number; deleted_at: string }[]
  items: { id: number; title: string; card_id: number; card_name: string; deleted_at: string }[]
}

export class ApiError extends Error {
  status: number
  code: string
  data: any
  constructor(status: number, code: string, message: string, data?: any) {
    super(message)
    this.status = status
    this.code = code
    this.data = data
  }
}

const BASE = '/api/v1'

function csrfToken(): string {
  const m = document.cookie.match(/(?:^|;\s*)anntodo_csrf=([^;]+)/)
  return m ? decodeURIComponent(m[1]) : ''
}

let unauthorizedHandler: (() => void) | null = null
export function onUnauthorized(fn: () => void) {
  unauthorizedHandler = fn
}

async function ensureCsrf() {
  if (!csrfToken()) await fetch(BASE + '/system/status', { credentials: 'same-origin' })
}

export async function request<T = any>(method: string, path: string, body?: unknown, raw = false): Promise<T> {
  const headers: Record<string, string> = {}
  let payload: BodyInit | undefined
  if (method !== 'GET') {
    await ensureCsrf()
    headers['X-CSRF-Token'] = csrfToken()
  }
  if (body instanceof FormData) {
    payload = body
  } else if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
    payload = raw ? (body as string) : JSON.stringify(body)
  }
  let res: Response
  try {
    res = await fetch(BASE + path, { method, headers, body: payload, credentials: 'same-origin' })
  } catch (e) {
    throw new ApiError(0, 'network', String(e))
  }
  if (res.status === 204) return undefined as T
  const text = await res.text()
  let data: any = undefined
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  if (!res.ok) {
    const err = data?.error ?? {}
    if (res.status === 401 && path !== '/auth/login' && unauthorizedHandler) unauthorizedHandler()
    throw new ApiError(res.status, err.code ?? 'http_' + res.status, err.message ?? res.statusText, data)
  }
  return data as T
}

export const api = {
  get: <T = any>(p: string) => request<T>('GET', p),
  post: <T = any>(p: string, b?: unknown) => request<T>('POST', p, b ?? {}),
  put: <T = any>(p: string, b?: unknown) => request<T>('PUT', p, b ?? {}),
  del: <T = any>(p: string) => request<T>('DELETE', p),
}

export interface McpTokenInfo {
  exists: boolean
  hint: string
  created_at: string | null
  last_used_at: string | null
}

