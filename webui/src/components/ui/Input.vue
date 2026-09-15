<script setup lang="ts">
import { cn } from '@/lib/utils'
import { computed } from 'vue'

const props = defineProps<{
  id?: string
  modelValue?: string
  type?: string
  placeholder?: string
  required?: boolean
  disabled?: boolean
  dir?: 'ltr' | 'rtl' | 'auto'
  class?: string
  ariaLabel?: string
}>()

const emit = defineEmits<{ 'update:modelValue': [string] }>()

const classes = computed(() =>
  cn(
    'flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
)
</script>

<template>
  <input
    :id="id"
    :class="classes"
    :type="type || 'text'"
    :value="modelValue"
    :placeholder="placeholder"
    :required="required"
    :disabled="disabled"
    :dir="dir || 'ltr'"
    :aria-label="ariaLabel"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>
