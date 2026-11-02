<template>
  <div
    ref="root"
    class="item-swipe"
    :class="{ swiping: dragging }"
    @pointerdown="onDown"
    @pointermove="onMove"
    @pointerup="onUp"
    @pointercancel="onCancel"
    @click.capture="onClickCapture"
  >
    <button
      v-show="offset < 0"
      class="swipe-delete"
      :style="{ width: `${-offset}px` }"
      :aria-label="t('item.delete')"
      tabindex="-1"
      @click.stop="remove"
    >
      <v-icon :icon="mdiTrashCanOutline" size="22" />
    </button>
    <div class="item-row" :class="{ done: item.completed, leaving }" :style="rowStyle">
      <v-checkbox-btn
        :model-value="item.completed || leaving"
        density="compact"
        :color="color || 'primary'"
        class="item-check"
        @update:model-value="toggle"
        @click.stop
      />
      <div class="item-main" @click="ui.openDetail(item.id)">
        <div class="item-title">{{ item.title }}</div>
        <div v-if="due" class="item-due" :class="{ overdue }">
          <v-icon :icon="mdiClockOutline" size="12" />
          {{ due }}
        </div>
      </div>
      <slot name="append" />
      <v-btn
        class="hover-delete"
        :icon="mdiTrashCanOutline"
        size="x-small"
        variant="text"
        :aria-label="t('item.delete')"
        :title="t('item.delete')"
        @click.stop="remove"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { ref } from 'vue'
/** Id of the row whose delete button is revealed; only one row is open at a time. */
const openRowId = ref<number | null>(null)
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiClockOutline, mdiTrashCanOutline } from '@mdi/js'
import type { Item } from '@/api'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { formatDue, isOverdue } from '@/utils/time'

const props = defineProps<{ item: Item; color?: string }>()
const todo = useTodoStore()
const ui = useUiStore()
const { t } = useI18n()

const leaving = ref(false)
const due = computed(() => (props.item.completed ? '' : formatDue(props.item)))
const overdue = computed(() => isOverdue(props.item))

async function toggle(v: unknown) {
  const completed = !!v
  if (completed && !props.item.completed) {
    // Short strike-through animation before the item moves to "Completed".
    leaving.value = true
    await new Promise((r) => setTimeout(r, 280))
    leaving.value = false
  }
  await todo.setCompleted(props.item.id, completed)
}

async function remove() {
  const id = props.item.id
  openRowId.value = null
  await todo.deleteItem(id)
  ui.notify(t('item.deleted'), { actionText: t('common.undo'), action: () => todo.restoreItem(id), timeout: 6000 })
}

// ---------- swipe left to reveal the delete button (touch / pen only) ----------
// Mouse drags keep their meaning (drag to reorder); mouse users get the hover button instead.

const REVEAL = 72 // width of the revealed delete button
const LOCK = 8 // movement needed before deciding between swipe and scroll

const root = ref<HTMLElement>()
const offset = ref(0)
const dragging = ref(false)
let startX = 0
let startY = 0
let base = 0
let pointerId: number | null = null
let mode: 'none' | 'swipe' | 'scroll' = 'none'
let suppressClick = false
let downOnDelete = false

const rowStyle = computed(() =>
  offset.value ? { transform: `translateX(${offset.value}px)` } : undefined,
)

function onDown(e: PointerEvent) {
  if (e.pointerType === 'mouse' || pointerId !== null) return
  pointerId = e.pointerId
  startX = e.clientX
  startY = e.clientY
  base = offset.value
  mode = 'none'
  suppressClick = false // a swipe that ended without a click must not swallow this gesture's click
  downOnDelete = !!(e.target as HTMLElement).closest('.swipe-delete')
}

function onMove(e: PointerEvent) {
  if (e.pointerId !== pointerId) return
  const dx = e.clientX - startX
  const dy = e.clientY - startY
  if (mode === 'none') {
    if (Math.abs(dx) < LOCK && Math.abs(dy) < LOCK) return
    mode = Math.abs(dx) > Math.abs(dy) ? 'swipe' : 'scroll'
    if (mode === 'swipe') {
      dragging.value = true
      openRowId.value = props.item.id
      root.value?.setPointerCapture(e.pointerId)
    }
  }
  if (mode !== 'swipe') return
  // Rubber-band past the reveal width, no movement to the right of the rest position.
  let next = base + dx
  if (next < -REVEAL) next = -REVEAL + (next + REVEAL) * 0.3
  offset.value = Math.min(0, next)
}

function finish(e: PointerEvent, cancelled: boolean) {
  if (e.pointerId !== pointerId) return
  pointerId = null
  if (mode === 'swipe') {
    suppressClick = !cancelled
    dragging.value = false
    const open = offset.value < -REVEAL / 2
    offset.value = open ? -REVEAL : 0
    if (!open && openRowId.value === props.item.id) openRowId.value = null
  } else if (mode === 'none' && offset.value && !downOnDelete) {
    // A tap on an open row closes it instead of opening the item.
    suppressClick = true
    close()
  }
  mode = 'none'
}
const onUp = (e: PointerEvent) => finish(e, false)
const onCancel = (e: PointerEvent) => finish(e, true)

function onClickCapture(e: MouseEvent) {
  if (!suppressClick) return
  suppressClick = false
  e.stopPropagation()
  e.preventDefault()
}

function close() {
  offset.value = 0
  if (openRowId.value === props.item.id) openRowId.value = null
}

// Close when another row opens, or when the user touches anywhere outside this row.
watch(openRowId, (id) => {
  if (id !== props.item.id && offset.value && !dragging.value) offset.value = 0
})
function onDocDown(e: PointerEvent) {
  if (offset.value && root.value && !root.value.contains(e.target as Node)) close()
}
document.addEventListener('pointerdown', onDocDown, true)
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocDown, true)
  if (openRowId.value === props.item.id) openRowId.value = null
})
</script>

<style scoped>
.item-swipe {
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  touch-action: pan-y;
  animation: item-in 0.25s ease-out;
}
.swipe-delete {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  color: #fff;
  background: rgb(var(--v-theme-error));
  border-radius: 0 8px 8px 0;
  transition: width 0.2s ease;
}
.item-row {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  padding: 2px 4px;
  border-radius: 8px;
  transition: opacity 0.28s, background-color 0.15s, transform 0.2s ease;
}
.swiping .item-row,
.swiping .swipe-delete {
  transition: none;
}
.item-row:hover {
  background: rgba(var(--v-theme-on-surface), 0.04);
}
.item-check {
  flex: 0 0 auto;
}
.item-main {
  flex: 1 1 auto;
  min-width: 0;
  padding: 6px 0;
  cursor: pointer;
}
:global(html.density-compact) .item-main {
  padding: 3px 0;
}
.item-title {
  overflow-wrap: anywhere;
  line-height: 1.35;
}
.item-due {
  font-size: 0.78rem;
  line-height: 1.3;
  margin-top: 2px;
  color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
  display: flex;
  align-items: center;
  gap: 3px;
}
.done .item-title,
.leaving .item-title {
  text-decoration: line-through;
  color: rgba(var(--v-theme-on-surface), var(--v-disabled-opacity));
}
.leaving {
  opacity: 0.35;
}
/* Hover delete button: only on devices with a mouse, visible on hover or keyboard focus. */
.hover-delete {
  flex: 0 0 auto;
  align-self: center;
  display: none;
  opacity: 0;
  color: rgb(var(--v-theme-error));
  transition: opacity 0.15s;
}
@media (hover: hover) and (pointer: fine) {
  .hover-delete {
    display: inline-flex;
  }
  .item-row:hover .hover-delete,
  .hover-delete:focus-visible {
    opacity: 1;
  }
}
@keyframes item-in {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}
</style>
