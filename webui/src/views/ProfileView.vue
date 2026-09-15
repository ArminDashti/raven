<script setup lang="ts">
import UserAvatar from '@/components/UserAvatar.vue'
import Button from '@/components/ui/Button.vue'
import { api, type BugStatsPayload, type UserProfile } from '@/lib/api'
import { getUser, setSession, getToken, type AuthUser } from '@/lib/auth'
import { AVATAR_STYLES } from '@/lib/avatar'
import { fullNameFromParts } from '@/lib/status'
import { UserRound } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const sessionUser = getUser()
const profile = ref<UserProfile | null>(null)
const avatarStyle = ref('')
const stats = ref<BugStatsPayload | null>(null)
const error = ref('')
const busy = ref(false)
const saved = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const routeUsername = computed(() => {
  const u = route.params.username
  return typeof u === 'string' && u ? u : ''
})

const isSelf = computed(() => {
  if (!profile.value || !sessionUser) return false
  return profile.value.id === sessionUser.id
})

const displayName = computed(() =>
  fullNameFromParts(profile.value?.first_name, profile.value?.last_name, profile.value?.username),
)

async function load() {
  error.value = ''
  profile.value = null
  stats.value = null
  try {
    if (routeUsername.value) {
      profile.value = await api.user(routeUsername.value)
      stats.value = await api.userBugStats(profile.value.username)
    } else {
      profile.value = await api.me()
      stats.value = await api.myBugStats()
    }
    avatarStyle.value = profile.value.avatar_style || profile.value.gender || ''
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load profile'
  }
}

function syncSession(next: UserProfile) {
  const token = getToken()
  const session = getUser()
  if (!token || !session || session.id !== next.id) return
  setSession(token, {
    ...session,
    first_name: next.first_name,
    last_name: next.last_name,
    gender: next.gender,
    avatar_style: next.avatar_style,
    has_avatar: next.has_avatar,
  } as AuthUser)
}

async function saveAvatar() {
  if (!isSelf.value) return
  error.value = ''
  busy.value = true
  try {
    const style = avatarStyle.value
    const body: { gender?: string; avatar_style?: string } = { avatar_style: style }
    if (style === 'man' || style === 'woman' || style === '') {
      body.gender = style
    }
    profile.value = await api.updateMe(body)
    syncSession(profile.value)
    saved.value = true
    window.setTimeout(() => {
      saved.value = false
    }, 2000)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Save failed'
  } finally {
    busy.value = false
  }
}

async function onPhotoChange(e: Event) {
  if (!isSelf.value) return
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  error.value = ''
  busy.value = true
  try {
    if (file.size > 2 * 1024 * 1024) {
      throw new Error('Photo exceeds 2MB limit')
    }
    if (!file.type.startsWith('image/')) {
      throw new Error('Photo must be an image')
    }
    profile.value = await api.uploadAvatar(file)
    syncSession(profile.value)
    saved.value = true
    window.setTimeout(() => {
      saved.value = false
    }, 2000)
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Upload failed'
  } finally {
    busy.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

watch(
  () => route.fullPath,
  () => {
    void load()
  },
)

onMounted(() => {
  void load()
})
</script>

<template>
  <div class="w-full px-4 py-8">
    <h1 class="mb-6 inline-flex items-center gap-2 text-2xl font-semibold">
      <UserRound class="h-6 w-6 opacity-80" :stroke-width="1.75" />
      Profile
    </h1>
    <p v-if="error" class="text-rose-400">{{ error }}</p>
    <div v-else-if="profile" class="max-w-xl space-y-4">
      <div class="flex items-center gap-4 rounded-lg border border-border bg-card p-6">
        <UserAvatar
          :name="displayName"
          :gender="profile.gender"
          :avatar-style="profile.avatar_style"
          :user-id="profile.id"
          :has-avatar="!!profile.has_avatar"
          size="lg"
        />
        <div class="min-w-0">
          <p class="font-medium truncate">{{ displayName }}</p>
          <p class="text-sm text-muted-foreground truncate">{{ profile.username }}</p>
        </div>
      </div>

      <div class="space-y-3 rounded-lg border border-border bg-card p-6">
        <div class="flex justify-between gap-4">
          <span class="text-muted-foreground">Name</span>
          <span class="font-medium">{{ displayName }}</span>
        </div>
        <div class="flex justify-between gap-4">
          <span class="text-muted-foreground">Username</span>
          <span class="font-medium">{{ profile.username }}</span>
        </div>
        <div class="flex justify-between gap-4">
          <span class="text-muted-foreground">Role</span>
          <span class="font-medium">{{ profile.role }}</span>
        </div>
        <div class="flex justify-between gap-4">
          <span class="text-muted-foreground">User ID</span>
          <span class="font-medium">{{ profile.id }}</span>
        </div>
      </div>

      <div v-if="stats" class="space-y-3 rounded-lg border border-border bg-card p-6">
        <h2 class="text-sm font-medium">{{ isSelf ? 'Your stats' : 'Stats' }}</h2>
        <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Bugs</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ stats.total.bugs }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Pending</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ stats.total.pending_to_fix }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Fixed by dev</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ stats.total.fixed_by_dev }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Finished</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ stats.total.finished }}</p>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span class="text-muted-foreground">Today:</span>
            {{ stats.today.bugs }} bugs
          </div>
          <div>
            <span class="text-muted-foreground">This week:</span>
            {{ stats.this_week.bugs }} bugs
          </div>
        </div>
      </div>

      <div v-if="isSelf" class="field-panel-stack space-y-3">
        <h2 class="field-panel-label">Avatar</h2>
        <div class="flex flex-wrap items-center gap-3">
          <UserAvatar
            v-for="s in ['', ...AVATAR_STYLES]"
            :key="s || 'empty'"
            :avatar-style="s"
            :gender="s === 'man' || s === 'woman' ? s : ''"
            size="sm"
          />
        </div>
        <select
          v-model="avatarStyle"
          class="flex h-9 w-full max-w-[14rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
        >
          <option value="">Empty</option>
          <option v-for="s in AVATAR_STYLES" :key="s" :value="s">
            {{ s.charAt(0).toUpperCase() + s.slice(1) }}
          </option>
        </select>
        <div class="flex flex-wrap items-center gap-2">
          <Button class="h-8" :disabled="busy" @click="saveAvatar">Save avatar</Button>
          <label class="inline-flex h-8 cursor-pointer items-center rounded-md border border-input bg-background px-3 text-sm hover:bg-accent">
            Upload photo
            <input
              ref="fileInput"
              type="file"
              accept="image/*"
              class="sr-only"
              @change="onPhotoChange"
            />
          </label>
          <span v-if="saved" class="text-sm text-emerald-600 dark:text-emerald-400">Saved</span>
        </div>
      </div>
    </div>
    <p v-else class="text-muted-foreground">Loading…</p>
  </div>
</template>
