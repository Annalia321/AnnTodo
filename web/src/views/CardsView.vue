<template>
  <div class="page">
    <div class="page-head">
      <v-btn :icon="mdiArrowLeft" variant="text" :title="t('common.back')" to="/" />
      <span class="page-title">{{ t('cards.title') }}</span>
      <v-spacer />
      <span class="text-caption muted d-none d-sm-inline">{{ t('cards.dragHint') }}</span>
    </div>

    <draggable
      :list="list"
      item-key="id"
      :animation="150"
      ghost-class="ghost"
      :delay="150"
      :delay-on-touch-only="true"
      filter=".no-drag"
      :prevent-on-filter="false"
      class="mini-grid"
      @end="onEnd"
    >
      <template #item="{ element }">
        <div class="mini-card" :style="{ '--card-color': cardColor(element) }">
          <div class="strip" />
          <div class="mini-name">{{ element.name }}</div>
          <div class="mini-foot">
            <span class="text-caption muted">{{ todo.openCount(element.id) }}</span>
            <v-spacer />
            <v-menu location="bottom end">
              <template #activator="{ props: p }">
                <v-btn v-bind="p" :icon="mdiDotsHorizontal" variant="text" size="small" class="no-drag" />
              </template>
              <v-list min-width="180">
                <v-list-item :prepend-icon="mdiPencilOutline" :title="t('cards.rename')" @click="openRename(element)" />
                <v-list-item :prepend-icon="mdiPaletteOutline" :title="t('cards.changeColor')" @click="coloring = element" />
                <v-list-item
                  :prepend-icon="mdiDeleteOutline"
                  :title="t('common.delete')"
                  base-color="error"
                  @click="deleting = element"
                />
              </v-list>
            </v-menu>
          </div>
        </div>
      </template>
      <template #footer>
        <button class="mini-card new-card no-drag" @click="createOpen = true">
          <v-icon :icon="mdiPlus" />
          <span>{{ t('cards.newCard') }}</span>
        </button>
      </template>
    </draggable>

    <CardNameDialog v-model="createOpen" :title="t('cards.newCard')" with-color :initial-color="nextCardColor(list.length)" @submit="create" />
    <CardNameDialog
      v-model="renameOpen"
      :title="t('cards.rename')"
      :initial="renaming?.name"
      @submit="(name: string) => renaming && save(renaming, { name })"
    />

    <v-dialog :model-value="!!coloring" max-width="340" @update:model-value="(v: boolean) => !v && (coloring = null)">
      <v-card v-if="coloring" :title="t('cards.changeColor')" :subtitle="coloring.name">
        <v-card-text class="d-flex flex-wrap ga-3">
          <span
            v-for="c in cardColors"
            :key="c"
            class="swatch"
            :class="{ selected: c === cardColor(coloring) }"
            :style="{ background: c }"
            @click="setColor(coloring, c)"
          />
        </v-card-text>
      </v-card>
    </v-dialog>

    <v-dialog :model-value="!!deleting" max-width="440" @update:model-value="(v: boolean) => !v && (deleting = null)">
      <v-card v-if="deleting" :title="t('cards.deleteTitle', { name: deleting.name })">
        <v-card-text>{{ t('cards.deleteHint', { n: todo.itemCount(deleting.id) }) }}</v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleting = null">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="remove">{{ t('common.delete') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import draggable from 'vuedraggable'
import {
  mdiArrowLeft,
  mdiDeleteOutline,
  mdiDotsHorizontal,
  mdiPaletteOutline,
  mdiPencilOutline,
  mdiPlus,
} from '@mdi/js'
import CardNameDialog from '@/components/CardNameDialog.vue'
import type { Card } from '@/api'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor, cardColors, nextCardColor } from '@/utils/cards'

const { t } = useI18n()
const todo = useTodoStore()
const ui = useUiStore()

const list = ref<Card[]>([])
watch(
  () => todo.sortedCards,
  (v) => (list.value = [...v]),
  { immediate: true },
)

function onEnd(e: { oldIndex?: number; newIndex?: number }) {
  if (e.oldIndex === e.newIndex) return
  todo.reorderCards(list.value.map((c) => c.id))
}

const createOpen = ref(false)
const renameOpen = ref(false)
const renaming = ref<Card | null>(null)
const deleting = ref<Card | null>(null)
const coloring = ref<Card | null>(null)

async function create(name: string, color: string) {
  try {
    const card = await todo.createCard(name, color)
    if (todo.cards.length === 1) await todo.setCurrentCard(card.id)
  } catch (e) {
    ui.error(e)
  }
}

function openRename(c: Card) {
  renaming.value = c
  renameOpen.value = true
}

async function save(c: Card, patch: Partial<Pick<Card, 'name' | 'color'>>) {
  try {
    await todo.updateCard(c.id, patch)
  } catch (e) {
    ui.error(e)
  }
}

function setColor(c: Card, color: string) {
  coloring.value = null
  save(c, { color })
}

async function remove() {
  const c = deleting.value
  deleting.value = null
  if (!c) return
  try {
    await todo.deleteCard(c.id)
  } catch (e) {
    ui.error(e)
  }
}
</script>

<style scoped>
.mini-grid {
  display: grid;
  gap: var(--gap);
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
}
.mini-card {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 112px;
  border-radius: 12px;
  overflow: hidden;
  background: rgb(var(--v-theme-surface));
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  cursor: grab;
  user-select: none;
}
.strip {
  height: 5px;
  background: var(--card-color);
}
.mini-name {
  flex: 1 1 auto;
  padding: 10px 12px 0;
  font-size: 1.1rem;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.mini-foot {
  display: flex;
  align-items: center;
  padding: 0 4px 4px 12px;
}
.new-card {
  align-items: center;
  justify-content: center;
  gap: 4px;
  cursor: pointer;
  border: 2px dashed rgba(var(--v-theme-on-surface), 0.2);
  background: transparent;
  box-shadow: none;
  color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
  font: inherit;
}
.new-card:hover {
  border-color: rgb(var(--v-theme-primary));
  color: rgb(var(--v-theme-primary));
}
</style>
