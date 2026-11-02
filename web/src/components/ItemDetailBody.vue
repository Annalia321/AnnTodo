<template>
  <div class="detail">
    <div class="detail-head">
      <v-checkbox-btn :model-value="item.completed" color="primary" @update:model-value="(v: unknown) => todo.setCompleted(item.id, !!v)" />
      <span class="text-subtitle-1 font-weight-medium">{{ t('item.details') }}</span>
      <v-spacer />
      <v-btn :icon="mdiClose" variant="text" size="small" :title="t('common.close')" @click="emit('close')" />
    </div>

    <div class="detail-body">
      <v-textarea
        v-model="title"
        :label="t('item.title')"
        variant="outlined"
        rows="1"
        auto-grow
        hide-details="auto"
        :error-messages="title.trim() ? [] : [t('common.required')]"
        class="mb-4 title-field"
        @update:model-value="schedule"
        @blur="flush"
        @keydown.enter.prevent="flush"
      />
      <v-textarea
        v-model="note"
        :label="t('item.note')"
        variant="outlined"
        rows="3"
        auto-grow
        hide-details
        class="mb-4"
        @update:model-value="schedule"
        @blur="flush"
      />

      <div class="section-label">{{ t('item.dueDate') }}</div>
      <div class="d-flex ga-2 align-center mb-4">
        <v-text-field
          :model-value="due?.date ?? ''"
          type="date"
          variant="outlined"
          hide-details
          :label="t('item.dueDate')"
          @update:model-value="(v: string) => setDue(v || null, v ? (due?.time ?? null) : null)"
        />
        <v-text-field
          :model-value="due?.time ?? ''"
          type="time"
          variant="outlined"
          hide-details
          :disabled="!due"
          :label="t('item.dueTime')"
          style="max-width: 150px"
          @update:model-value="(v: string) => setDue(due?.date ?? null, v || null)"
        />
        <v-btn
          v-if="due"
          :icon="mdiCloseCircleOutline"
          variant="text"
          size="small"
          :title="t('item.clearDue')"
          @click="setDue(null, null)"
        />
      </div>
      <div v-if="due && overdue" class="overdue text-caption mt-n3 mb-3">{{ formatDue(item) }}</div>

      <div class="section-label">{{ t('item.quadrant') }}</div>
      <div class="d-flex flex-wrap ga-4 mb-2">
        <v-switch
          :model-value="item.important"
          color="primary"
          :label="t('item.important')"
          hide-details
          inset
          @update:model-value="(v: unknown) => todo.updateItem(item.id, { important: !!v })"
        />
        <v-switch
          :model-value="item.urgent"
          color="primary"
          :label="t('item.urgent')"
          hide-details
          inset
          @update:model-value="(v: unknown) => todo.updateItem(item.id, { urgent: !!v })"
        />
      </div>
      <v-select
        :model-value="quadrant"
        :items="quadrants"
        variant="outlined"
        hide-details
        class="mb-4"
        @update:model-value="setQuadrant"
      />

      <div class="section-label">{{ t('item.card') }}</div>
      <v-select
        :model-value="item.card_id"
        :items="cardOptions"
        variant="outlined"
        hide-details
        class="mb-4"
        @update:model-value="(v: number) => todo.moveItem(item.id, v)"
      >
        <template #selection="{ item: opt }">
          <span class="dot" :style="{ background: opt.raw.color }" />{{ opt.title }}
        </template>
        <template #item="{ props: p, item: opt }">
          <v-list-item v-bind="p">
            <template #prepend><span class="dot" :style="{ background: opt.raw.color }" /></template>
          </v-list-item>
        </template>
      </v-select>

      <div class="text-caption muted">
        <div v-if="item.completed_at">{{ t('item.completedAt', { t: fmtDateTime(item.completed_at) }) }}</div>
        <div>{{ t('item.createdAt', { t: fmtDateTime(item.created_at) }) }}</div>
      </div>
    </div>

    <div class="detail-foot">
      <v-btn color="error" variant="tonal" :prepend-icon="mdiDeleteOutline" block @click="remove">
        {{ t('item.delete') }}
      </v-btn>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiClose, mdiCloseCircleOutline, mdiDeleteOutline } from '@mdi/js'
import type { Item } from '@/api'
import { useTodoStore, type ItemPatch } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor } from '@/utils/cards'
import { fmtDateTime, formatDue, isOverdue, localDue, toStoredDue } from '@/utils/time'

const props = defineProps<{ item: Item }>()
const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const todo = useTodoStore()
const ui = useUiStore()

const title = ref(props.item.title)
const note = ref(props.item.note ?? '')
let timer: ReturnType<typeof setTimeout> | undefined

function schedule() {
  clearTimeout(timer)
  timer = setTimeout(flush, 600)
}

function flush() {
  clearTimeout(timer)
  timer = undefined
  const patch: ItemPatch = {}
  const tt = title.value.trim()
  if (tt && tt !== props.item.title) patch.title = tt
  if (note.value !== (props.item.note ?? '')) patch.note = note.value
  if (Object.keys(patch).length) todo.updateItem(props.item.id, patch)
}

onBeforeUnmount(() => {
  flush()
})

const due = computed(() => localDue(props.item))
const overdue = computed(() => isOverdue(props.item))

function setDue(date: string | null, time: string | null) {
  todo.updateItem(props.item.id, toStoredDue(date, time))
}

const quadrants = computed(() => [
  { title: t('quadrant.q1'), value: 'q1' },
  { title: t('quadrant.q2'), value: 'q2' },
  { title: t('quadrant.q3'), value: 'q3' },
  { title: t('quadrant.q4'), value: 'q4' },
])

const quadrant = computed(() => {
  const { important, urgent } = props.item
  if (important) return urgent ? 'q1' : 'q2'
  return urgent ? 'q3' : 'q4'
})

function setQuadrant(q: string) {
  todo.updateItem(props.item.id, { important: q === 'q1' || q === 'q2', urgent: q === 'q1' || q === 'q3' })
}

const cardOptions = computed(() =>
  todo.sortedCards.map((c) => ({ title: c.name, value: c.id, color: cardColor(c) })),
)

async function remove() {
  clearTimeout(timer)
  const id = props.item.id
  emit('close')
  await todo.deleteItem(id)
  ui.notify(t('item.deleted'), { actionText: t('common.undo'), action: () => todo.restoreItem(id), timeout: 6000 })
}
</script>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.detail-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 8px 8px 12px;
  border-bottom: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.detail-body {
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 16px;
}
.detail-foot {
  padding: 12px 16px;
  border-top: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
.section-label {
  font-size: 0.8rem;
  font-weight: 600;
  margin-bottom: 6px;
  color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
}
.title-field :deep(textarea) {
  font-size: 1.1rem;
  font-weight: 500;
}
.dot {
  display: inline-block;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  margin-right: 8px;
}
</style>
