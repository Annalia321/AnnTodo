<template>
  <v-dialog :model-value="modelValue" max-width="420" @update:model-value="(v: boolean) => emit('update:modelValue', v)">
    <v-card :title="title">
      <v-card-text>
        <v-text-field
          v-model="name"
          :label="t('cards.cardName')"
          variant="outlined"
          autofocus
          maxlength="100"
          hide-details
          @keydown.enter="submit"
        />
        <div v-if="withColor" class="d-flex flex-wrap ga-2 mt-4">
          <span
            v-for="c in cardColors"
            :key="c"
            class="swatch"
            :class="{ selected: c === color }"
            :style="{ background: c }"
            @click="color = c"
          />
        </div>
      </v-card-text>
      <v-card-actions>
        <v-spacer />
        <v-btn variant="text" @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</v-btn>
        <v-btn color="primary" variant="flat" :disabled="!name.trim()" @click="submit">{{ t('common.save') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { cardColors } from '@/utils/cards'

const props = defineProps<{ modelValue: boolean; title: string; initial?: string; initialColor?: string; withColor?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [boolean]; submit: [name: string, color: string] }>()
const { t } = useI18n()

const name = ref('')
const color = ref('')
watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      name.value = props.initial ?? ''
      color.value = props.initialColor ?? cardColors[0]
    }
  },
  { immediate: true },
)

function submit() {
  const n = name.value.trim()
  if (!n) return
  emit('submit', n, color.value)
  emit('update:modelValue', false)
}
</script>
