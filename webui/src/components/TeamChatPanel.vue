<script setup lang="ts">
import Button from '@/components/ui/Button.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { api, type TeamMessage } from '@/lib/api'
import { getUser } from '@/lib/auth'
import { autoDir, toShamsiDateTime } from '@/lib/format'
import { displayPersonName } from '@/lib/status'
import { MessageSquare, Send } from 'lucide-vue-next'
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'

const props = defineProps<{ bugId: number }>()
const emit = defineEmits<{ 'message-sent': [] }>()

const POLL_MS = 10_000
const MAX_BODY = 2000

const user = getUser()
const messages = ref<TeamMessage[]>([])
const draft = ref('')
const error = ref('')
const loading = ref(true)
const posting = ref(false)
const listEl = ref<HTMLElement | null>(null)
let timer: number | null = null

function roleChipClass(role: string): string {
  switch (role) {
    case 'developer':
      return 'bg-sky-500/15 text-sky-700 dark:text-white'
    case 'manager':
      return 'bg-amber-500/15 text-amber-800 dark:text-white'
    case 'tester':
      return 'bg-emerald-500/15 text-emerald-800 dark:text-white'
    case 'admin':
      return 'bg-rose-500/15 text-rose-800 dark:text-white'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

async function scrollToBottom() {
  await nextTick()
  const el = listEl.value
  if (el) el.scrollTop = el.scrollHeight
}

async function load(opts: { silent?: boolean } = {}) {
  if (!props.bugId) return
  if (!opts.silent) loading.value = true
  try {
    const next = await api.teamChat(props.bugId)
    const prevLast = messages.value[messages.value.length - 1]?.id
    const nextLast = next[next.length - 1]?.id
    messages.value = next
    error.value = ''
    if (!opts.silent || prevLast !== nextLast) {
      await scrollToBottom()
    }
  } catch (e: unknown) {
    if (!opts.silent) {
      error.value = e instanceof Error ? e.message : 'Failed to load comments'
    }
  } finally {
    if (!opts.silent) loading.value = false
  }
}

async function send() {
  const body = draft.value.trim()
  if (!body || posting.value || !props.bugId) return
  if ([...body].length > MAX_BODY) {
    error.value = `Comment must be at most ${MAX_BODY} characters`
    return
  }
  posting.value = true
  error.value = ''
  try {
    const created = await api.postTeamChat(props.bugId, body)
    draft.value = ''
    if (!messages.value.some((m) => m.id === created.id)) {
      messages.value = [...messages.value, created]
    }
    await scrollToBottom()
    emit('message-sent')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to send'
  } finally {
    posting.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void send()
  }
}

function onVisibility() {
  if (!document.hidden) void load({ silent: true })
}

function startPoll() {
  stopPoll()
  timer = window.setInterval(() => {
    if (!document.hidden) void load({ silent: true })
  }, POLL_MS)
}

function stopPoll() {
  if (timer != null) {
    window.clearInterval(timer)
    timer = null
  }
}

watch(
  () => props.bugId,
  () => {
    messages.value = []
    void load()
  },
)

watch(
  () => messages.value.length,
  () => {
    void scrollToBottom()
  },
)

onMounted(() => {
  void load()
  startPoll()
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  stopPoll()
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>

<template>
  <aside
    class="flex h-full min-h-[40vh] flex-col overflow-hidden rounded-lg border border-border bg-card shadow-sm lg:min-h-0"
  >
    <header class="flex shrink-0 items-center gap-2 border-b border-border px-4 py-3">
      <MessageSquare class="h-4 w-4 text-muted-foreground" :stroke-width="1.75" />
      <div class="min-w-0 flex-1">
        <h2 class="text-sm font-semibold leading-none">Comments</h2>
        <p class="mt-1 text-xs text-muted-foreground">Updates every few seconds</p>
      </div>
    </header>

    <div ref="listEl" class="min-h-0 flex-1 space-y-3 overflow-y-auto px-3 py-3">
      <p v-if="loading" class="px-1 text-sm text-muted-foreground">Loading comments…</p>
      <p
        v-else-if="messages.length === 0"
        class="px-1 py-8 text-center text-sm text-muted-foreground"
      >
        No comments yet — say hello to the team.
      </p>
      <template v-else>
        <div
          v-for="msg in messages"
          :key="msg.id"
          class="flex"
          :class="msg.author_user_id === user?.id ? 'justify-end' : 'justify-start'"
        >
          <div
            class="max-w-[85%] rounded-2xl px-3 py-2 text-sm shadow-sm"
            :class="
              msg.author_user_id === user?.id
                ? 'rounded-br-md bg-primary text-primary-foreground'
                : 'rounded-bl-md bg-muted/70 text-foreground'
            "
          >
            <div
              class="mb-1 flex flex-wrap items-center gap-1.5 text-[11px] leading-none"
              :class="
                msg.author_user_id === user?.id
                  ? 'text-primary-foreground/80'
                  : 'text-muted-foreground'
              "
            >
              <UserAvatar :name="displayPersonName(msg.author_username)" size="sm" />
              <span class="font-medium">{{ displayPersonName(msg.author_username) }}</span>
              <span
                class="rounded px-1.5 py-0.5 font-medium capitalize"
                :class="
                  msg.author_user_id === user?.id
                    ? 'bg-primary-foreground/15 text-primary-foreground'
                    : roleChipClass(msg.author_role)
                "
              >
                {{ msg.author_role }}
              </span>
              <span class="font-mono opacity-80">{{ toShamsiDateTime(msg.created_at) }}</span>
            </div>
            <p class="whitespace-pre-wrap break-words" :dir="autoDir(msg.body)">{{ msg.body }}</p>
          </div>
        </div>
      </template>
    </div>

    <p v-if="error" class="shrink-0 px-4 pb-1 text-xs text-rose-500">{{ error }}</p>

    <form
      class="flex shrink-0 items-end gap-2 border-t border-border p-3"
      @submit.prevent="send"
    >
      <label class="sr-only" for="team-comment-draft">Comment</label>
      <textarea
        id="team-comment-draft"
        v-model="draft"
        rows="2"
        maxlength="2000"
        placeholder="Write a comment…"
        class="min-h-[2.5rem] max-h-28 flex-1 resize-y rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        dir="auto"
        :disabled="posting"
        @keydown="onKeydown"
      />
      <Button type="submit" class="h-10 shrink-0 px-3" :disabled="posting || !draft.trim()">
        <Send class="h-4 w-4" :stroke-width="1.75" />
        <span class="sr-only">Send</span>
      </Button>
    </form>
  </aside>
</template>
