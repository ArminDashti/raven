<script setup lang="ts">
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { api } from '@/lib/api'
import { setSession, type AuthUser } from '@/lib/auth'
import { clearPushBannerDismiss } from '@/lib/push'
import { resolveTheme, toggleTheme, type ThemeMode } from '@/lib/theme'
import { LogIn, Moon, Sun } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const theme = ref<ThemeMode>(resolveTheme())
const isDark = computed(() => theme.value === 'dark')

function onToggleTheme() {
  theme.value = toggleTheme()
}

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    const res = await api.login(username.value.trim(), password.value)
    setSession(res.token, res.user as AuthUser)
    clearPushBannerDismiss()
    await router.push({ name: 'bugs' })
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center px-4 relative">
    <Button
      variant="outline"
      class="absolute top-4 right-4 h-8 inline-flex items-center gap-1.5"
      :title="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
      @click="onToggleTheme"
    >
      <Sun v-if="isDark" class="h-3.5 w-3.5" :stroke-width="1.75" />
      <Moon v-else class="h-3.5 w-3.5" :stroke-width="1.75" />
      {{ isDark ? 'Light' : 'Dark' }}
    </Button>
    <form
      class="w-full max-w-sm space-y-4 rounded-lg border border-border bg-card p-6 shadow"
      @submit.prevent="onSubmit"
    >
      <div>
        <h1 class="text-xl font-semibold">Sign in</h1>
        <p class="text-sm text-muted-foreground mt-1">Raven</p>
      </div>
      <div class="space-y-2">
        <label class="text-sm">Username</label>
        <Input v-model="username" required autocomplete="username" />
      </div>
      <div class="space-y-2">
        <label class="text-sm">Password</label>
        <Input v-model="password" type="password" required autocomplete="current-password" />
      </div>
      <p v-if="error" class="text-sm text-rose-600 dark:text-rose-400">{{ error }}</p>
      <Button type="submit" class="w-full inline-flex items-center justify-center gap-2" :disabled="loading">
        <LogIn class="h-4 w-4" :stroke-width="1.75" />
        {{ loading ? 'Signing in…' : 'Sign in' }}
      </Button>
    </form>
  </div>
</template>
