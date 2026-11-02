<template>
  <div class="auth-page">
    <div class="lang">
      <v-menu location="bottom end">
        <template #activator="{ props: p }">
          <v-btn v-bind="p" variant="text" :prepend-icon="mdiTranslate">{{ currentLabel }}</v-btn>
        </template>
        <v-list density="compact">
          <v-list-item
            v-for="l in languages"
            :key="l.code"
            :title="l.label"
            :active="l.code === language"
            color="primary"
            @click="setGuestLanguage(l.code)"
          />
        </v-list>
      </v-menu>
    </div>
    <v-card class="auth-card" rounded="xl" elevation="3">
      <div class="brand">
        <img src="/icons/icon-192.png" alt="" width="56" height="56" />
        <div class="site">{{ auth.status?.site_name || 'AnnTodo' }}</div>
      </div>
      <slot />
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiTranslate } from '@mdi/js'
import { languages } from '@/locales'
import { setGuestLanguage } from '@/composables/appearance'
import { useAuthStore } from '@/stores/auth'

const { locale } = useI18n()
const auth = useAuthStore()
const language = computed(() => locale.value)
const currentLabel = computed(() => languages.find((l) => l.code === language.value)?.label ?? language.value)
</script>

<style scoped>
.auth-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 56px 16px 24px;
  position: relative;
}
.lang {
  position: absolute;
  top: 8px;
  right: 8px;
}
.auth-card {
  width: 100%;
  max-width: 420px;
  padding: 28px 24px 20px;
}
.brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
}
.site {
  font-size: 1.5rem;
  font-weight: 600;
}
</style>
