<template>
  <v-card id="data" class="section" rounded="lg">
    <v-card-title>{{ t('settings.data') }}</v-card-title>
    <v-card-text>
      <div class="d-flex flex-wrap ga-3">
        <v-btn variant="tonal" color="primary" :prepend-icon="mdiDownload" href="/api/v1/data/export" download>{{ t('settings.export') }}</v-btn>
        <v-btn variant="tonal" :prepend-icon="mdiUpload" :loading="importBusy" @click="fileInput?.click()">{{ t('settings.import') }}</v-btn>
        <input ref="fileInput" type="file" accept="application/json,.json" hidden @change="importFile" />
      </div>

      <v-divider class="my-6" />

      <div id="trash" class="d-flex align-center flex-wrap ga-2 mb-1">
        <div class="text-subtitle-1 font-weight-medium">{{ t('settings.trash') }}</div>
        <v-spacer />
        <v-btn v-if="!trashEmpty" variant="text" color="error" size="small" @click="confirmEmpty = true">{{ t('settings.emptyTrash') }}</v-btn>
      </div>
      <div class="muted text-body-2 mb-2">{{ t('settings.trashHint') }}</div>

      <div v-if="trashEmpty" class="muted py-4">{{ t('settings.trashEmpty') }}</div>
      <v-list v-else density="compact" class="bg-transparent">
        <v-list-item v-for="c in trash.cards" :key="'c' + c.id" class="px-0">
          <template #prepend>
            <span class="dot mr-3" :style="{ background: cardColor(c) }" />
          </template>
          <v-list-item-title>{{ c.name }}</v-list-item-title>
          <v-list-item-subtitle>
            {{ t('settings.trashCard', { n: c.item_count }) }} · {{ t('settings.deletedAt', { t: fmtDateTime(c.deleted_at) }) }}
          </v-list-item-subtitle>
          <template #append>
            <v-btn variant="text" size="small" color="primary" @click="restore('card', c.id)">{{ t('common.restore') }}</v-btn>
            <v-btn variant="text" size="small" color="error" @click="purge('card', c.id)">{{ t('settings.deleteForever') }}</v-btn>
          </template>
        </v-list-item>
        <v-list-item v-for="i in trash.items" :key="'i' + i.id" class="px-0">
          <template #prepend>
            <v-icon :icon="mdiCheckboxBlankOutline" class="mr-3" size="small" />
          </template>
          <v-list-item-title>{{ i.title }}</v-list-item-title>
          <v-list-item-subtitle>
            {{ t('settings.trashItem', { card: i.card_name }) }} · {{ t('settings.deletedAt', { t: fmtDateTime(i.deleted_at) }) }}
          </v-list-item-subtitle>
          <template #append>
            <v-btn variant="text" size="small" color="primary" @click="restore('item', i.id)">{{ t('common.restore') }}</v-btn>
            <v-btn variant="text" size="small" color="error" @click="purge('item', i.id)">{{ t('settings.deleteForever') }}</v-btn>
          </template>
        </v-list-item>
      </v-list>
    </v-card-text>

    <v-dialog v-model="confirmEmpty" max-width="420">
      <v-card :title="t('settings.emptyTrash')" :text="t('settings.emptyTrashConfirm')">
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirmEmpty = false">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="emptyTrash">{{ t('common.delete') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiCheckboxBlankOutline, mdiDownload, mdiUpload } from '@mdi/js'
import { api, request, type TrashData } from '@/api'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'
import { cardColor } from '@/utils/cards'
import { fmtDateTime } from '@/utils/time'

const { t } = useI18n()
const todo = useTodoStore()
const ui = useUiStore()

// ----- import -----
const fileInput = ref<HTMLInputElement | null>(null)
const importBusy = ref(false)
async function importFile(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  importBusy.value = true
  try {
    const text = await file.text()
    const res = await request<{ cards: number; items: number }>('POST', '/data/import', text, true)
    ui.notify(t('settings.imported', { cards: res.cards, items: res.items }))
    await todo.load()
  } catch (e) {
    ui.error(e)
  } finally {
    importBusy.value = false
  }
}

// ----- trash -----
const trash = ref<TrashData>({ cards: [], items: [] })
const trashEmpty = computed(() => !trash.value.cards.length && !trash.value.items.length)
const confirmEmpty = ref(false)

async function loadTrash() {
  try {
    const d = await api.get<TrashData>('/trash')
    trash.value = { cards: d?.cards ?? [], items: d?.items ?? [] }
  } catch (e) {
    ui.error(e)
  }
}

async function restore(type: 'card' | 'item', id: number) {
  try {
    await api.post(`/trash/${type}/${id}/restore`)
    await Promise.all([loadTrash(), todo.load()])
  } catch (e) {
    ui.error(e)
  }
}

async function purge(type: 'card' | 'item', id: number) {
  try {
    await api.del(`/trash/${type}/${id}`)
    await loadTrash()
  } catch (e) {
    ui.error(e)
  }
}

async function emptyTrash() {
  confirmEmpty.value = false
  try {
    await api.del('/trash')
    trash.value = { cards: [], items: [] }
  } catch (e) {
    ui.error(e)
  }
}

onMounted(loadTrash)
</script>

<style scoped>
.dot {
  display: inline-block;
  width: 14px;
  height: 14px;
  border-radius: 50%;
}
</style>
