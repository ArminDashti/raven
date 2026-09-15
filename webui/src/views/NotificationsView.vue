<script setup lang="ts">
import StatusBadge from '@/components/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import { api } from '@/lib/api'
import { useRoleDirectory } from '@/lib/roleDirectory'
import { autoDir, displayBugUrl } from '@/lib/format'
import { displayPersonName } from '@/lib/status'
import { CheckCheck, Eye, RefreshCw } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

type Notif = {
  id: number
  bug_report_id: number
  kind: string
  is_read: boolean
  message: string
  url: string
  bug_status: string
  fixer_username: string
  reporter_username: string
  rejection_reason: string | null
}

const rows = ref<Notif[]>([])
const error = ref('')
const router = useRouter()
const { developerNames, testerNames, managerNames, ensureRoleDirectory } = useRoleDirectory()

async function load() {
  error.value = ''
  try {
    await ensureRoleDirectory()
    rows.value = await api.notifications()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  }
}

async function markRead(id: number) {
  await api.markRead(id)
  await load()
}

async function markAll() {
  await api.markAllRead()
  await load()
}

function openBug(bugId: number) {
  void router.push({ name: 'bug-detail', params: { id: String(bugId) } })
}

function statusFor(row: Notif): string {
  if (row.kind === 'done') return 'done'
  if (row.kind === 'rejected_for_dev') return 'rejected'
  if (row.kind === 'fixed_for_tester') return 'fixed'
  if (row.kind === 'team_message') return 'team_message'
  return row.bug_status
}

onMounted(() => void load())
</script>

<template>
  <div class="w-full py-6">
    <div class="px-4 mb-4 flex items-center justify-between gap-4">
      <h1 class="text-2xl font-semibold">Notifications</h1>
      <div class="flex gap-2">
        <Button variant="outline" class="inline-flex items-center gap-1.5" @click="load">
          <RefreshCw class="h-3.5 w-3.5" :stroke-width="1.75" />
          Refresh
        </Button>
        <Button variant="secondary" class="inline-flex items-center gap-1.5" @click="markAll">
          <CheckCheck class="h-3.5 w-3.5" :stroke-width="1.75" />
          Mark all read
        </Button>
      </div>
    </div>
    <p v-if="error" class="px-4 text-rose-400 mb-2">{{ error }}</p>
    <div class="w-full overflow-x-auto">
      <table class="w-full min-w-full text-sm border-y border-border">
        <thead class="bg-muted/50 text-left">
          <tr>
            <th class="px-4 py-3 font-medium">URL</th>
            <th class="px-4 py-3 font-medium">Comment</th>
            <th class="px-4 py-3 font-medium">Dev who fixed</th>
            <th class="px-4 py-3 font-medium">Status</th>
            <th class="px-4 py-3 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in rows"
            :key="row.id"
            class="border-t border-border"
            :class="{ 'bg-accent/20': !row.is_read }"
          >
            <td class="px-4 py-3 max-w-[12rem] truncate" :title="row.url">
              <button class="text-left hover:underline" @click="openBug(row.bug_report_id)">
                {{ displayBugUrl(row.url) }}
              </button>
              <span v-if="!row.is_read" class="ml-2 text-[10px] uppercase text-amber-400">new</span>
            </td>
            <td class="px-4 py-3 max-w-md">
              <p
                class="line-clamp-3 whitespace-pre-wrap text-muted-foreground"
                :dir="autoDir(row.message || '')"
                :title="row.message"
              >
                {{ row.message || '—' }}
              </p>
            </td>
            <td class="px-4 py-3">{{ displayPersonName(row.fixer_username) || '—' }}</td>
            <td class="px-4 py-3">
              <StatusBadge
                :status="statusFor(row)"
                :fixer-username="row.fixer_username"
                :reporter-username="row.reporter_username"
                :developer-names="developerNames"
                :tester-names="testerNames"
                :manager-names="managerNames"
              />
            </td>
            <td class="px-4 py-3">
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  class="h-8 inline-flex items-center gap-1.5"
                  variant="outline"
                  @click="openBug(row.bug_report_id)"
                >
                  <Eye class="h-3.5 w-3.5" :stroke-width="1.75" />
                  Open
                </Button>
                <Button
                  v-if="!row.is_read"
                  variant="ghost"
                  class="h-8"
                  @click="markRead(row.id)"
                >
                  Mark read
                </Button>
              </div>
            </td>
          </tr>
          <tr v-if="rows.length === 0">
            <td colspan="5" class="px-4 py-8 text-center text-muted-foreground">No notifications</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
