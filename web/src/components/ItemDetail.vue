<template>
  <v-navigation-drawer
    v-if="!mobile"
    :model-value="!!item"
    location="right"
    temporary
    :width="420"
    @update:model-value="onModel"
  >
    <ItemDetailBody v-if="item" :key="item.id" :item="item" @close="ui.closeDetail()" />
  </v-navigation-drawer>
  <v-bottom-sheet v-else :model-value="!!item" scrollable @update:model-value="onModel">
    <v-card v-if="item" class="sheet">
      <ItemDetailBody :key="item.id" :item="item" @close="ui.closeDetail()" />
    </v-card>
  </v-bottom-sheet>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useDisplay } from 'vuetify'
import ItemDetailBody from './ItemDetailBody.vue'
import { useTodoStore } from '@/stores/todo'
import { useUiStore } from '@/stores/ui'

const { mobile } = useDisplay()
const ui = useUiStore()
const todo = useTodoStore()

const item = computed(() => (ui.detailId == null ? null : (todo.itemById(ui.detailId) ?? null)))

// The item may disappear (deleted elsewhere, data reloaded); close the panel then.
watch(item, (v) => {
  if (!v && ui.detailId != null) ui.closeDetail()
})

function onModel(v: boolean) {
  if (!v) ui.closeDetail()
}
</script>

<style scoped>
.sheet {
  max-height: 88dvh;
  border-top-left-radius: 16px !important;
  border-top-right-radius: 16px !important;
}
</style>
