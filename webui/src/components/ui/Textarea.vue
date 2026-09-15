<script setup lang="ts">
import { cn } from '@/lib/utils'
import { computed } from 'vue'

const props = defineProps<{
  modelValue?: string
  placeholder?: string
  required?: boolean
  rows?: number
  dir?: 'ltr' | 'rtl' | 'auto'
  class?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [string] }>()

const classes = computed(() =>
  cn(
    'flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
)
</script>

<template>
  <textarea
    :class="classes"
    :value="modelValue"
    :placeholder="placeholder"
    :required="required"
    :rows="rows || 4"
    :dir="dir || 'ltr'"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>
