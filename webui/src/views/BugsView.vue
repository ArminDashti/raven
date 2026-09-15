<script setup lang="ts">
import PriorityLabel from '@/components/PriorityLabel.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { api } from '@/lib/api'
import { autoDir, displayBugUrl, toShamsiDateTime } from '@/lib/format'
import { useRoleDirectory } from '@/lib/roleDirectory'
import { badgeActorUsername, badgeStatus, displayPersonName } from '@/lib/status'
import {
  Calendar,
  CircleDot,
  ExternalLink,
  Flag,
  Link2,
  PanelLeftClose,
  PanelLeftOpen,
  RefreshCw,
  Search,
  User,
} from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

type BugRow = {
  id: number
  url: string
  status: string
  priority: string
  cycle: number
  created_at: string
  updated_at: string
  reporter_username: string
  reporter_user_id: number
  fixer_username: string
  last_history_status?: string
  last_history_actor?: string
}

const SHORTCUT_KEY = 'bug_report_shortcut_collapsed'

const rows = ref<BugRow[]>([])
const urlQuery = ref('')
const filterPriority = ref('')
const filterOwner = ref('')
const filterStatus = ref('')
const error = ref('')
const loading = ref(true)
const { developerNames, testerNames, managerNames, ensureRoleDirectory } = useRoleDirectory()

const selectedId = ref<number | null>(null)
const shortcutUrl = ref('')
const shortcutDescription = ref('')
const shortcutPageParameters = ref('')
const previewLoading = ref(false)
const previewError = ref('')
const shortcutCollapsed = ref(
  typeof localStorage !== 'undefined' && localStorage.getItem(SHORTCUT_KEY) === '1',
)
let selectSeq = 0

const descriptionDir = computed(() => autoDir(shortcutDescription.value))
const paramsDir = computed(() => autoDir(shortcutPageParameters.value))

const ownerOptions = computed(() => {
  const seen = new Set<string>()
  const out: Array<{ value: string; label: string }> = []
  for (const row of rows.value) {
    const key = row.reporter_username
    if (!key || seen.has(key)) continue
    seen.add(key)
    out.push({ value: key, label: displayPersonName(key) })
  }
  return out.sort((a, b) => a.label.localeCompare(b.label))
})

const statusOptions = computed(() => {
  const seen = new Set<string>()
  for (const row of rows.value) {
    seen.add(badgeStatus(row))
  }
  for (const s of ['reported', 'fixed', 'waiting_manager', 'done', 'rejected', 'team_message']) {
    seen.add(s)
  }
  return [...seen].sort()
})

const filteredRows = computed(() => {
  const q = urlQuery.value.trim().toLowerCase()
  const pri = filterPriority.value
  const owner = filterOwner.value
  const st = filterStatus.value
  return rows.value.filter((row) => {
    if (q && !row.url.toLowerCase().includes(q)) return false
    if (pri && (row.priority || 'normal').toLowerCase() !== pri) return false
    if (owner && row.reporter_username !== owner) return false
    if (st && badgeStatus(row) !== st) return false
    return true
  })
})

function toggleShortcut() {
  shortcutCollapsed.value = !shortcutCollapsed.value
  localStorage.setItem(SHORTCUT_KEY, shortcutCollapsed.value ? '1' : '0')
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await ensureRoleDirectory()
    rows.value = await api.bugReports()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load'
  } finally {
    loading.value = false
  }
}

async function selectRow(row: BugRow) {
  const seq = ++selectSeq
  selectedId.value = row.id
  shortcutUrl.value = row.url
  shortcutDescription.value = ''
  shortcutPageParameters.value = ''
  previewError.value = ''
  previewLoading.value = true
  try {
    const full = await api.bugReport(row.id)
    if (seq !== selectSeq) return
    shortcutUrl.value = full.url
    shortcutDescription.value = full.description
    shortcutPageParameters.value = full.page_parameters
  } catch (e: unknown) {
    if (seq !== selectSeq) return
    previewError.value = e instanceof Error ? e.message : 'Failed to load preview'
  } finally {
    if (seq === selectSeq) previewLoading.value = false
  }
}

onMounted(() => void load())
</script>

<template>
  <div class="w-full px-4 py-6">
    <div class="flex flex-col gap-6 xl:flex-row xl:items-start">
      <section class="min-w-0 flex-1" aria-label="Bug list">
        <div class="mb-4 flex flex-wrap items-center gap-2">
          <label class="sr-only" for="bug-url-search">Search URLs</label>
          <div class="relative min-w-[12rem] flex-1 max-w-xl">
            <Search
              class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
              :stroke-width="1.75"
            />
            <Input
              id="bug-url-search"
              v-model="urlQuery"
              type="search"
              placeholder="Search URLs…"
              class="pl-9"
              dir="ltr"
              aria-label="Search URLs"
            />
          </div>
          <label class="sr-only" for="filter-priority">Priority</label>
          <select
            id="filter-priority"
            v-model="filterPriority"
            class="flex h-9 min-w-[8rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
            aria-label="Filter by Priority"
          >
            <option value="">All priorities</option>
            <option value="low">Low</option>
            <option value="normal">Medium</option>
            <option value="high">High</option>
          </select>
          <label class="sr-only" for="filter-owner">Owner</label>
          <select
            id="filter-owner"
            v-model="filterOwner"
            class="flex h-9 min-w-[10rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
            aria-label="Filter by Owner"
          >
            <option value="">All owners</option>
            <option v-for="o in ownerOptions" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
          <label class="sr-only" for="filter-status">Status</label>
          <select
            id="filter-status"
            v-model="filterStatus"
            class="flex h-9 min-w-[10rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
            aria-label="Filter by Status"
          >
            <option value="">All statuses</option>
            <option v-for="s in statusOptions" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
        <p v-if="error" class="text-rose-400 mb-2">{{ error }}</p>
        <p v-if="loading" class="text-muted-foreground">Loading…</p>
        <div v-else class="w-full overflow-x-auto">
          <table class="w-full min-w-full text-sm border-y border-border">
            <thead class="bg-muted/50 text-left">
              <tr>
                <th class="px-4 py-3 font-medium">
                  <span class="inline-flex items-center gap-1.5">
                    <Link2 class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    URL
                  </span>
                </th>
                <th class="px-4 py-3 font-medium whitespace-nowrap">
                  <span class="inline-flex items-center gap-1.5">
                    <Flag class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    Priority
                  </span>
                </th>
                <th class="px-4 py-3 font-medium whitespace-nowrap">
                  <span class="inline-flex items-center gap-1.5">
                    <RefreshCw class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    Cycle
                  </span>
                </th>
                <th class="px-4 py-3 font-medium whitespace-nowrap">
                  <span class="inline-flex items-center gap-1.5">
                    <Calendar class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    Created at
                  </span>
                </th>
                <th class="px-4 py-3 font-medium">
                  <span class="inline-flex items-center gap-1.5">
                    <User class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    Owner
                  </span>
                </th>
                <th class="px-4 py-3 font-medium">
                  <span class="inline-flex items-center gap-1.5">
                    <CircleDot class="h-3.5 w-3.5 shrink-0 opacity-80" :stroke-width="1.75" />
                    Status
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in filteredRows"
                :key="row.id"
                class="border-t border-border hover:bg-accent/40 cursor-pointer"
                :class="{ 'bg-primary/25': selectedId === row.id }"
                :aria-selected="selectedId === row.id"
                tabindex="0"
                @click="selectRow(row)"
                @keydown.enter.prevent="selectRow(row)"
              >
                <td class="px-4 py-3 max-w-md truncate" :title="row.url">{{ displayBugUrl(row.url) }}</td>
                <td class="px-4 py-3 whitespace-nowrap">
                  <PriorityLabel :priority="row.priority" />
                </td>
                <td class="px-4 py-3 whitespace-nowrap font-mono text-xs">{{ row.cycle ?? 1 }}</td>
                <td class="px-4 py-3 whitespace-nowrap font-mono text-xs">
                  {{ toShamsiDateTime(row.created_at) }}
                </td>
                <td class="px-4 py-3">
                  <span class="inline-flex items-center gap-2">
                    <UserAvatar :name="displayPersonName(row.reporter_username)" size="sm" />
                    <RouterLink
                      :to="{ name: 'user-profile', params: { username: row.reporter_username } }"
                      class="text-sky-700 hover:underline dark:text-sky-300"
                      @click.stop
                    >
                      {{ displayPersonName(row.reporter_username) }}
                    </RouterLink>
                  </span>
                </td>
                <td class="px-4 py-3">
                  <StatusBadge
                    :status="badgeStatus(row)"
                    :fixer-username="row.fixer_username"
                    :reporter-username="row.reporter_username"
                    :actor-username="badgeActorUsername(row)"
                    :developer-names="developerNames"
                    :tester-names="testerNames"
                    :manager-names="managerNames"
                    :closed-at="row.status === 'done' ? row.updated_at : null"
                  />
                </td>
              </tr>
              <tr v-if="filteredRows.length === 0">
                <td colspan="6" class="px-4 py-8 text-center text-muted-foreground">
                  {{ rows.length === 0 ? 'No bugs yet' : 'No bugs match these filters' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section
        class="w-full shrink-0 xl:sticky xl:top-6 transition-[width] duration-200"
        :class="shortcutCollapsed ? 'xl:w-14' : 'xl:w-[29.7rem]'"
        aria-label="Report bug shortcut"
      >
        <div
          class="mb-3 flex items-center"
          :class="shortcutCollapsed ? 'justify-center' : 'justify-between gap-2'"
        >
          <h2 v-if="!shortcutCollapsed" class="text-sm font-medium">Report shortcut</h2>
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md hover:bg-accent text-muted-foreground"
            :title="shortcutCollapsed ? 'Expand report shortcut' : 'Collapse report shortcut'"
            @click="toggleShortcut"
          >
            <PanelLeftOpen v-if="shortcutCollapsed" class="h-4 w-4" :stroke-width="1.75" />
            <PanelLeftClose v-else class="h-4 w-4" :stroke-width="1.75" />
          </button>
        </div>
        <div v-if="!shortcutCollapsed" class="space-y-3">
          <div class="field-panel-stack">
            <span class="field-panel-label">URL</span>
            <p class="text-sm break-all" dir="ltr">{{ shortcutUrl || '—' }}</p>
          </div>
          <div class="field-panel-stack">
            <span class="field-panel-label">Description</span>
            <p v-if="previewLoading" class="text-sm text-muted-foreground">Loading…</p>
            <p
              v-else
              class="max-h-40 overflow-y-auto whitespace-pre-wrap text-sm"
              :dir="descriptionDir"
            >
              {{ shortcutDescription || 'Select a bug row to preview' }}
            </p>
          </div>
          <div class="field-panel-stack">
            <span class="field-panel-label">Page parameters</span>
            <p v-if="previewLoading" class="text-sm text-muted-foreground">Loading…</p>
            <p
              v-else
              class="max-h-32 overflow-y-auto whitespace-pre-wrap text-sm"
              :dir="paramsDir"
            >
              {{ shortcutPageParameters || '—' }}
            </p>
          </div>
          <p v-if="previewError" class="text-sm text-rose-400">{{ previewError }}</p>
          <RouterLink
            v-if="selectedId != null"
            class="inline-flex h-9 w-full items-center justify-center gap-1.5 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            :to="{ name: 'bug-detail', params: { id: String(selectedId) } }"
          >
            <ExternalLink class="h-3.5 w-3.5" :stroke-width="1.75" />
            Open bug
          </RouterLink>
          <Button
            v-else
            class="h-9 w-full inline-flex items-center justify-center gap-1.5"
            disabled
          >
            <ExternalLink class="h-3.5 w-3.5" :stroke-width="1.75" />
            Open bug
          </Button>
        </div>
      </section>
    </div>
  </div>
</template>
