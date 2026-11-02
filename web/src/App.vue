<template>
  <v-app>
    <v-defaults-provider :defaults="defaults">
      <template v-if="route.meta.guest">
        <v-main>
          <router-view />
        </v-main>
      </template>
      <template v-else-if="auth.user">
        <AppShell>
          <router-view />
        </AppShell>
        <ItemDetail />
        <OnboardingGuide />
      </template>
      <v-main v-else>
        <div class="d-flex justify-center align-center" style="height: 60vh">
          <v-progress-circular indeterminate color="primary" />
        </div>
      </v-main>
      <ToastHost />
    </v-defaults-provider>
  </v-app>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import AppShell from '@/components/AppShell.vue'
import ItemDetail from '@/components/ItemDetail.vue'
import OnboardingGuide from '@/components/OnboardingGuide.vue'
import ToastHost from '@/components/ToastHost.vue'
import { useAppearance } from '@/composables/appearance'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const { defaults } = useAppearance()
</script>
