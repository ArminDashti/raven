<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ priority?: string | null }>()

const meta = computed(() => {
  const v = (props.priority || 'normal').toLowerCase()
  if (v === 'low') {
    return {
      label: 'Low',
      dot: 'bg-emerald-500',
    }
  }
  if (v === 'high') {
    return {
      label: 'High',
      dot: 'bg-red-500',
    }
  }
  return {
    label: 'Medium',
    dot: 'bg-yellow-400',
  }
})
</script>

<template>
  <span class="inline-flex items-center gap-1.5 whitespace-nowrap text-foreground">
    <span class="h-2.5 w-2.5 shrink-0 rounded-full" :class="meta.dot" aria-hidden="true" />
    {{ meta.label }}
  </span>
</template>
