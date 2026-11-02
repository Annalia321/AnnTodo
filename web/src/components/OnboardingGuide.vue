<template>
  <v-dialog :model-value="open" max-width="440" persistent scrollable @keydown.esc="finish">
    <v-card class="guide" rounded="lg">
      <div class="guide-head">
        <span class="text-caption muted">{{ t('onboarding.progress', { n: step + 1, total: steps.length }) }}</span>
        <v-spacer />
        <v-btn v-if="!isLast" variant="text" size="small" class="muted" @click="finish">{{ t('onboarding.skip') }}</v-btn>
      </div>

      <v-window v-model="step" class="guide-window">
        <v-window-item v-for="(st, i) in steps" :key="i" :value="i">
          <div class="guide-body">
            <div class="guide-icon">
              <v-icon :icon="st.icon" size="40" />
            </div>
            <div class="guide-title">{{ t(`onboarding.s${i + 1}Title`) }}</div>
            <div class="guide-text muted">{{ t(`onboarding.s${i + 1}Text`) }}</div>
          </div>
        </v-window-item>
      </v-window>

      <div class="guide-dots" role="tablist">
        <button
          v-for="(_, i) in steps"
          :key="i"
          class="dot"
          :class="{ active: i === step }"
          role="tab"
          :aria-selected="i === step"
          :aria-label="t('onboarding.progress', { n: i + 1, total: steps.length })"
          @click="step = i"
        />
      </div>

      <v-card-actions class="guide-actions">
        <v-btn v-if="step > 0" variant="text" @click="step--">{{ t('onboarding.prev') }}</v-btn>
        <v-spacer />
        <v-btn v-if="!isLast" color="primary" variant="flat" @click="step++">{{ t('onboarding.next') }}</v-btn>
        <v-btn v-else color="primary" variant="flat" @click="finish">{{ t('onboarding.done') }}</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  mdiGestureSwipeLeft,
  mdiGestureTapHold,
  mdiTextBoxEditOutline,
  mdiViewDashboardEditOutline,
  mdiViewGridOutline,
} from '@mdi/js'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'

const { t } = useI18n()
const auth = useAuthStore()
const settings = useSettingsStore()

// Five steps at most. Texts live in locales under `onboarding.s{n}Title` / `onboarding.s{n}Text`.
const steps = [
  { icon: mdiGestureTapHold },
  { icon: mdiViewDashboardEditOutline },
  { icon: mdiViewGridOutline },
  { icon: mdiGestureSwipeLeft },
  { icon: mdiTextBoxEditOutline },
]

const step = ref(0)
const dismissed = ref(false)
const isLast = computed(() => step.value === steps.length - 1)

// Shown once per account: the flag is stored in the user's server-side settings,
// so it does not reappear on other devices after it was finished or skipped.
const open = computed(() => !!auth.user && !settings.s.onboarding_done && !dismissed.value)

watch(
  () => auth.user?.id,
  () => {
    step.value = 0
    dismissed.value = false
  },
)

function finish() {
  dismissed.value = true
  settings.set({ onboarding_done: true })
}
</script>

<style scoped>
.guide-head {
  display: flex;
  align-items: center;
  min-height: 44px;
  padding: 4px 8px 0 20px;
}
.guide-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px 28px 4px;
}
.guide-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  margin-bottom: 20px;
  color: rgb(var(--v-theme-primary));
  background: rgba(var(--v-theme-primary), 0.1);
}
.guide-title {
  font-size: 1.1rem;
  font-weight: 600;
  margin-bottom: 8px;
}
.guide-text {
  font-size: 0.9rem;
  line-height: 1.6;
  min-height: 4.8em;
}
.guide-dots {
  display: flex;
  justify-content: center;
  gap: 8px;
  padding: 12px 0 4px;
}
.dot {
  width: 8px;
  height: 8px;
  padding: 0;
  border-radius: 4px;
  background: rgba(var(--v-theme-on-surface), 0.18);
  transition: width 150ms ease, background-color 150ms ease;
}
.dot.active {
  width: 20px;
  background: rgb(var(--v-theme-primary));
}
.dot:focus-visible {
  outline: 2px solid rgb(var(--v-theme-primary));
  outline-offset: 2px;
}
.guide-actions {
  padding: 8px 16px 16px;
}
</style>
