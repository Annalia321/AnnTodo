<template>
  <div class="page">
    <div class="page-head">
      <span class="page-title">{{ t('admin.title') }}</span>
      <v-spacer />
      <v-btn color="primary" :prepend-icon="mdiAccountPlus" @click="openCreate">{{ t('admin.newUser') }}</v-btn>
    </div>

    <v-card rounded="lg" class="mb-4">
      <v-card-title>{{ t('admin.users') }}</v-card-title>
      <v-table class="users">
        <thead>
          <tr>
            <th>{{ t('auth.username') }}</th>
            <th>{{ t('auth.nickname') }}</th>
            <th>{{ t('admin.role') }}</th>
            <th>{{ t('admin.status') }}</th>
            <th>{{ t('admin.createdAt') }}</th>
            <th>{{ t('admin.lastLogin') }}</th>
            <th class="text-right">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>
              <div class="d-flex align-center ga-2">
                <UserAvatar :user="u" :size="28" />
                <span>{{ u.username }}</span>
                <span v-if="u.id === auth.user?.id" class="muted text-caption">({{ t('admin.you') }})</span>
              </div>
            </td>
            <td>{{ u.nickname }}</td>
            <td>
              <v-chip size="small" :color="u.role === 'admin' ? 'primary' : undefined" variant="tonal">
                {{ u.role === 'admin' ? t('admin.roleAdmin') : t('admin.roleUser') }}
              </v-chip>
            </td>
            <td>
              <v-chip size="small" :color="u.status === 'active' ? 'success' : 'error'" variant="tonal">
                {{ u.status === 'active' ? t('admin.active') : t('admin.disabled') }}
              </v-chip>
            </td>
            <td class="nowrap">{{ fmtDateTime(u.created_at) }}</td>
            <td class="nowrap">{{ u.last_login_at ? fmtDateTime(u.last_login_at) : '-' }}</td>
            <td class="text-right">
              <v-menu location="bottom end">
                <template #activator="{ props: p }">
                  <v-btn v-bind="p" :icon="mdiDotsHorizontal" variant="text" size="small" :title="t('common.actions')" />
                </template>
                <v-list density="compact">
                  <v-list-item :prepend-icon="mdiLockReset" :title="t('admin.resetPassword')" @click="openReset(u)" />
                  <template v-if="u.id !== auth.user?.id">
                    <v-list-item
                      :prepend-icon="mdiShieldAccount"
                      :title="u.role === 'admin' ? t('admin.makeUser') : t('admin.makeAdmin')"
                      @click="update(u, { role: u.role === 'admin' ? 'user' : 'admin' })"
                    />
                    <v-list-item
                      :prepend-icon="u.status === 'active' ? mdiAccountCancel : mdiAccountCheck"
                      :title="u.status === 'active' ? t('admin.disable') : t('admin.enable')"
                      @click="update(u, { status: u.status === 'active' ? 'disabled' : 'active' })"
                    />
                    <v-list-item :prepend-icon="mdiDelete" :title="t('common.delete')" base-color="error" @click="deleting = u" />
                  </template>
                </v-list>
              </v-menu>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <v-card rounded="lg">
      <v-card-title>{{ t('admin.system') }}</v-card-title>
      <v-card-text>
        <v-form class="form" @submit.prevent="saveSystem">
          <v-switch v-model="system.allow_register" :label="t('admin.allowRegister')" color="primary" inset hide-details class="mb-2" />
          <v-text-field v-model="system.site_name" :label="t('admin.siteName')" variant="outlined" density="comfortable" maxlength="64" />
          <v-btn type="submit" color="primary" :loading="systemBusy">{{ t('common.save') }}</v-btn>
        </v-form>
      </v-card-text>
    </v-card>

    <!-- New user -->
    <v-dialog v-model="createOpen" max-width="440">
      <v-card :title="t('admin.newUser')">
        <v-form @submit.prevent="create">
          <v-card-text class="d-flex flex-column ga-1">
            <v-text-field
              v-model="form.username"
              :label="t('auth.username')"
              variant="outlined"
              density="comfortable"
              autocomplete="off"
              :hint="t('auth.usernameRule')"
              :rules="[(v: string) => !v || usernameOk(v) || t('auth.usernameRule')]"
            />
            <v-text-field
              v-model="form.password"
              :label="t('auth.password')"
              type="password"
              variant="outlined"
              density="comfortable"
              autocomplete="new-password"
              :hint="t('auth.passwordRule')"
              :rules="[(v: string) => !v || passwordOk(v) || t('auth.passwordRule')]"
            />
            <v-text-field v-model="form.nickname" :label="t('auth.nickname')" variant="outlined" density="comfortable" maxlength="64" />
            <v-text-field
              v-model="form.email"
              :label="t('auth.emailOptional')"
              type="email"
              variant="outlined"
              density="comfortable"
              :rules="[(v: string) => emailRule(v) || t('errors.invalid_email')]"
            />
            <v-btn-toggle v-model="form.role" mandatory color="primary" variant="outlined" divided density="comfortable">
              <v-btn value="user">{{ t('admin.roleUser') }}</v-btn>
              <v-btn value="admin">{{ t('admin.roleAdmin') }}</v-btn>
            </v-btn-toggle>
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" @click="createOpen = false">{{ t('common.cancel') }}</v-btn>
            <v-btn type="submit" color="primary" variant="flat" :loading="busy" :disabled="!createValid">{{ t('common.create') }}</v-btn>
          </v-card-actions>
        </v-form>
      </v-card>
    </v-dialog>

    <!-- Reset password -->
    <v-dialog :model-value="!!resetting" max-width="400" @update:model-value="(v: boolean) => !v && (resetting = null)">
      <v-card :title="t('admin.resetPassword')" :subtitle="resetting?.username">
        <v-form @submit.prevent="resetPassword">
          <v-card-text>
            <v-text-field
              v-model="resetValue"
              :label="t('settings.newPassword')"
              type="password"
              variant="outlined"
              density="comfortable"
              autocomplete="new-password"
              :hint="t('auth.passwordRule')"
              :rules="[(v: string) => !v || passwordOk(v) || t('auth.passwordRule')]"
            />
          </v-card-text>
          <v-card-actions>
            <v-spacer />
            <v-btn variant="text" @click="resetting = null">{{ t('common.cancel') }}</v-btn>
            <v-btn type="submit" color="primary" variant="flat" :loading="busy" :disabled="!passwordOk(resetValue)">{{ t('common.save') }}</v-btn>
          </v-card-actions>
        </v-form>
      </v-card>
    </v-dialog>

    <!-- Delete -->
    <v-dialog :model-value="!!deleting" max-width="420" @update:model-value="(v: boolean) => !v && (deleting = null)">
      <v-card :title="t('common.delete')" :text="t('admin.deleteConfirm', { name: deleting?.username ?? '' })">
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="deleting = null">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" :loading="busy" @click="remove">{{ t('common.delete') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  mdiAccountCancel,
  mdiAccountCheck,
  mdiAccountPlus,
  mdiDelete,
  mdiDotsHorizontal,
  mdiLockReset,
  mdiShieldAccount,
} from '@mdi/js'
import { api, type User } from '@/api'
import UserAvatar from '@/components/UserAvatar.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { fmtDateTime } from '@/utils/time'
import { emailRule, passwordOk, usernameOk } from '@/utils/validate'

const { t } = useI18n()
const auth = useAuthStore()
const ui = useUiStore()

const users = ref<User[]>([])
const busy = ref(false)

async function loadUsers() {
  try {
    users.value = await api.get<User[]>('/admin/users')
  } catch (e) {
    ui.error(e)
  }
}

function replace(u: User) {
  const i = users.value.findIndex((x) => x.id === u.id)
  if (i >= 0) users.value[i] = u
  else users.value.push(u)
}

async function update(u: User, patch: Partial<Pick<User, 'nickname' | 'email' | 'role' | 'status'>>) {
  try {
    replace(await api.put<User>(`/admin/users/${u.id}`, patch))
  } catch (e) {
    ui.error(e)
  }
}

// ----- create -----
const createOpen = ref(false)
const form = reactive({ username: '', password: '', nickname: '', email: '', role: 'user' as User['role'] })
const createValid = computed(() => usernameOk(form.username) && passwordOk(form.password) && emailRule(form.email))
function openCreate() {
  Object.assign(form, { username: '', password: '', nickname: '', email: '', role: 'user' })
  createOpen.value = true
}
async function create() {
  if (!createValid.value) return
  busy.value = true
  try {
    replace(
      await api.post<User>('/admin/users', {
        username: form.username.trim(),
        password: form.password,
        nickname: form.nickname.trim(),
        email: form.email.trim(),
        role: form.role,
      }),
    )
    createOpen.value = false
  } catch (e) {
    ui.error(e)
  } finally {
    busy.value = false
  }
}

// ----- reset password -----
const resetting = ref<User | null>(null)
const resetValue = ref('')
function openReset(u: User) {
  resetValue.value = ''
  resetting.value = u
}
async function resetPassword() {
  if (!resetting.value || !passwordOk(resetValue.value)) return
  busy.value = true
  try {
    await api.post(`/admin/users/${resetting.value.id}/password`, { password: resetValue.value })
    resetting.value = null
    ui.notify(t('common.saved'))
  } catch (e) {
    ui.error(e)
  } finally {
    busy.value = false
  }
}

// ----- delete -----
const deleting = ref<User | null>(null)
async function remove() {
  const u = deleting.value
  if (!u) return
  busy.value = true
  try {
    await api.del(`/admin/users/${u.id}`)
    users.value = users.value.filter((x) => x.id !== u.id)
    deleting.value = null
  } catch (e) {
    ui.error(e)
  } finally {
    busy.value = false
  }
}

// ----- system settings -----
const system = reactive({ allow_register: false, site_name: '' })
const systemBusy = ref(false)
async function loadSystem() {
  try {
    Object.assign(system, await api.get<{ allow_register: boolean; site_name: string }>('/admin/settings'))
  } catch (e) {
    ui.error(e)
  }
}
async function saveSystem() {
  systemBusy.value = true
  try {
    const st = await api.put<{ allow_register: boolean; site_name: string }>('/admin/settings', {
      allow_register: system.allow_register,
      site_name: system.site_name.trim(),
    })
    Object.assign(system, st)
    if (auth.status) Object.assign(auth.status, st)
    ui.notify(t('common.saved'))
  } catch (e) {
    ui.error(e)
  } finally {
    systemBusy.value = false
  }
}

onMounted(() => {
  loadUsers()
  loadSystem()
})
</script>

<style scoped>
.users {
  overflow-x: auto;
}
.nowrap {
  white-space: nowrap;
}
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
