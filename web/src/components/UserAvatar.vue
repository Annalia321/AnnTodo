<template>
  <v-avatar :size="size" :color="user?.has_avatar ? undefined : 'primary'">
    <v-img v-if="user?.has_avatar" :src="user.avatar_url" :alt="name" cover />
    <span v-else class="initial" :style="{ fontSize: size * 0.42 + 'px' }">{{ initial }}</span>
  </v-avatar>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { User } from '@/api'

const props = withDefaults(defineProps<{ user: User | null; size?: number }>(), { size: 36 })
const name = computed(() => props.user?.nickname || props.user?.username || '')
const initial = computed(() => Array.from(name.value)[0]?.toUpperCase() ?? '?')
</script>

<style scoped>
.initial {
  font-weight: 600;
  color: #fff;
}
</style>
