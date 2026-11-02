<template>
  <v-navigation-drawer v-if="!mobile" permanent rail :rail-width="76" class="shell-rail">
    <div class="rail-inner">
      <div class="logo" :title="siteName">
        <img src="/icons/icon-192.png" alt="" width="36" height="36" />
      </div>
      <nav class="rail-nav">
        <router-link
          v-for="n in nav"
          :key="n.to"
          :to="n.to"
          class="rail-link"
          :class="{ active: isActive(n.to) }"
          :title="t(n.label)"
        >
          <v-icon :icon="n.icon" />
          <span class="rail-label">{{ t(n.label) }}</span>
        </router-link>
      </nav>
      <v-spacer />
      <v-menu location="end bottom">
        <template #activator="{ props: p }">
          <button v-bind="p" class="avatar-btn" :title="auth.displayName">
            <UserAvatar :user="auth.user" :size="40" />
          </button>
        </template>
        <v-list min-width="200">
          <v-list-item :title="auth.displayName" :subtitle="auth.user?.username" />
          <v-divider />
          <v-list-item :prepend-icon="mdiAccountCircleOutline" :title="t('nav.profile')" to="/settings#account" />
          <v-list-item v-if="auth.isAdmin" :prepend-icon="mdiShieldAccountOutline" :title="t('nav.admin')" to="/admin" />
          <v-list-item :prepend-icon="mdiLogout" :title="t('nav.logout')" @click="logout" />
        </v-list>
      </v-menu>
    </div>
  </v-navigation-drawer>

  <v-main>
    <slot />
  </v-main>

  <v-bottom-navigation v-if="mobile" grow :model-value="activeNav" color="primary">
    <v-btn v-for="n in nav" :key="n.to" :value="n.to" :to="n.to">
      <v-icon :icon="n.icon" />
      <span>{{ t(n.label) }}</span>
    </v-btn>
  </v-bottom-navigation>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useDisplay } from 'vuetify'
import {
  mdiAccountCircleOutline,
  mdiCalendarMonthOutline,
  mdiCogOutline,
  mdiFormatListChecks,
  mdiLogout,
  mdiShieldAccountOutline,
  mdiViewGridOutline,
} from '@mdi/js'
import UserAvatar from './UserAvatar.vue'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const { mobile } = useDisplay()
const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const siteName = computed(() => auth.status?.site_name || 'AnnTodo')

const nav = [
  { to: '/', icon: mdiFormatListChecks, label: 'nav.list' },
  { to: '/calendar', icon: mdiCalendarMonthOutline, label: 'nav.calendar' },
  { to: '/quadrant', icon: mdiViewGridOutline, label: 'nav.quadrant' },
  { to: '/settings', icon: mdiCogOutline, label: 'nav.settings' },
]

function isActive(to: string) {
  if (to === '/') return route.path === '/' || route.path === '/cards'
  return route.path.startsWith(to)
}

const activeNav = computed(() => nav.find((n) => isActive(n.to))?.to)

async function logout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<style scoped>
.rail-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  padding: 12px 0;
}
.logo {
  margin-bottom: 16px;
}
.rail-nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  align-items: center;
}
.rail-link {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  width: 64px;
  padding: 8px 0;
  border-radius: 12px;
  color: rgba(var(--v-theme-on-surface), 0.72);
  text-decoration: none;
  transition: background-color 0.15s;
}
.rail-link:hover {
  background: rgba(var(--v-theme-on-surface), 0.06);
}
.rail-link.active {
  color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.12);
}
.rail-label {
  font-size: 0.72rem;
  line-height: 1.1;
  text-align: center;
}
.avatar-btn {
  border-radius: 50%;
  margin-top: 8px;
}
</style>
