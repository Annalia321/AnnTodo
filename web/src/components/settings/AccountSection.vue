<template>
  <v-card id="account" class="section" rounded="lg">
    <v-card-title>{{ t('settings.account') }}</v-card-title>
    <v-card-text>
      <!-- Avatar -->
      <div class="row">
        <div class="label">{{ t('settings.avatar') }}</div>
        <UserAvatar :user="auth.user" :size="64" />
        <v-btn variant="tonal" color="primary" :loading="avatarBusy" @click="fileInput?.click()">{{ t('settings.uploadAvatar') }}</v-btn>
        <v-btn v-if="auth.user?.has_avatar" variant="text" :disabled="avatarBusy" @click="removeAvatar">{{ t('settings.removeAvatar') }}</v-btn>
        <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/gif,image/webp" hidden @change="uploadAvatar" />
      </div>

      <!-- Profile -->
      <v-form class="form" @submit.prevent="saveProfile">
        <v-text-field :model-value="auth.user?.username" :label="t('auth.username')" variant="outlined" density="comfortable" disabled />
        <v-text-field v-model="nickname" :label="t('auth.nickname')" variant="outlined" density="comfortable" maxlength="64" />
        <v-text-field
          v-model="email"
          :label="t('auth.emailOptional')"
          type="email"
          variant="outlined"
          density="comfortable"
          :rules="[(v: string) => emailRule(v) || t('errors.invalid_email')]"
        />
        <v-btn type="submit" color="primary" :loading="profileBusy" :disabled="!profileDirty || !emailRule(email)">{{ t('common.save') }}</v-btn>
      </v-form>

      <v-divider class="my-6" />

      <!-- Password -->
      <div class="text-subtitle-1 font-weight-medium mb-3">{{ t('settings.changePassword') }}</div>
      <v-form class="form" @submit.prevent="changePassword">
        <v-text-field v-model="oldPassword" :label="t('settings.oldPassword')" type="password" autocomplete="current-password" variant="outlined" density="comfortable" />
        <v-text-field
          v-model="newPassword"
          :label="t('settings.newPassword')"
          type="password"
          autocomplete="new-password"
          variant="outlined"
          density="comfortable"
          :hint="t('auth.passwordRule')"
          :rules="[(v: string) => !v || passwordOk(v) || t('auth.passwordRule')]"
        />
        <v-text-field
          v-model="confirmPassword"
          :label="t('auth.confirmPassword')"
          type="password"
          autocomplete="new-password"
          variant="outlined"
          density="comfortable"
          :rules="[(v: string) => !v || v === newPassword || t('auth.passwordMismatch')]"
        />
        <v-btn type="submit" color="primary" :loading="passwordBusy" :disabled="!passwordValid">{{ t('settings.changePassword') }}</v-btn>
      </v-form>

      <v-divider class="my-6" />

      <!-- Sessions -->
      <div class="text-subtitle-1 font-weight-medium mb-1">{{ t('settings.sessions') }}</div>
      <v-list density="compact" class="bg-transparent">
        <v-list-item v-for="s in sessions" :key="s.id" class="px-0">
          <template #prepend>
            <v-icon :icon="mdiDevices" class="mr-2" />
          </template>
          <v-list-item-title>
            {{ s.device || '-' }}
            <v-chip v-if="s.current" size="x-small" color="primary" class="ml-2">{{ t('settings.currentDevice') }}</v-chip>
          </v-list-item-title>
          <v-list-item-subtitle>{{ s.ip }} · {{ t('settings.lastActive', { t: fmtDateTime(s.last_active_at) }) }}</v-list-item-subtitle>
          <template v-if="!s.current" #append>
            <v-btn variant="text" color="error" size="small" @click="revoke(s.id)">{{ t('settings.revoke') }}</v-btn>
          </template>
        </v-list-item>
      </v-list>

      <div class="muted text-body-2 mt-4">{{ t('settings.totpLater') }}</div>
    </v-card-text>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiDevices } from '@mdi/js'
import { api, request, type SessionInfo, type User } from '@/api'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { fmtDateTime } from '@/utils/time'
import { emailRule, passwordOk } from '@/utils/validate'

const { t } = useI18n()
const auth = useAuthStore()
const ui = useUiStore()

// ----- profile -----
const nickname = ref('')
const email = ref('')
watch(
  () => auth.user,
  (u) => {
    nickname.value = u?.nickname ?? ''
    email.value = u?.email ?? ''
  },
  { immediate: true },
)
const profileDirty = computed(() => nickname.value !== (auth.user?.nickname ?? '') || email.value !== (auth.user?.email ?? ''))
const profileBusy = ref(false)
async function saveProfile() {
  profileBusy.value = true
  try {
    auth.user = await api.put<User>('/account/profile', { nickname: nickname.value.trim(), email: email.value.trim() })
    ui.notify(t('settings.profileSaved'))
  } catch (e) {
    ui.error(e)
  } finally {
    profileBusy.value = false
  }
}

// ----- avatar -----
const fileInput = ref<HTMLInputElement | null>(null)
const avatarBusy = ref(false)
async function uploadAvatar(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (file.size > 2 << 20) {
    ui.notify(t('errors.invalid_avatar'), { color: 'error' })
    return
  }
  const fd = new FormData()
  fd.append('avatar', file)
  avatarBusy.value = true
  try {
    auth.user = await request<User>('POST', '/account/avatar', fd)
  } catch (e) {
    ui.error(e)
  } finally {
    avatarBusy.value = false
  }
}
async function removeAvatar() {
  avatarBusy.value = true
  try {
    auth.user = await api.del<User>('/account/avatar')
  } catch (e) {
    ui.error(e)
  } finally {
    avatarBusy.value = false
  }
}

// ----- password -----
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordBusy = ref(false)
const passwordValid = computed(() => !!oldPassword.value && passwordOk(newPassword.value) && newPassword.value === confirmPassword.value)
async function changePassword() {
  if (!passwordValid.value) return
  passwordBusy.value = true
  try {
    await api.put('/account/password', { old_password: oldPassword.value, new_password: newPassword.value })
    oldPassword.value = newPassword.value = confirmPassword.value = ''
    ui.notify(t('settings.passwordChanged'))
    await loadSessions()
  } catch (e) {
    ui.error(e)
  } finally {
    passwordBusy.value = false
  }
}

// ----- sessions -----
const sessions = ref<SessionInfo[]>([])
async function loadSessions() {
  try {
    sessions.value = await api.get<SessionInfo[]>('/account/sessions')
  } catch (e) {
    ui.error(e)
  }
}
async function revoke(id: string) {
  try {
    await api.del('/account/sessions/' + encodeURIComponent(id))
    sessions.value = sessions.value.filter((s) => s.id !== id)
  } catch (e) {
    ui.error(e)
  }
}

onMounted(loadSessions)
</script>

<style scoped>
.form {
  max-width: 420px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form .v-btn {
  align-self: flex-start;
}
</style>
