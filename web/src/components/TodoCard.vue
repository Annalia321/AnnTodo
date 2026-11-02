<template>
  <v-card
    class="todo-card"
    :class="{ current, collapsed }"
    :style="{ '--card-color': color }"
    :elevation="current ? 3 : 1"
    rounded="lg"
  >
    <div class="strip" />
    <div class="card-head" @click="todo.setCurrentCard(card.id)">
      <span class="card-name">{{ card.name }}</span>
      <v-spacer />
      <span class="count" :title="String(open.length)">{{ open.length }}</span>
    </div>

    <template v-if="!collapsed">
      <draggable
        :list="openList"
        item-key="id"
        group="items"
        :animation="150"
        ghost-class="ghost"
        :delay="200"
        :delay-on-touch-only="true"
        class="item-list"
        @change="onChange"
      >
        <template #item="{ element }">
          <ItemRow :item="element" :color="color" />
        </template>
      </draggable>
      <div v-if="!openList.length" class="empty muted">{{ t('list.empty') }}</div>

      <div class="add-row">
        <v-icon :icon="mdiPlus" size="20" class="muted" />
        <input
          v-model="newTitle"
          class="add-input"
          :placeholder="t('list.addItem')"
          maxlength="500"
          @keydown.enter="add"
        />
      </div>

      <template v-if="done.length">
        <button class="done-toggle muted" @click="showDone = !showDone">
          <v-icon :icon="showDone ? mdiChevronDown : mdiChevronRight" size="18" />
          {{ t('list.completed', { n: done.length }) }}
        </button>
        <v-expand-transition>
          <div v-show="showDone" class="done-list">
            <ItemRow v-for="it in done" :key="it.id" :item="it" :color="color" />
          </div>
        </v-expand-transition>
      </template>
    </template>
  </v-card>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import draggable from 'vuedraggable'
import { mdiChevronDown, mdiChevronRight, mdiPlus } from '@mdi/js'
import ItemRow from './ItemRow.vue'
import type { Card, Item } from '@/api'
import { useSettingsStore } from '@/stores/settings'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor } from '@/utils/cards'

const props = defineProps<{ card: Card; current: boolean; collapsed: boolean }>()
const { t } = useI18n()
const todo = useTodoStore()
const settings = useSettingsStore()
const ui = useUiStore()

const color = computed(() => cardColor(props.card))
const open = computed(() => todo.openItems(props.card.id))
const done = computed(() => todo.doneItems(props.card.id))

// vuedraggable mutates the array it is given, so it works on a local copy.
const openList = ref<Item[]>([])
watch(open, (v) => (openList.value = [...v]), { immediate: true })

const showDone = ref(settings.s.expand_completed)

const newTitle = ref('')
let adding = false
async function add(e: KeyboardEvent) {
  if (e.isComposing || adding) return
  const title = newTitle.value.trim()
  if (!title) return
  adding = true
  try {
    await todo.createItem({ card_id: props.card.id, title })
    newTitle.value = ''
  } catch (err) {
    ui.error(err)
  } finally {
    adding = false
  }
}

function onChange(e: { added?: unknown; moved?: unknown; removed?: unknown }) {
  // "removed" is handled by the receiving card.
  if (e.added || e.moved) {
    todo.reorderList(
      props.card.id,
      openList.value.map((i) => i.id),
    )
  }
}
</script>

<style scoped>
.todo-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 2px solid transparent;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.todo-card.current {
  border-color: rgb(var(--v-theme-primary));
}
.strip {
  height: 5px;
  background: var(--card-color);
  flex: 0 0 auto;
}
.card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px 6px;
  cursor: pointer;
  user-select: none;
}
.collapsed .card-head {
  padding: 10px 14px;
}
.card-name {
  font-size: 1.2rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.current .card-name {
  font-size: 1.35rem;
}
.count {
  min-width: 26px;
  height: 22px;
  padding: 0 7px;
  border-radius: 11px;
  font-size: 0.8rem;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: var(--card-color);
}
.item-list {
  min-height: 12px;
  max-height: 340px;
  overflow-y: auto;
  padding: 0 6px;
}
.current .item-list {
  max-height: 620px;
}
.empty {
  font-size: 0.85rem;
  padding: 4px 16px 8px;
}
.add-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 4px 10px 8px;
  padding: 6px 8px;
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
.done-toggle {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 12px 8px;
  font-size: 0.85rem;
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
}
.done-list {
  padding: 0 6px 8px;
}
</style>
