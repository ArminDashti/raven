<script setup lang="ts">
import { api } from '@/lib/api'
import {
  Bug,
  CalendarDays,
  CalendarRange,
  CircleCheck,
  Clock3,
  FolderTree,
  RefreshCw,
  Wrench,
} from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import Button from '@/components/ui/Button.vue'

export type PeriodStats = {
  bugs: number
  fixed_by_dev: number
  finished: number
}

export type TotalStats = PeriodStats & {
  pending_to_fix: number
}

export type BugStats = {
  total: TotalStats
  today: PeriodStats
  yesterday: PeriodStats
  this_week: PeriodStats
  last_week: PeriodStats
  by_page?: Array<{ page: string; bugs: number }>
}

const stats = ref<BugStats | null>(null)
const error = ref('')
const loading = ref(true)
let timer: number | undefined

const n = (value: number | undefined) => (value == null ? '—' : value)

const pctOf = (part: number | undefined, total: number | undefined) => {
  if (part == null || total == null || total <= 0) return null
  return Math.round((part / total) * 100)
}

const withPct = (part: number | undefined, total: number | undefined) => {
  const count = n(part)
  const pct = pctOf(part, total)
  return pct == null ? count : `${count} (${pct} %)`
}

const totalProgress = computed(() => {
  const t = stats.value?.total
  if (!t || t.bugs <= 0) return { pending: 0, fixed: 0, finished: 0 }
  return {
    pending: (t.pending_to_fix / t.bugs) * 100,
    fixed: (t.fixed_by_dev / t.bugs) * 100,
    finished: (t.finished / t.bugs) * 100,
  }
})

const pageMax = computed(() => {
  const rows = stats.value?.by_page || []
  return Math.max(1, ...rows.map((r) => r.bugs))
})

const pageRows = computed(() => stats.value?.by_page || [])

async function load() {
  loading.value = true
  error.value = ''
  try {
    stats.value = await api.bugStats()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load stats'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
  timer = window.setInterval(() => void load(), 30000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="w-full px-4 py-8">
    <div class="mb-8 flex flex-wrap items-end justify-end gap-4">
      <Button
        variant="outline"
        class="h-9 inline-flex items-center gap-1.5"
        :disabled="loading"
        @click="load"
      >
        <RefreshCw class="h-3.5 w-3.5" :class="{ 'animate-spin': loading }" :stroke-width="1.75" />
        Refresh
      </Button>
    </div>

    <p v-if="error" class="mb-4 text-rose-500">{{ error }}</p>

    <section class="mb-8">
      <div class="mb-3 flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
        <Bug class="h-4 w-4" :stroke-width="1.75" />
        Total
      </div>
      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="text-sm text-muted-foreground">Bugs</span>
            <Bug class="h-4 w-4 text-foreground/60" :stroke-width="1.75" />
          </div>
          <p class="mt-3 text-3xl font-semibold tabular-nums tracking-tight">
            {{ n(stats?.total.bugs) }}
          </p>
        </div>
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="text-sm text-muted-foreground">Pending to fix</span>
            <Clock3 class="h-4 w-4 text-amber-600 dark:text-amber-300" :stroke-width="1.75" />
          </div>
          <p class="mt-3 text-3xl font-semibold tabular-nums tracking-tight text-amber-700 dark:text-amber-300">
            {{ n(stats?.total.pending_to_fix) }}
          </p>
        </div>
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="text-sm text-muted-foreground">Fixed by dev</span>
            <Wrench class="h-4 w-4 text-sky-600 dark:text-sky-300" :stroke-width="1.75" />
          </div>
          <p class="mt-3 text-3xl font-semibold tabular-nums tracking-tight text-sky-700 dark:text-white">
            {{ withPct(stats?.total.fixed_by_dev, stats?.total.bugs) }}
          </p>
        </div>
        <div class="rounded-xl border border-border bg-card p-5 shadow-sm">
          <div class="flex items-center justify-between gap-2">
            <span class="text-sm text-muted-foreground">Finished</span>
            <CircleCheck class="h-4 w-4 text-emerald-600 dark:text-emerald-300" :stroke-width="1.75" />
          </div>
          <p class="mt-3 text-3xl font-semibold tabular-nums tracking-tight text-emerald-700 dark:text-emerald-300">
            {{ n(stats?.total.finished) }}
          </p>
        </div>
      </div>

      <div
        v-if="stats && stats.total.bugs > 0"
        class="mt-4 overflow-hidden rounded-full bg-muted h-2.5 flex"
        title="Share of all bugs by stage"
      >
        <div
          class="bg-amber-500/80 transition-all duration-500"
          :style="{ width: `${totalProgress.pending}%` }"
        />
        <div
          class="bg-sky-500/80 transition-all duration-500"
          :style="{ width: `${totalProgress.fixed}%` }"
        />
        <div
          class="bg-emerald-500/80 transition-all duration-500"
          :style="{ width: `${totalProgress.finished}%` }"
        />
      </div>
    </section>

    <div class="grid gap-4 lg:grid-cols-2">
      <section class="rounded-xl border border-border bg-card p-5 shadow-sm">
        <div class="mb-4 flex items-center gap-2 font-semibold">
          <CalendarDays class="h-4 w-4 opacity-80" :stroke-width="1.75" />
          Today
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Bugs</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ n(stats?.today.bugs) }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Fixed by dev</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-sky-700 dark:text-sky-300">
              {{ n(stats?.today.fixed_by_dev) }}
            </p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Finished</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-emerald-700 dark:text-emerald-300">
              {{ n(stats?.today.finished) }}
            </p>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-border bg-card p-5 shadow-sm">
        <div class="mb-4 flex items-center gap-2 font-semibold">
          <CalendarDays class="h-4 w-4 opacity-70" :stroke-width="1.75" />
          Yesterday
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Bugs</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ n(stats?.yesterday.bugs) }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Fixed by dev</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-sky-700 dark:text-sky-300">
              {{ n(stats?.yesterday.fixed_by_dev) }}
            </p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Finished</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-emerald-700 dark:text-emerald-300">
              {{ n(stats?.yesterday.finished) }}
            </p>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-border bg-card p-5 shadow-sm">
        <div class="mb-4 flex items-center gap-2 font-semibold" title="Week starts Saturday">
          <CalendarRange class="h-4 w-4 opacity-80" :stroke-width="1.75" />
          This week
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Bugs</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ n(stats?.this_week.bugs) }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Fixed by dev</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-sky-700 dark:text-sky-300">
              {{ n(stats?.this_week.fixed_by_dev) }}
            </p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Finished</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-emerald-700 dark:text-emerald-300">
              {{ n(stats?.this_week.finished) }}
            </p>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-border bg-card p-5 shadow-sm">
        <div class="mb-4 flex items-center gap-2 font-semibold" title="Week starts Saturday">
          <CalendarRange class="h-4 w-4 opacity-70" :stroke-width="1.75" />
          Last week
        </div>
        <div class="grid grid-cols-3 gap-3">
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Bugs</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums">{{ n(stats?.last_week.bugs) }}</p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Fixed by dev</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-sky-700 dark:text-sky-300">
              {{ n(stats?.last_week.fixed_by_dev) }}
            </p>
          </div>
          <div class="rounded-lg bg-muted/40 px-3 py-3">
            <p class="text-xs text-muted-foreground">Finished</p>
            <p class="mt-1 text-2xl font-semibold tabular-nums text-emerald-700 dark:text-emerald-300">
              {{ n(stats?.last_week.finished) }}
            </p>
          </div>
        </div>
      </section>
    </div>

    <section class="mt-8 rounded-xl border border-border bg-card p-5 shadow-sm">
      <div class="mb-4 flex items-center gap-2 font-semibold">
        <FolderTree class="h-4 w-4 opacity-80" :stroke-width="1.75" />
        By page
      </div>
      <p v-if="!pageRows.length" class="text-sm text-muted-foreground">No page data yet</p>
      <ul v-else class="space-y-3">
        <li v-for="row in pageRows" :key="row.page" class="space-y-1">
          <div class="flex items-center justify-between gap-3 text-sm">
            <span class="truncate font-mono text-xs" :title="row.page">{{ row.page }}</span>
            <span class="shrink-0 tabular-nums text-muted-foreground">{{ row.bugs }}</span>
          </div>
          <div class="h-2.5 overflow-hidden rounded-full bg-muted">
            <div
              class="h-full rounded-full bg-sky-500/80 transition-all duration-500"
              :style="{ width: `${(row.bugs / pageMax) * 100}%` }"
            />
          </div>
        </li>
      </ul>
    </section>
  </div>
</template>
