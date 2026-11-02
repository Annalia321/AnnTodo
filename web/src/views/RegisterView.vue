<template>
  <AuthLayout>
    <div v-if="mode === 'setup'" class="text-center mb-5">
      <div class="text-h6">{{ t('auth.setupTitle') }}</div>
      <div class="text-body-2 muted">{{ t('auth.setupHint') }}</div>
    </div>
    <v-form ref="form" @submit.prevent="submit">
      <v-text-field
        v-model="username"
        :label="t('auth.username')"
        :hint="t('auth.usernameRule')"
        :rules="[usernameRule]"
        variant="outlined"
        autocomplete="username"
        autofocus
        class="mb-2"
      />
      <v-text-field v-model="nickname" :label="t('auth.nickname')" variant="outlined" maxlength="64" class="mb-2" />
      <v-text-field
        v-model="email"
        :label="t('auth.emailOptional')"
        :rules="[emailRule]"
        type="email"
        variant="outlined"
        autocomplete="email"
        class="mb-2"
      />
      <v-text-field
        v-model="password"
        :label="t('auth.password')"
        :hint="t('auth.passwordRule')"
        :rules="[passwordRule]"
        type="password"
        variant="outlined"
        autocomplete="new-password"
        class="mb-2"
      />
      <v-text-field
        v-model="confirm"
        :label="t('auth.confirmPassword')"
        :rules="[(v: string) => v === password || t('auth.passwordMismatch')]"
        type="password"
        variant="outlined"
        autocomplete="new-password"
        class="mb-2"
      />
      <v-alert v-if="error" type="error" variant="tonal" density="compact" class="mb-3">{{ error }}</v-alert>
      <v-btn type="submit" color="primary" size="large" block :loading="busy">
        {{ mode === 'setup' ? t('auth.setupSubmit') : t('auth.register') }}
      </v-btn>
    </v-form>
    <div v-if="mode === 'register'" class="text-center mt-4 text-body-2">
      {{ t('auth.haveAccount') }}
      <router-link to="/login" class="link">{{ t('auth.login') }}</router-link>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AuthLayout from '@/components/AuthLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { errorText } from '@/stores/ui'
import { emailRule as emailOk, passwordOk, usernameOk } from '@/utils/validate'

const props = defineProps<{ mode: 'register' | 'setup' }>()
const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const form = ref<{ validate: () => Promise<{ valid: boolean }> } | null>(null)
const username = ref('')
const nickname = ref('')
const email = ref('')
const password = ref('')
const confirm = ref('')
const busy = ref(false)
const error = ref('')

const usernameRule = (v: string) => usernameOk(v) || t('auth.usernameRule')
const passwordRule = (v: string) => passwordOk(v) || t('auth.passwordRule')
const emailRule = (v: string) => emailOk(v) || t('errors.invalid_email')

async function submit() {
  if (busy.value) return
  const res = await form.value?.validate()
  if (res && !res.valid) return
  busy.value = true
  error.value = ''
  const body = {
    username: username.value.trim(),
    password: password.value,
    nickname: nickname.value.trim(),
    email: email.value.trim(),
  }
  try {
    if (props.mode === 'setup') await auth.setup(body)
    else await auth.register(body)
    await router.replace('/')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.link {
  color: rgb(var(--v-theme-primary));
  text-decoration: none;
  font-weight: 500;
}
</style>
