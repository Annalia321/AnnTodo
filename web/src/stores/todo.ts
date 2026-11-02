import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, type Card, type Item } from '@/api'
import { useSettingsStore } from './settings'
import { useUiStore } from './ui'

export interface NewItem {
  card_id: number
  title: string
  note?: string
  due_date?: string | null
  due_time?: string | null
  important?: boolean
  urgent?: boolean
  position?: 'top' | 'bottom'
}

export type ItemPatch = Partial<Pick<Item, 'title' | 'note' | 'due_date' | 'due_time' | 'important' | 'urgent' | 'card_id'>>

const byOrder = (a: { sort_order: number; id: number }, b: { sort_order: number; id: number }) =>
  a.sort_order - b.sort_order || a.id - b.id

export const useTodoStore = defineStore('todo', () => {
  const cards = ref<Card[]>([])
  const items = ref<Item[]>([])
  const loaded = ref(false)
  const settings = useSettingsStore()

  const sortedCards = computed(() => [...cards.value].sort(byOrder))

  const currentCardId = computed<number | null>(() => {
    const id = settings.s.current_card_id
    if (id && cards.value.some((c) => c.id === id)) return id
    return sortedCards.value[0]?.id ?? null
  })
  const currentCard = computed(() => cards.value.find((c) => c.id === currentCardId.value) ?? null)

  const cardById = (id: number) => cards.value.find((c) => c.id === id)
  const itemById = (id: number) => items.value.find((i) => i.id === id)

  function openItems(cardId: number) {
    return items.value.filter((i) => i.card_id === cardId && !i.completed).sort(byOrder)
  }

  function doneItems(cardId: number) {
    return items.value
      .filter((i) => i.card_id === cardId && i.completed)
      .sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? '') || b.id - a.id)
  }

  function openCount(cardId: number) {
    return items.value.reduce((n, i) => n + (i.card_id === cardId && !i.completed ? 1 : 0), 0)
  }

  function itemCount(cardId: number) {
    return items.value.reduce((n, i) => n + (i.card_id === cardId ? 1 : 0), 0)
  }

  function replaceItem(it: Item) {
    const idx = items.value.findIndex((i) => i.id === it.id)
    if (idx >= 0) items.value[idx] = it
    else items.value.push(it)
  }

  function reset() {
    cards.value = []
    items.value = []
    loaded.value = false
  }

  async function load() {
    const [c, i] = await Promise.all([api.get<Card[]>('/cards'), api.get<Item[]>('/items')])
    cards.value = c ?? []
    items.value = i ?? []
    loaded.value = true
  }

  /** Runs a server call; on failure reports the error and reloads the data. */
  async function guarded<T>(fn: () => Promise<T>): Promise<T | undefined> {
    try {
      return await fn()
    } catch (e) {
      useUiStore().error(e)
      try {
        await load()
      } catch {
        /* ignore */
      }
      return undefined
    }
  }

  // ---------- cards ----------

  async function setCurrentCard(id: number | null) {
    if (id === settings.s.current_card_id) return
    await settings.set({ current_card_id: id })
  }

  async function createCard(name: string, color = '') {
    const card = await api.post<Card>('/cards', { name, color })
    cards.value.push(card)
    return card
  }

  async function updateCard(id: number, patch: Partial<Pick<Card, 'name' | 'color'>>) {
    const card = await api.put<Card>(`/cards/${id}`, patch)
    const idx = cards.value.findIndex((c) => c.id === id)
    if (idx >= 0) cards.value[idx] = card
    return card
  }

  async function deleteCard(id: number) {
    const wasCurrent = currentCardId.value === id
    await api.del(`/cards/${id}`)
    cards.value = cards.value.filter((c) => c.id !== id)
    items.value = items.value.filter((i) => i.card_id !== id)
    if (wasCurrent) await setCurrentCard(sortedCards.value[0]?.id ?? null)
  }

  async function reorderCards(ids: number[]) {
    ids.forEach((id, i) => {
      const c = cardById(id)
      if (c) c.sort_order = i
    })
    await guarded(() => api.put('/cards/order', { ids }))
  }

  // ---------- items ----------

  async function createItem(input: NewItem) {
    const it = await api.post<Item>('/items', { position: settings.s.new_item_position, ...input })
    items.value.push(it)
    return it
  }

  async function updateItem(id: number, patch: ItemPatch) {
    const it = itemById(id)
    if (!it) return
    Object.assign(it, patch)
    if ('due_date' in patch && !patch.due_date) it.due_time = null
    const res = await guarded(() => api.put<Item>(`/items/${id}`, patch))
    if (res) replaceItem(res)
    return res
  }

  async function setCompleted(id: number, completed: boolean) {
    const it = itemById(id)
    if (!it) return
    it.completed = completed
    it.completed_at = completed ? new Date().toISOString() : null
    const res = await guarded(() => api.post<Item>(`/items/${id}/complete`, { completed }))
    if (res) replaceItem(res)
  }

  async function moveItem(id: number, cardId: number) {
    const it = itemById(id)
    if (!it || it.card_id === cardId) return
    it.card_id = cardId
    it.sort_order = Number.MAX_SAFE_INTEGER
    const res = await guarded(() => api.post<Item>(`/items/${id}/move`, { card_id: cardId }))
    if (res) replaceItem(res)
  }

  async function deleteItem(id: number) {
    items.value = items.value.filter((i) => i.id !== id)
    if (useUiStore().detailId === id) useUiStore().closeDetail()
    await guarded(() => api.del(`/items/${id}`))
  }

  async function restoreItem(id: number) {
    await guarded(async () => {
      await api.post(`/trash/item/${id}/restore`)
      await load()
    })
  }

  /** Stores the new order of a card's open items; items from other cards are moved into it. */
  async function reorderList(cardId: number, ids: number[]) {
    ids.forEach((id, i) => {
      const it = itemById(id)
      if (it) {
        it.card_id = cardId
        it.sort_order = i
      }
    })
    await guarded(() => api.put('/items/order', { lists: [{ card_id: cardId, ids }] }))
  }

  /**
   * Stores the order of one quadrant cell. When flags are given, the moved
   * item gets them; the order of the others changes without touching flags.
   */
  async function reorderQuadrant(ids: number[], moved?: { id: number; important: boolean; urgent: boolean }) {
    ids.forEach((id, i) => {
      const it = itemById(id)
      if (it) it.quadrant_sort_order = i
    })
    if (moved) {
      const it = itemById(moved.id)
      if (it) {
        it.important = moved.important
        it.urgent = moved.urgent
      }
    }
    await guarded(async () => {
      if (moved) await api.put(`/items/${moved.id}`, { important: moved.important, urgent: moved.urgent })
      await api.put('/items/quadrant-order', { cells: [{ ids }] })
    })
  }

  return {
    cards, items, loaded, sortedCards, currentCardId, currentCard, cardById, itemById,
    openItems, doneItems, openCount, itemCount, reset, load, setCurrentCard,
    createCard, updateCard, deleteCard, reorderCards,
    createItem, updateItem, setCompleted, moveItem, deleteItem, restoreItem, reorderList, reorderQuadrant,
  }
})
