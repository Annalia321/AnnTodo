<template>
  <div class="page-narrow settings">
    <div class="page-head">
      <span class="page-title">{{ t('settings.title') }}</span>
    </div>

    <!-- Language -->
    <v-card id="language" class="section" rounded="lg">
      <v-card-title>{{ t('settings.language') }}</v-card-title>
      <v-card-text>
        <v-select
          :model-value="s.language"
          :items="languageOptions"
          variant="outlined"
          density="compact"
          hide-details
          class="set-select"
          @update:model-value="(v: string) => settings.set({ language: v })"
        />
      </v-card-text>
    </v-card>

    <!-- Appearance -->
    <v-card id="appearance" class="section" rounded="lg">
      <v-card-title>{{ t('settings.appearance') }}</v-card-title>
      <v-card-text>
        <div class="row set-row">
          <div class="label">{{ t('settings.theme') }}</div>
          <v-btn-toggle :model-value="s.theme" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: UserSettings['theme']) => settings.set({ theme: v })">
            <v-btn value="light" :prepend-icon="mdiWhiteBalanceSunny">{{ t('settings.themeLight') }}</v-btn>
            <v-btn value="dark" :prepend-icon="mdiWeatherNight">{{ t('settings.themeDark') }}</v-btn>
            <v-btn value="system" :prepend-icon="mdiThemeLightDark">{{ t('settings.themeSystem') }}</v-btn>
          </v-btn-toggle>
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.accent') }}</div>
          <div class="d-flex flex-wrap ga-2">
            <span
              v-for="c in accentPresets"
              :key="c"
              class="swatch"
              :class="{ selected: c === s.accent }"
              :style="{ background: c }"
              @click="settings.set({ accent: c })"
            >
              <v-icon v-if="c === s.accent" :icon="mdiCheck" size="16" />
            </span>
          </div>
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.font') }}</div>
          <div class="set-col">
            <v-select
              :model-value="s.font"
              :items="fontOptions"
              variant="outlined"
              density="compact"
              hide-details
              class="set-select"
              @update:model-value="(v: UserSettings['font']) => settings.set({ font: v })"
            />
            <v-text-field
              v-if="s.font === 'custom'"
              v-model="customFont"
              :label="t('settings.customFontName')"
              placeholder="LXGW WenKai"
              variant="outlined"
              hide-details
              class="mt-3"
              @blur="saveCustomFont"
              @keydown.enter="saveCustomFont"
            />
          </div>
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.fontSize') }}</div>
          <v-btn-toggle :model-value="s.font_size" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: UserSettings['font_size']) => settings.set({ font_size: v })">
            <v-btn value="s">{{ t('settings.fontS') }}</v-btn>
            <v-btn value="m">{{ t('settings.fontM') }}</v-btn>
            <v-btn value="l">{{ t('settings.fontL') }}</v-btn>
            <v-btn value="xl">{{ t('settings.fontXL') }}</v-btn>
          </v-btn-toggle>
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.density') }}</div>
          <v-btn-toggle :model-value="s.density" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: UserSettings['density']) => settings.set({ density: v })">
            <v-btn value="comfortable">{{ t('settings.comfortable') }}</v-btn>
            <v-btn value="compact">{{ t('settings.compact') }}</v-btn>
          </v-btn-toggle>
        </div>
      </v-card-text>
    </v-card>

    <!-- Date & time -->
    <v-card id="datetime" class="section" rounded="lg">
      <v-card-title>{{ t('settings.datetime') }}</v-card-title>
      <v-card-text>
        <div class="row set-row">
          <div class="label">{{ t('settings.timezone') }}</div>
          <v-autocomplete
            :model-value="s.timezone"
            :items="timezoneOptions"
            variant="outlined"
            density="compact"
            hide-details
            class="set-select"
            @update:model-value="(v: string | null) => settings.set({ timezone: v ?? '' })"
          />
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.weekStart') }}</div>
          <v-btn-toggle :model-value="s.week_start" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: 0 | 1) => settings.set({ week_start: v })">
            <v-btn :value="0">{{ t('settings.sunday') }}</v-btn>
            <v-btn :value="1">{{ t('settings.monday') }}</v-btn>
          </v-btn-toggle>
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.dateFormat') }}</div>
          <v-select
            :model-value="s.date_format"
            :items="dateFormats"
            variant="outlined"
            density="compact"
            hide-details
            class="set-select"
            @update:model-value="(v: UserSettings['date_format']) => settings.set({ date_format: v })"
          />
        </div>
        <div class="row set-row">
          <div class="label">{{ t('settings.timeFormat') }}</div>
          <v-btn-toggle :model-value="s.time_format" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: UserSettings['time_format']) => settings.set({ time_format: v })">
            <v-btn value="24h">{{ t('settings.h24') }}</v-btn>
            <v-btn value="12h">{{ t('settings.h12') }}</v-btn>
          </v-btn-toggle>
        </div>
      </v-card-text>
    </v-card>

    <!-- Tasks -->
    <v-card id="tasks" class="section" rounded="lg">
      <v-card-title>{{ t('settings.tasks') }}</v-card-title>
      <v-card-text>
        <div class="row set-row">
          <div class="label">{{ t('settings.newItemPosition') }}</div>
          <v-btn-toggle :model-value="s.new_item_position" class="set-seg" mandatory color="primary" variant="text" divided @update:model-value="(v: UserSettings['new_item_position']) => settings.set({ new_item_position: v })">
            <v-btn value="top">{{ t('settings.top') }}</v-btn>
            <v-btn value="bottom">{{ t('settings.bottom') }}</v-btn>
          </v-btn-toggle>
        </div>
        <v-switch
          :model-value="s.expand_completed"
          :label="t('settings.expandCompleted')"
          color="primary"
          hide-details
          inset
          @update:model-value="(v: unknown) => settings.set({ expand_completed: !!v })"
        />
        <div class="d-flex align-center flex-wrap ga-4">
          <v-switch
            :model-value="s.auto_urgent"
            :label="t('settings.autoUrgent')"
            color="primary"
            hide-details
            inset
            class="flex-grow-0"
            @update:model-value="(v: unknown) => settings.set({ auto_urgent: !!v })"
          />
          <v-text-field
            :model-value="s.auto_urgent_days"
            :label="t('settings.autoUrgentDays')"
            type="number"
            min="0"
            max="365"
            variant="outlined"
            hide-details
            :disabled="!s.auto_urgent"
            style="max-width: 130px"
            @update:model-value="setUrgentDays"
          />
        </div>
      </v-card-text>
    </v-card>

    <AccountSection />
    <McpSection />
    <DataSection />

    <!-- About -->
    <v-card id="about" class="section" rounded="lg">
      <v-card-title>{{ t('settings.about') }}</v-card-title>
      <v-list density="compact">
        <v-list-item :title="t('settings.version')" :subtitle="auth.status?.version || '-'" />
        <v-list-item v-if="REPOSITORY_URL" :title="t('settings.repository')">
          <v-list-item-subtitle>
            <a :href="REPOSITORY_URL" target="_blank" rel="noopener">{{ REPOSITORY_URL }}</a>
          </v-list-item-subtitle>
        </v-list-item>
        <v-list-item :title="t('settings.license')" subtitle="MIT" />
      </v-list>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiCheck, mdiThemeLightDark, mdiWeatherNight, mdiWhiteBalanceSunny } from '@mdi/js'
import AccountSection from '@/components/settings/AccountSection.vue'
import DataSection from '@/components/settings/DataSection.vue'
import McpSection from '@/components/settings/McpSection.vue'
import { REPOSITORY_URL } from '@/config'
import { languages } from '@/locales'
import { useAuthStore } from '@/stores/auth'
import { accentPresets, useSettingsStore, type UserSettings } from '@/stores/settings'
import { browserTimezone, dayjs } from '@/utils/time'

const { t } = useI18n()
const auth = useAuthStore()
const settings = useSettingsStore()
const s = computed(() => settings.s)

const languageOptions = computed(() => [
  { title: t('settings.followBrowser'), value: '' },
  ...languages.map((l) => ({ title: l.label, value: l.code })),
])

const fontOptions = computed(() => [
  { title: t('settings.fontSystem'), value: 'system' },
  { title: t('settings.fontSans'), value: 'sans' },
  { title: t('settings.fontSerif'), value: 'serif' },
  { title: t('settings.fontCustom'), value: 'custom' },
])

const customFont = ref(s.value.custom_font)

watch(
  () => s.value.custom_font,
  (v) => {
    customFont.value = v
  }
)

const saveCustomFont = () => {
  const v = customFont.value.trim()
  if (v !== s.value.custom_font) {
    settings.set({ custom_font: v })
  }
}

const timezoneOptions = computed(() => {
  let zones: string[] = []
  try {
    zones = Intl.supportedValuesOf('timeZone')
  } catch {
    zones = [browserTimezone, 'UTC']
  }
  if (!zones.includes('UTC')) zones = ['UTC', ...zones]
  return [{ title: `${t('settings.followBrowser')} (${browserTimezone})`, value: '' }, ...zones.map((z) => ({ title: z, value: z }))]
})

const sample = dayjs('2026-10-30')
const dateFormats = (['YYYY-MM-DD', 'YYYY/MM/DD', 'MM/DD/YYYY'] as const).map((f) => ({ title: sample.format(f), value: f }))

function setUrgentDays(v: string) {
  const n = Math.round(Number(v))
  if (!Number.isFinite(n) || n < 0 || n > 365) return
  settings.set({ auto_urgent_days: n })
}
</script>

<style scoped>
/* Settings form-control tokens. Sizes use rem so they follow the user's font-size setting
   (40px / 14px at the default 16px root). */
.settings {
  --set-label-w: 10rem;
  --set-row-gap: 16px;
  --set-ctl-h: 2.5rem;
  --set-ctl-w: 20rem;
  --set-ctl-minw: 9rem;
  --set-ctl-px: 0.75rem;
  --set-ctl-fs: 0.875rem;
  --set-ctl-radius: 8px;
  --set-ctl-bg: rgb(var(--v-theme-surface));
  --set-ctl-border: rgba(var(--v-border-color), 0.1);
  --set-ctl-divider: rgba(var(--v-border-color), 0.08);
  --set-ctl-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  --set-ctl-shadow-hover: 0 2px 6px rgba(0, 0, 0, 0.12);
  --set-ctl-focus: 0 0 0 3px rgba(var(--v-theme-primary), 0.3);
  --set-ctl-transition: 150ms ease;
}
:global(.v-theme--dark .settings) {
  --set-ctl-border: rgba(var(--v-border-color), 0.14);
  --set-ctl-divider: rgba(var(--v-border-color), 0.1);
  --set-ctl-shadow: 0 1px 3px rgba(0, 0, 0, 0.45);
  --set-ctl-shadow-hover: 0 2px 8px rgba(0, 0, 0, 0.6);
  --set-ctl-focus: 0 0 0 3px rgba(var(--v-theme-primary), 0.45);
}

.settings :deep(.section) {
  margin-bottom: var(--gap);
}
/* Shared row style (also used by AccountSection / DataSection). */
.settings :deep(.row) {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-bottom: var(--set-row-gap);
}
.settings :deep(.label) {
  width: 140px;
  flex: 0 0 auto;
  font-weight: 500;
}
/* Two-column rows on this page: label | control. */
.settings .set-row {
  display: grid;
  grid-template-columns: var(--set-label-w) minmax(0, 1fr);
  align-items: center;
  justify-items: start;
  gap: 8px 16px;
  min-height: var(--set-ctl-h);
}
.settings .set-row > .label {
  width: auto;
  font-size: var(--set-ctl-fs);
}

/* Segmented button groups */
.settings .set-seg.v-btn-toggle {
  display: inline-flex;
  height: var(--set-ctl-h);
  min-width: var(--set-ctl-w);
  max-width: 100%;
  border: 1px solid var(--set-ctl-border);
  border-radius: var(--set-ctl-radius);
  background: var(--set-ctl-bg);
  box-shadow: var(--set-ctl-shadow);
  transition: box-shadow var(--set-ctl-transition), border-color var(--set-ctl-transition);
}
.settings .set-seg.v-btn-toggle:hover {
  box-shadow: var(--set-ctl-shadow-hover);
}
.settings .set-seg :deep(.v-btn) {
  flex: 1 1 0;
  height: 100%;
  min-width: 0;
  padding: 0 var(--set-ctl-px);
  font-size: var(--set-ctl-fs);
  font-weight: 400;
  letter-spacing: normal;
  text-transform: none;
  white-space: nowrap;
  transition: color var(--set-ctl-transition), background-color var(--set-ctl-transition);
}
.settings .set-seg.v-btn-group--divided :deep(.v-btn:not(:last-child)) {
  border-inline-end-color: var(--set-ctl-divider);
}
.settings .set-seg :deep(.v-btn--active) {
  font-weight: 500;
}
.settings .set-seg :deep(.v-btn:focus-visible) {
  outline: none;
  box-shadow: inset 0 0 0 2px rgb(var(--v-theme-primary));
}
.settings .set-seg :deep(.v-btn:active:not(.v-btn--disabled)) {
  background-color: rgba(var(--v-theme-on-surface), 0.06);
}
.settings .set-seg :deep(.v-btn .v-btn__prepend) {
  margin-inline-end: 6px;
}

/* Selects / autocomplete: same height, radius, border and shadow as the segmented groups */
.settings .set-select {
  width: var(--set-ctl-w);
  min-width: var(--set-ctl-minw);
  max-width: 100%;
  flex: 0 0 auto;
}
.settings .set-col {
  width: var(--set-ctl-w);
  max-width: 100%;
}
.settings .set-col .set-select {
  width: 100%;
}
.settings .set-select :deep(.v-field) {
  --v-field-padding-start: var(--set-ctl-px);
  --v-field-padding-end: var(--set-ctl-px);
  border: 1px solid var(--set-ctl-border);
  border-radius: var(--set-ctl-radius);
  background: var(--set-ctl-bg);
  box-shadow: var(--set-ctl-shadow);
  transition: box-shadow var(--set-ctl-transition), border-color var(--set-ctl-transition);
}
.settings .set-select :deep(.v-field__outline) {
  display: none;
}
.settings .set-select :deep(.v-field__input) {
  min-height: calc(var(--set-ctl-h) - 2px);
  padding-top: 0;
  padding-bottom: 0;
  font-size: var(--set-ctl-fs);
}
.settings .set-select :deep(.v-field:hover) {
  box-shadow: var(--set-ctl-shadow-hover);
}
.settings .set-select :deep(.v-field--focused) {
  border-color: rgb(var(--v-theme-primary));
  box-shadow: var(--set-ctl-focus);
}
.settings .set-select :deep(.v-field--disabled) {
  box-shadow: none;
}

@media (max-width: 599px) {
  .settings :deep(.label) {
    width: 100%;
  }
  .settings .set-row {
    grid-template-columns: minmax(0, 1fr);
    justify-items: stretch;
    gap: 8px;
  }
  .settings .set-seg.v-btn-toggle {
    min-width: 0;
    width: 100%;
    height: auto;
    min-height: var(--set-ctl-h);
    flex-wrap: wrap;
  }
  .settings .set-seg :deep(.v-btn) {
    height: auto;
    min-height: calc(var(--set-ctl-h) - 2px);
    padding: 0 8px;
  }
  .settings .set-select,
  .settings .set-col {
    width: 100%;
  }
}
@media (max-width: 379px) {
  .settings .set-seg :deep(.v-btn .v-btn__prepend) {
    display: none;
  }
}
</style>
