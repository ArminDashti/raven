<script setup lang="ts">
import { cn } from '@/lib/utils'
import { statusLabel, type StatusPeople } from '@/lib/status'
import {
  Clock,
  Wrench,
  ShieldCheck,
  CircleCheck,
  CircleX,
  CircleDot,
  MessageSquare,
} from 'lucide-vue-next'
import { computed } from 'vue'

const props = defineProps<{
  status: string
  fixerUsername?: string | null
  reporterUsername?: string | null
  actorUsername?: string | null
  developerNames?: string[]
  testerNames?: string[]
  managerNames?: string[]
  closedAt?: string | null
}>()

const people = computed<StatusPeople>(() => ({
  fixerUsername: props.fixerUsername,
  reporterUsername: props.reporterUsername,
  actorUsername: props.actorUsername,
  developerNames: props.developerNames,
  testerNames: props.testerNames,
  managerNames: props.managerNames,
  closedAt: props.closedAt,
}))

const meta = computed(() => {
  const label = statusLabel(props.status, people.value)
  switch (props.status) {
    case 'reported':
    case 'pending':
      return {
        label,
        icon: Clock,
        class: 'text-amber-700 dark:text-white border-amber-500/40 bg-amber-500/10',
      }
    case 'fixed':
    case 'changed_by_dev':
      return {
        label,
        icon: Wrench,
        class: 'text-sky-700 dark:text-white border-sky-500/40 bg-sky-500/10',
      }
    case 'waiting_manager':
      return {
        label,
        icon: ShieldCheck,
        class: 'text-violet-700 dark:text-white border-violet-500/40 bg-violet-500/10',
      }
    case 'done':
    case 'accepted':
    case 'accept':
      return {
        label,
        icon: CircleCheck,
        class: 'text-emerald-700 dark:text-white border-emerald-500/40 bg-emerald-500/10',
      }
    case 'rejected':
    case 'reject':
      return {
        label,
        icon: CircleX,
        class: 'text-rose-700 dark:text-white border-rose-500/40 bg-rose-500/10',
      }
    case 'edit':
      return {
        label,
        icon: CircleDot,
        class: 'text-slate-700 dark:text-white border-slate-500/40 bg-slate-500/10',
      }
    case 'team_message':
      return {
        label,
        icon: MessageSquare,
        class: 'text-indigo-700 dark:text-white border-indigo-500/40 bg-indigo-500/10',
      }
    default:
      return { label, icon: CircleDot, class: 'text-muted-foreground border-border bg-muted' }
  }
})
</script>

<template>
  <span
    :class="
      cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium',
        meta.class,
      )
    "
  >
    <component :is="meta.icon" class="h-3.5 w-3.5 shrink-0" :stroke-width="1.75" />
    {{ meta.label }}
  </span>
</template>
