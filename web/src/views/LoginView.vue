<template>
  <AuthLayout>
    <v-form @submit.prevent="submit">
      <v-text-field
        v-model="username"
        :label="t('auth.usernameOrEmail')"
        variant="outlined"
        autocomplete="username"
        autofocus
        class="mb-2"
      />
      <v-text-field
        v-model="password"
        :label="t('auth.password')"
        variant="outlined"
        :type="show ? 'text' : 'password'"
        :append-inner-icon="show ? mdiEyeOff : mdiEye"
        autocomplete="current-password"
        @click:append-inner="show = !show"
      />
      <v-checkbox v-model="remember" :label="t('auth.remember')" color="primary" hide-details density="compact" class="mb-3" />
      <v-alert v-if="error" type="error" variant="tonal" density="compact" class="mb-3">{{ error }}</v-alert>
      <v-btn type="submit" color="primary" size="large" block :loading="busy" :disabled="!username || !password">
        {{ t('auth.login') }}
      </v-btn>
    </v-form>
    <div v-if="auth.status?.allow_register" class="text-center mt-4 text-body-2">
      {{ t('auth.noAccount') }}
      <router-link to="/register" class="link">{{ t('auth.register') }}</router-link>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { mdiEye, mdiEyeOff } from '@mdi/js'
import AuthLayout from '@/components/AuthLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { errorText } from '@/stores/ui'

const { t } = useI18n()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const username = ref('')
const password = ref('')
const remember = ref(false)
const show = ref(false)
const busy = ref(false)
const error = ref('')

async function submit() {
  if (!username.value || !password.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await auth.login(username.value.trim(), password.value, remember.value)
    const redirect = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/') ? route.query.redirect : '/'
    await router.replace(redirect)
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
