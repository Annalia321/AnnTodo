<template>
  <v-snackbar
    v-if="current"
    :key="current.id"
    :model-value="true"
    :timeout="current.timeout ?? 4000"
    :color="current.color"
    location="bottom"
    @update:model-value="(v: boolean) => !v && ui.dismiss(current!.id)"
  >
    {{ current.text }}
    <template v-if="current.actionText" #actions>
      <v-btn variant="text" @click="runAction">{{ current.actionText }}</v-btn>
    </template>
  </v-snackbar>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const current = computed(() => ui.toasts[0])

function runAction() {
  const t = current.value
  if (!t) return
  ui.dismiss(t.id)
  t.action?.()
}
</script>
