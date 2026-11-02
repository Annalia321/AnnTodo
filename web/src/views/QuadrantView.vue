<template>
  <div class="quadrant-page page">
    <div class="page-head">
      <v-menu v-if="todo.cards.length" location="bottom start">
        <template #activator="{ props: p }">
          <v-btn v-bind="p" variant="text" class="card-picker" :append-icon="mdiChevronDown">
            <span class="dot" :style="{ background: cardColor(todo.currentCard) }" />
            <span class="page-title text-truncate">{{ todo.currentCard?.name }}</span>
          </v-btn>
        </template>
        <v-list min-width="220">
          <v-list-item
            v-for="c in todo.sortedCards"
            :key="c.id"
            :active="c.id === todo.currentCardId"
            color="primary"
            @click="todo.setCurrentCard(c.id)"
          >
            <template #prepend><span class="dot" :style="{ background: cardColor(c) }" /></template>
            <v-list-item-title>{{ c.name }}</v-list-item-title>
            <template #append>
              <span class="text-caption muted">{{ todo.openCount(c.id) }}</span>
            </template>
          </v-list-item>
        </v-list>
      </v-menu>
      <v-spacer />
      <v-switch
        v-if="todo.cards.length"
        :model-value="settings.s.quadrant_show_completed"
        :label="t('quadrant.showCompleted')"
        color="primary"
        density="compact"
        hide-details
        inset
        class="flex-grow-0"
        @update:model-value="(v: unknown) => settings.set({ quadrant_show_completed: !!v })"
      />
    </div>

    <div v-if="!todo.currentCard" class="empty-state">
      <div class="muted mb-4">{{ t('quadrant.noCard') }}</div>
      <v-btn color="primary" to="/cards">{{ t('cards.newCard') }}</v-btn>
    </div>

    <div v-else class="grid">
      <section v-for="cell in cells" :key="cell.key" class="cell" :class="cell.key">
        <header class="cell-head">
          <span class="cell-badge">{{ cell.key.toUpperCase() }}</span>
          <span class="cell-title">{{ t('quadrant.' + cell.key) }}</span>
          <v-spacer />
          <span class="text-caption muted">{{ lists[cell.key].length }}</span>
        </header>
        <div class="cell-body">
          <draggable
            :list="lists[cell.key]"
            item-key="id"
            group="quadrant"
            :animation="150"
            ghost-class="ghost"
            :delay="200"
            :delay-on-touch-only="true"
            class="cell-list"
            @change="(e: DragChange) => onChange(cell, e)"
          >
            <template #item="{ element }">
              <ItemRow :item="element" :color="cardColor(todo.currentCard)">
                <template v-if="isAutoUrgent(element) && !element.urgent" #append>
                  <v-chip size="x-small" color="error" variant="tonal" class="mt-2 mr-1">{{ t('quadrant.autoUrgentBadge') }}</v-chip>
                </template>
              </ItemRow>
            </template>
          </draggable>
          <div v-if="!lists[cell.key].length && !doneLists[cell.key].length" class="cell-empty">{{ t('quadrant.empty') }}</div>
          <ItemRow v-for="it in doneLists[cell.key]" :key="it.id" :item="it" />
        </div>
        <div class="add-row">
          <v-icon :icon="mdiPlus" size="18" class="muted" />
          <input
            v-model="drafts[cell.key]"
            class="add-input"
            :placeholder="t('list.addItem')"
            maxlength="500"
            @keydown.enter="(e: KeyboardEvent) => add(cell, e)"
          />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import draggable from 'vuedraggable'
import { mdiChevronDown, mdiPlus } from '@mdi/js'
import ItemRow from '@/components/ItemRow.vue'
import type { Item } from '@/api'
import { useSettingsStore } from '@/stores/settings'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor } from '@/utils/cards'
import { dueWithin } from '@/utils/time'

type CellKey = 'q1' | 'q2' | 'q3' | 'q4'
interface Cell {
  key: CellKey
  important: boolean
  urgent: boolean
}
interface DragChange {
  added?: { element: Item; newIndex: number }
  moved?: { element: Item; oldIndex: number; newIndex: number }
  removed?: { element: Item; oldIndex: number }
}

const cells: Cell[] = [
  { key: 'q1', important: true, urgent: true },
  { key: 'q2', important: true, urgent: false },
  { key: 'q3', important: false, urgent: true },
  { key: 'q4', important: false, urgent: false },
]

const { t } = useI18n()
const todo = useTodoStore()
const settings = useSettingsStore()
const ui = useUiStore()

/** Shown as urgent because of the "due within N days" rule (display only). */
function isAutoUrgent(it: Item) {
  return settings.s.auto_urgent && dueWithin(it, settings.s.auto_urgent_days)
}

function cellOf(it: Item): CellKey {
  const urgent = it.urgent || isAutoUrgent(it)
  if (it.important) return urgent ? 'q1' : 'q2'
  return urgent ? 'q3' : 'q4'
}

const byQuadrantOrder = (a: Item, b: Item) => a.quadrant_sort_order - b.quadrant_sort_order || a.id - b.id

const grouped = computed(() => {
  const open: Record<CellKey, Item[]> = { q1: [], q2: [], q3: [], q4: [] }
  const done: Record<CellKey, Item[]> = { q1: [], q2: [], q3: [], q4: [] }
  const cardId = todo.currentCardId
  for (const it of todo.items) {
    if (it.card_id !== cardId) continue
    if (it.completed) {
      if (settings.s.quadrant_show_completed) done[cellOf(it)].push(it)
    } else {
      open[cellOf(it)].push(it)
    }
  }
  for (const k of Object.keys(open) as CellKey[]) {
    open[k].sort(byQuadrantOrder)
    done[k].sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? ''))
  }
  return { open, done }
})

// Local copies for vuedraggable, which mutates the arrays it is given.
const lists = reactive<Record<CellKey, Item[]>>({ q1: [], q2: [], q3: [], q4: [] })
const doneLists = computed(() => grouped.value.done)

function rebuild() {
  for (const k of Object.keys(lists) as CellKey[]) lists[k] = [...grouped.value.open[k]]
}
watch(grouped, rebuild, { immediate: true })

function onChange(cell: Cell, e: DragChange) {
  const ids = lists[cell.key].map((i) => i.id)
  if (e.moved) {
    todo.reorderQuadrant(ids)
  } else if (e.added) {
    const it = e.added.element
    if (isAutoUrgent(it) && !cell.urgent) {
      ui.notify(t('quadrant.autoUrgentWarn'), { color: 'warning' })
      nextTick(rebuild)
      return
    }
    // Keep the stored "urgent" flag of auto-urgent items moved between urgent cells.
    const urgent = cell.urgent && isAutoUrgent(it) ? it.urgent : cell.urgent
    todo.reorderQuadrant(ids, { id: it.id, important: cell.important, urgent })
  }
}

const drafts = reactive<Record<CellKey, string>>({ q1: '', q2: '', q3: '', q4: '' })

async function add(cell: Cell, e: KeyboardEvent) {
  if (e.isComposing) return
  const title = drafts[cell.key].trim()
  const cardId = todo.currentCardId
  if (!title || !cardId) return
  drafts[cell.key] = ''
  try {
    await todo.createItem({ card_id: cardId, title, important: cell.important, urgent: cell.urgent })
  } catch (err) {
    drafts[cell.key] = title
    ui.error(err)
  }
}
</script>

<style scoped>
.quadrant-page {
  display: flex;
  flex-direction: column;
  height: calc(100dvh - var(--v-layout-top, 0px) - var(--v-layout-bottom, 0px));
}
.card-picker {
  text-transform: none;
  letter-spacing: normal;
  max-width: 70vw;
}
.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 8px;
  flex: 0 0 auto;
}
.grid {
  flex: 1 1 auto;
  min-height: 0;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  grid-template-rows: repeat(2, minmax(0, 1fr));
  gap: var(--gap);
}
.cell {
  --q-color: #e53935;
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-radius: 12px;
  background: rgb(var(--v-theme-surface));
  border-top: 4px solid var(--q-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}
.cell.q2 {
  --q-color: #fb8c00;
}
.cell.q3 {
  --q-color: #1e88e5;
}
.cell.q4 {
  --q-color: #78909c;
}
.cell-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 4px;
}
.cell-badge {
  font-size: 0.7rem;
  font-weight: 700;
  color: #fff;
  background: var(--q-color);
  border-radius: 6px;
  padding: 1px 6px;
}
.cell-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cell-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 0 6px;
  display: flex;
  flex-direction: column;
}
.cell-list {
  min-height: 40px;
}
.cell-empty {
  flex: 1 1 auto;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(var(--v-theme-on-surface), 0.3);
  font-size: 0.9rem;
  margin-top: -40px;
  pointer-events: none;
}
.add-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 4px 8px 8px;
  padding: 5px 8px;
  border-radius: 8px;
  background: rgba(var(--v-theme-on-surface), 0.04);
}
.add-input {
  flex: 1 1 auto;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: inherit;
  font: inherit;
}
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 50vh;
}
@media (max-width: 599px) {
  .quadrant-page {
    height: auto;
    min-height: calc(100dvh - var(--v-layout-bottom, 0px));
  }
  .grid {
    grid-template-rows: none;
    grid-auto-rows: minmax(220px, 42dvh);
  }
  .cell-title {
    font-size: 0.85rem;
  }
}
</style>
