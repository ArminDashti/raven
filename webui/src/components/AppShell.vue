<script setup lang="ts">
import { api } from '@/lib/api'
import { clearSession, getToken, getUser, setSession, type AuthUser } from '@/lib/auth'
import { canReport } from '@/lib/permissions'
import { displayPersonName, fullNameFromParts } from '@/lib/status'
import { ensureRoleDirectory } from '@/lib/roleDirectory'
import {
  dismissPushBanner,
  ensurePushSubscription,
  getPushCapability,
  isPushBannerDismissed,
  subscribeAfterPermissionGranted,
  type PushCapability,
} from '@/lib/push'
import {
  Bug,
  Bell,
  CirclePlus,
  LayoutDashboard,
  LogOut,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  Sun,
} from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import Button from '@/components/ui/Button.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { resolveTheme, toggleTheme, type ThemeMode } from '@/lib/theme'

const SIDEBAR_KEY = 'bug_report_sidebar_collapsed'

const route = useRoute()
const router = useRouter()
const user = ref<AuthUser | null>(getUser())
const unread = ref(0)
const toast = ref('')
const pushCapability = ref<PushCapability>('unsupported')
const pushDismissed = ref(false)
const pushBusy = ref(false)
const offline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)
const appVersion = __APP_VERSION__
const theme = ref<ThemeMode>(resolveTheme())
const collapsed = ref(
  typeof localStorage !== 'undefined' && localStorage.getItem(SIDEBAR_KEY) === '1',
)
let prevUnread = -1
let timer: number | undefined

function onOnline() {
  offline.value = false
}

function onOffline() {
  offline.value = true
}

const showNav = computed(() => route.name !== 'login')
const showReport = computed(() => canReport(user.value?.role))
const isDark = computed(() => theme.value === 'dark')
const brandLogoSrc = computed(
  () =>
    `${import.meta.env.BASE_URL}${isDark.value ? 'dark-theme-logo.png' : 'white-theme-logo.png'}`,
)
const profileLabel = computed(
  () =>
    fullNameFromParts(user.value?.first_name, user.value?.last_name, user.value?.username) ||
    displayPersonName(user.value?.username) ||
    'Profile',
)

function onToggleTheme() {
  theme.value = toggleTheme()
}

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem(SIDEBAR_KEY, collapsed.value ? '1' : '0')
}

const showEnableBanner = computed(
  () =>
    showNav.value &&
    !!user.value &&
    !pushDismissed.value &&
    pushCapability.value === 'default',
)

const showPushStatus = computed(() => {
  if (!showNav.value || !user.value || pushDismissed.value) return null
  if (pushCapability.value === 'denied') {
    return 'Browser notifications are blocked. Enable them in site settings if you want alerts.'
  }
  return null
})

function refreshPushState() {
  pushCapability.value = getPushCapability()
  pushDismissed.value = isPushBannerDismissed()
}

async function refreshUnread() {
  if (!user.value) return
  try {
    const res = await api.unreadCount()
    if (prevUnread >= 0 && res.count > prevUnread) {
      toast.value = `You have ${res.count} unread notification(s)`
      window.setTimeout(() => {
        toast.value = ''
      }, 3500)
    }
    prevUnread = res.count
    unread.value = res.count
  } catch {
    /* ignore poll errors */
  }
}

async function onEnableNotifications() {
  pushBusy.value = true
  try {
    const result = await ensurePushSubscription()
    refreshPushState()
    if (result === 'granted') {
      toast.value = 'Browser notifications enabled'
      window.setTimeout(() => {
        toast.value = ''
      }, 3000)
    }
  } finally {
    pushBusy.value = false
  }
}

function onNotNow() {
  dismissPushBanner()
  pushDismissed.value = true
}

function logout() {
  clearSession()
  user.value = null
  router.push({ name: 'login' })
}

async function refreshUserNames() {
  if (!user.value) return
  try {
    await ensureRoleDirectory()
    const me = await api.me()
    const next: AuthUser = {
      ...user.value,
      first_name: me.first_name,
      last_name: me.last_name,
      username: me.username,
      role: me.role as AuthUser['role'],
      gender: me.gender,
      avatar_style: me.avatar_style,
      has_avatar: me.has_avatar,
    }
    user.value = next
    const token = getToken()
    if (token) setSession(token, next)
  } catch {
    /* keep session user */
  }
}

onMounted(() => {
  user.value = getUser()
  theme.value = resolveTheme()
  offline.value = !navigator.onLine
  window.addEventListener('online', onOnline)
  window.addEventListener('offline', onOffline)
  refreshPushState()
  void refreshUnread()
  void refreshUserNames()
  if (user.value && pushCapability.value === 'granted') {
    void subscribeAfterPermissionGranted()
  }
  timer = window.setInterval(() => {
    void refreshUnread()
  }, 30000)
})

onUnmounted(() => {
  window.removeEventListener('online', onOnline)
  window.removeEventListener('offline', onOffline)
  if (timer) window.clearInterval(timer)
})

watch(
  () => route.fullPath,
  () => {
    user.value = getUser()
    refreshPushState()
    void refreshUnread()
  },
)
</script>

<template>
  <div class="min-h-screen flex">
    <aside
      v-if="showNav"
      class="sticky top-0 z-40 flex h-screen shrink-0 flex-col border-r border-border bg-card/80 backdrop-blur transition-[width] duration-200"
      :class="collapsed ? 'w-14' : 'w-56'"
    >
      <div
        class="border-b border-border flex items-center"
        :class="collapsed ? 'justify-center px-1 py-2' : 'justify-between gap-2 px-3 py-2'"
      >
        <span
          v-if="!collapsed"
          class="font-raven-brand truncate text-lg font-normal text-foreground"
          title="Raven"
        >
          Raven
        </span>
        <button
          type="button"
          class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md hover:bg-accent text-muted-foreground"
          :title="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
          @click="toggleSidebar"
        >
          <PanelLeftOpen v-if="collapsed" class="h-4 w-4" :stroke-width="1.75" />
          <PanelLeftClose v-else class="h-4 w-4" :stroke-width="1.75" />
        </button>
      </div>

      <nav class="flex flex-1 flex-col gap-1 p-2 text-sm overflow-y-auto">
        <RouterLink
          v-if="showReport"
          to="/report-bug"
          class="px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          title="Report Bug"
        >
          <CirclePlus class="h-4 w-4 shrink-0 opacity-80" :stroke-width="1.75" />
          <span v-if="!collapsed">Report Bug</span>
        </RouterLink>
        <RouterLink
          to="/bugs"
          class="px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          title="Bugs"
        >
          <Bug class="h-4 w-4 shrink-0 opacity-80" :stroke-width="1.75" />
          <span v-if="!collapsed">Bugs</span>
        </RouterLink>
        <RouterLink
          to="/stats"
          class="px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          title="Stats"
        >
          <LayoutDashboard class="h-4 w-4 shrink-0 opacity-80" :stroke-width="1.75" />
          <span v-if="!collapsed">Stats</span>
        </RouterLink>
        <RouterLink
          to="/notifications"
          class="relative px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          title="Notifications"
        >
          <Bell class="h-4 w-4 shrink-0 opacity-80" :stroke-width="1.75" />
          <span v-if="!collapsed">Notifications</span>
          <span
            v-if="unread > 0"
            class="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-rose-500 px-1.5 text-[11px] font-bold text-white"
          >
            {{ unread }}
          </span>
        </RouterLink>
        <RouterLink
          to="/settings"
          class="px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          title="Settings"
        >
          <Settings class="h-4 w-4 shrink-0 opacity-80" :stroke-width="1.75" />
          <span v-if="!collapsed">Settings</span>
        </RouterLink>
        <RouterLink
          to="/profile"
          class="px-3 py-2 rounded-md hover:bg-accent inline-flex items-center gap-2"
          :class="collapsed ? 'justify-center px-2' : ''"
          active-class="bg-accent"
          :title="profileLabel"
        >
          <UserAvatar
            :name="profileLabel"
            :gender="user?.gender"
            :avatar-style="user?.avatar_style"
            :user-id="user?.id"
            :has-avatar="!!user?.has_avatar"
            size="xs"
            class="opacity-80"
          />
          <span v-if="!collapsed" class="truncate">{{ profileLabel }}</span>
        </RouterLink>
      </nav>

      <div class="mt-auto border-t border-border p-2 space-y-2">
        <div class="flex items-stretch gap-2" :class="collapsed ? '' : ''">
          <RouterLink
            v-if="!collapsed"
            to="/bugs"
            class="flex w-1/2 self-stretch items-center justify-center min-w-0"
            title="Raven"
          >
            <img
              :src="brandLogoSrc"
              alt="Raven"
              class="h-full w-full object-contain"
            />
          </RouterLink>
          <div
            class="min-w-0 flex-col gap-2 flex"
            :class="collapsed ? 'w-full' : 'w-1/2'"
          >
            <Button
              variant="outline"
              class="h-8 w-full inline-flex items-center justify-center gap-1.5 px-1"
              :title="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
              @click="onToggleTheme"
            >
              <Sun v-if="isDark" class="h-3.5 w-3.5 shrink-0" :stroke-width="1.75" />
              <Moon v-else class="h-3.5 w-3.5 shrink-0" :stroke-width="1.75" />
              <span v-if="!collapsed" class="truncate text-xs">{{ isDark ? 'Light' : 'Dark' }}</span>
            </Button>
            <Button
              variant="outline"
              class="h-8 w-full inline-flex items-center justify-center gap-1.5 px-1"
              title="Logout"
              @click="logout"
            >
              <LogOut class="h-3.5 w-3.5 shrink-0" :stroke-width="1.75" />
              <span v-if="!collapsed" class="truncate text-xs">Logout</span>
            </Button>
            <p v-if="!collapsed" class="px-1 text-center text-[11px] text-muted-foreground">
              v{{ appVersion }}
            </p>
          </div>
        </div>
      </div>
    </aside>

    <div class="flex min-h-screen min-w-0 flex-1 flex-col">
      <div
        v-if="offline"
        class="border-b border-rose-500/40 bg-rose-500/10 px-4 py-2 text-sm text-foreground"
        role="status"
      >
        You are offline. The app shell still works; live data needs a network connection.
      </div>

      <div
        v-if="showEnableBanner"
        class="border-b border-amber-500/40 bg-amber-500/10 px-4 py-3 text-sm"
      >
        <div class="flex flex-wrap items-center gap-3 w-full">
          <p class="flex-1 min-w-[12rem] text-foreground">
            Allow browser notifications so you can get alerts about bug updates. Click Enable, then
            choose Allow in the browser prompt.
          </p>
          <div class="flex items-center gap-2 shrink-0">
            <Button class="h-8" :disabled="pushBusy" @click="onEnableNotifications">
              {{ pushBusy ? 'Enabling…' : 'Enable notifications' }}
            </Button>
            <Button variant="outline" class="h-8" :disabled="pushBusy" @click="onNotNow">
              Not now
            </Button>
          </div>
        </div>
      </div>

      <div
        v-else-if="showPushStatus"
        class="border-b border-border bg-muted/40 px-4 py-2 text-sm text-muted-foreground"
      >
        <div class="flex flex-wrap items-center gap-3 w-full">
          <p class="flex-1">{{ showPushStatus }}</p>
          <Button variant="outline" class="h-7 text-xs" @click="onNotNow">Dismiss</Button>
        </div>
      </div>

      <main class="flex-1 w-full">
        <RouterView />
      </main>
    </div>

    <div
      v-if="toast"
      class="fixed bottom-4 right-4 z-50 rounded-md border border-border bg-card px-4 py-3 text-sm shadow-lg"
    >
      {{ toast }}
    </div>
  </div>
</template>
