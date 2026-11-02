<template>
  <div
    class="list-page page"
    :class="{ pressing }"
    @pointerdown="pressStart"
    @pointermove="pressMove"
    @pointerup="pressCancel"
    @pointercancel="pressCancel"
    @pointerleave="pressCancel"
    @contextmenu="onContextMenu"
  >
    <div class="page-head" data-blank>
      <v-spacer data-blank />
      <v-btn
        :icon="mdiViewDashboardEditOutline"
        variant="text"
        size="small"
        class="muted"
        :title="t('list.manageCards') + ' · ' + t('list.longPressHint')"
        to="/cards"
      />
    </div>

    <div v-if="!todo.cards.length" class="empty-state">
      <div class="muted mb-4">{{ t('list.noCards') }}</div>
      <v-btn color="primary" size="large" :prepend-icon="mdiPlus" @click="dialog = true">{{ t('list.createFirst') }}</v-btn>
    </div>

    <div v-else class="card-grid" data-blank>
      <TodoCard
        v-for="c in todo.sortedCards"
        :key="c.id"
        :card="c"
        :current="c.id === todo.currentCardId"
        :collapsed="mobile && c.id !== todo.currentCardId"
      />
    </div>

    <CardNameDialog v-model="dialog" :title="t('cards.newCard')" with-color @submit="createFirst" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import { mdiPlus, mdiViewDashboardEditOutline } from '@mdi/js'
import CardNameDialog from '@/components/CardNameDialog.vue'
import TodoCard from '@/components/TodoCard.vue'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'

const { t } = useI18n()
const { mobile } = useDisplay()
const router = useRouter()
const todo = useTodoStore()
const ui = useUiStore()

const dialog = ref(false)

async function createFirst(name: string, color: string) {
  try {
    const card = await todo.createCard(name, color)
    await todo.setCurrentCard(card.id)
  } catch (e) {
    ui.error(e)
  }
}

// ----- long press on blank space opens the card editor -----
const LONG_PRESS_MS = 500
const pressing = ref(false)
let timer: ReturnType<typeof setTimeout> | undefined
let startX = 0
let startY = 0

function isBlank(target: EventTarget | null, current: EventTarget | null) {
  return target === current || (target instanceof HTMLElement && target.hasAttribute('data-blank'))
}

function pressStart(e: PointerEvent) {
  if (e.button !== 0 || !isBlank(e.target, e.currentTarget)) return
  startX = e.clientX
  startY = e.clientY
  pressing.value = true
  timer = setTimeout(() => {
    pressing.value = false
    router.push('/cards')
  }, LONG_PRESS_MS)
}

function pressMove(e: PointerEvent) {
  if (pressing.value && Math.hypot(e.clientX - startX, e.clientY - startY) > 10) pressCancel()
}

function pressCancel() {
  clearTimeout(timer)
  pressing.value = false
}

function onContextMenu(e: Event) {
  // Touch long press fires a context menu on some browsers.
  if (isBlank(e.target, e.currentTarget)) e.preventDefault()
}

onBeforeUnmount(pressCancel)
</script>

<style scoped>
.list-page {
  min-height: 100%;
  position: relative;
  -webkit-touch-callout: none;
}
.list-page::after {
  content: '';
  position: fixed;
  inset: 0;
  pointer-events: none;
  background: rgb(0, 0, 0);
  opacity: 0;
  transition: opacity 0.15s;
  z-index: 5;
}
.list-page.pressing::after {
  opacity: 0.14;
  transition: opacity 0.5s linear;
}
.card-grid {
  display: grid;
  gap: var(--gap);
  grid-template-columns: 1fr;
  align-items: start;
  min-height: 60vh;
}
@media (min-width: 600px) {
  .card-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .card-grid > :deep(.current) {
    grid-column: span 2;
  }
}
@media (min-width: 1150px) {
  .card-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (min-width: 1600px) {
  .card-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}
</style>
