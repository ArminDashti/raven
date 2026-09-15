<script setup lang="ts">
import StatusBadge from '@/components/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import { api } from '@/lib/api'
import { useRoleDirectory } from '@/lib/roleDirectory'
import { displayBugUrl, toShamsiDateTime } from '@/lib/format'
import { displayPersonName } from '@/lib/status'
import { ArrowLeft } from 'lucide-vue-next'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const rows = ref<
  Array<{
    id: number
    url: string
    status: string
    rejection_reason: string | null
    created_at: string
    actor_username: string
    reporter_username: string
  }>
>([])
const error = ref('')
const { developerNames, testerNames, managerNames, ensureRoleDirectory } = useRoleDirectory()

onMounted(async () => {
  try {
    await ensureRoleDirectory()
    const id = Number(route.params.id)
    rows.value = await api.history(id)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to load history'
  }
})
</script>

<template>
  <div class="w-full py-6">
    <div class="px-4 mb-4 flex items-center gap-3">
      <Button
        variant="outline"
        class="inline-flex items-center gap-1.5"
        @click="router.push({ name: 'bug-detail', params: { id: String(route.params.id) } })"
      >
        <ArrowLeft class="h-3.5 w-3.5" :stroke-width="1.75" />
        Back
      </Button>
      <h1 class="text-2xl font-semibold">Bug History</h1>
    </div>
    <p v-if="error" class="px-4 text-rose-400">{{ error }}</p>
    <div class="w-full overflow-x-auto">
      <table class="w-full min-w-full text-sm border-y border-border">
        <thead class="bg-muted/50 text-left">
          <tr>
            <th class="px-4 py-3 font-medium">URL</th>
            <th class="px-4 py-3 font-medium">User</th>
            <th class="px-4 py-3 font-medium">Datetime</th>
            <th class="px-4 py-3 font-medium">Status</th>
            <th class="px-4 py-3 font-medium">Note</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.id" class="border-t border-border">
            <td class="px-4 py-3 max-w-md truncate" :title="row.url">{{ displayBugUrl(row.url) }}</td>
            <td class="px-4 py-3">{{ displayPersonName(row.actor_username) }}</td>
            <td class="px-4 py-3 whitespace-nowrap font-mono text-xs">
              {{ toShamsiDateTime(row.created_at) }}
            </td>
            <td class="px-4 py-3">
              <StatusBadge
                :status="row.status"
                :actor-username="row.actor_username"
                :reporter-username="row.reporter_username"
                :developer-names="developerNames"
                :tester-names="testerNames"
                :manager-names="managerNames"
                :closed-at="row.status === 'done' ? row.created_at : null"
              />
            </td>
            <td class="px-4 py-3" :dir="/[\u0600-\u06FF]/.test(row.rejection_reason || '') ? 'rtl' : 'ltr'">
              {{ row.rejection_reason || '—' }}
            </td>
          </tr>
          <tr v-if="rows.length === 0 && !error">
            <td colspan="5" class="px-4 py-8 text-center text-muted-foreground">No history</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
