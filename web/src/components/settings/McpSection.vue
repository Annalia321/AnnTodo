<template>
  <v-card id="mcp" class="section" rounded="lg">
    <v-card-title>{{ t('settings.mcp.title') }}</v-card-title>
    <v-card-text>
      <div class="muted text-body-2 mb-2">{{ t('settings.mcp.description') }}</div>
      <ol class="steps muted text-body-2 mb-5">
        <li>{{ t('settings.mcp.step1') }}</li>
        <li>{{ t('settings.mcp.step2') }}</li>
        <li>{{ t('settings.mcp.step3') }}</li>
      </ol>

      <div class="row">
        <div class="label">{{ t('settings.mcp.status') }}</div>
        <div class="flex-grow-1" style="min-width: 0">
          <template v-if="info?.exists">
            <div><code class="mono">{{ t('settings.mcp.tokenHint', { hint: info.hint }) }}</code></div>
            <div class="muted text-body-2 mt-1">
              {{ t('settings.mcp.createdAt', { t: fmtDateTime(info.created_at) }) }} ·
              {{ t('settings.mcp.lastUsed', { t: info.last_used_at ? fmtDateTime(info.last_used_at) : t('settings.mcp.never') }) }}
            </div>
          </template>
          <div v-else class="muted">{{ loaded ? t('settings.mcp.noToken') : '…' }}</div>
        </div>
      </div>

      <div class="d-flex flex-wrap ga-3 mb-2">
        <v-btn color="primary" variant="tonal" :prepend-icon="mdiKeyPlus" :loading="busy" :disabled="!loaded" @click="onGenerate">
          {{ info?.exists ? t('settings.mcp.regenerate') : t('settings.mcp.generate') }}
        </v-btn>
        <v-btn v-if="info?.exists" variant="text" color="error" :disabled="busy" @click="confirm = 'revoke'">
          {{ t('settings.mcp.revoke') }}
        </v-btn>
      </div>

      <v-alert v-if="newToken" type="warning" variant="tonal" density="compact" class="my-4">
        <div class="font-weight-medium mb-2">{{ t('settings.mcp.newToken') }}</div>
        <div class="copy-line">
          <code class="mono token">{{ newToken }}</code>
          <v-btn size="small" variant="flat" color="primary" :prepend-icon="copied === 'token' ? mdiCheck : mdiContentCopy" @click="copy(newToken, 'token')">{{ copied === 'token' ? t('settings.mcp.copied') : t('settings.mcp.copy') }}</v-btn>
        </div>
        <div class="text-body-2 mt-2">{{ t('settings.mcp.onceWarning') }}</div>
      </v-alert>

      <v-divider class="my-5" />

      <div class="row">
        <div class="label">{{ t('settings.mcp.serverUrl') }}</div>
        <div class="copy-line flex-grow-1">
          <code class="mono">{{ serverUrl }}</code>
          <v-btn size="small" variant="text" :icon="copied === 'url' ? mdiCheck : mdiContentCopy" :color="copied === 'url' ? 'success' : undefined" :aria-label="t('settings.mcp.copy')" @click="copy(serverUrl, 'url')" />
        </div>
      </div>
      <div class="row">
        <div class="label">{{ t('settings.mcp.header') }}</div>
        <div class="copy-line flex-grow-1">
          <code class="mono">{{ headerLine }}</code>
          <v-btn size="small" variant="text" :icon="copied === 'header' ? mdiCheck : mdiContentCopy" :color="copied === 'header' ? 'success' : undefined" :aria-label="t('settings.mcp.copy')" @click="copy(headerLine, 'header')" />
        </div>
      </div>

      <div class="d-flex align-center mt-2 mb-1">
        <div class="font-weight-medium">{{ t('settings.mcp.config') }}</div>
        <v-spacer />
        <v-btn size="small" variant="tonal" :prepend-icon="copied === 'config' ? mdiCheck : mdiContentCopy" @click="copy(configJson, 'config')">{{ copied === 'config' ? t('settings.mcp.copied') : t('settings.mcp.copy') }}</v-btn>
      </div>
      <pre class="mono config">{{ configJson }}</pre>
      <div class="muted text-body-2 mt-2">{{ t('settings.mcp.configHint') }}</div>
    </v-card-text>

    <v-dialog :model-value="confirm !== null" max-width="440" @update:model-value="(v) => !v && (confirm = null)">
      <v-card
        :title="confirm === 'revoke' ? t('settings.mcp.revoke') : t('settings.mcp.regenerate')"
        :text="confirm === 'revoke' ? t('settings.mcp.revokeConfirm') : t('settings.mcp.regenerateConfirm')"
      >
        <v-card-actions>
          <v-spacer />
          <v-btn variant="text" @click="confirm = null">{{ t('common.cancel') }}</v-btn>
          <v-btn color="error" variant="flat" @click="confirm === 'revoke' ? revoke() : generate()">{{ t('common.confirm') }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { mdiCheck, mdiContentCopy, mdiKeyPlus } from '@mdi/js'
import { api, type McpTokenInfo } from '@/api'
import { useUiStore } from '@/stores/ui'
import { browserTimezone, fmtDateTime } from '@/utils/time'

const { t } = useI18n()
const ui = useUiStore()

const info = ref<McpTokenInfo | null>(null)
const loaded = ref(false)
const busy = ref(false)
const newToken = ref('')
const confirm = ref<'regenerate' | 'revoke' | null>(null)
/** Which copy button shows the "copied" state; toasts are queued, so inline feedback is clearer. */
const copied = ref('')
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const serverUrl = `${window.location.origin}/mcp`
const headerLine = computed(() => `Authorization: Bearer ${newToken.value || 'sk-...'}`)
const configJson = computed(() =>
  JSON.stringify(
    {
      mcpServers: {
        anntodo: {
          type: 'http',
          url: serverUrl,
          headers: { Authorization: `Bearer ${newToken.value || 'sk-...'}` },
        },
      },
    },
    null,
    2
  )
)

async function load() {
  try {
    info.value = await api.get<McpTokenInfo>('/mcp/token')
  } catch (e) {
    ui.error(e)
  } finally {
    loaded.value = true
  }
}

function onGenerate() {
  if (info.value?.exists) confirm.value = 'regenerate'
  else generate()
}

async function generate() {
  confirm.value = null
  busy.value = true
  try {
    const res = await api.post<{ token: string; info: McpTokenInfo }>('/mcp/token', { timezone: browserTimezone })
    newToken.value = res.token
    info.value = res.info
    ui.notify(t('settings.mcp.generated'), { color: 'success' })
  } catch (e) {
    ui.error(e)
  } finally {
    busy.value = false
  }
}

async function revoke() {
  confirm.value = null
  busy.value = true
  try {
    await api.del('/mcp/token')
    newToken.value = ''
    info.value = { exists: false, hint: '', created_at: null, last_used_at: null }
    ui.notify(t('settings.mcp.revoked'))
  } catch (e) {
    ui.error(e)
  } finally {
    busy.value = false
  }
}

// navigator.clipboard only exists in secure contexts (https or localhost); a phone
// opening http://192.168.x.x:8080 needs the execCommand fallback.
async function copy(text: string, key: string) {
  let done = false
  if (window.isSecureContext && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      done = true
    } catch {
      done = false
    }
  }
  if (!done) {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-1000px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    ta.setSelectionRange(0, text.length)
    try {
      done = document.execCommand('copy')
    } catch {
      done = false
    }
    document.body.removeChild(ta)
  }
  if (done) {
    copied.value = key
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => (copied.value = ''), 2000)
  } else {
    ui.notify(t('settings.mcp.copyFailed'), { color: 'error' })
  }
}

onMounted(load)
</script>

<style scoped>
.steps {
  padding-left: 1.25rem;
}
.mono {
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 0.8125rem;
  background: rgba(var(--v-theme-on-surface), 0.06);
  padding: 4px 8px;
  border-radius: 6px;
  word-break: break-all;
}
.copy-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.copy-line .mono {
  flex: 1 1 auto;
  min-width: 0;
}
.token {
  user-select: all;
  padding: 8px 10px;
}
.config {
  margin: 0;
  padding: 12px;
  white-space: pre-wrap;
  overflow-x: auto;
}
</style>