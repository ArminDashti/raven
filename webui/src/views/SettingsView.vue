<script setup lang="ts">
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import { applyBugBaseUrl } from '@/lib/format'
import { getBugBaseUrlOverride, setBugBaseUrlOverride } from '@/lib/settings'
import { Settings } from 'lucide-vue-next'
import { computed, ref } from 'vue'

const baseUrl = ref(getBugBaseUrlOverride())
const saved = ref(false)
const error = ref('')

const exampleOriginal =
  'http://erp.dpdc.co:8880/Pages/Warehouse/02-EtelaatePaieh/KalaAdamForoshGorohiVahedKharid.aspx'
const preview = computed(() => applyBugBaseUrl(exampleOriginal, baseUrl.value))

function save() {
  error.value = ''
  const trimmed = baseUrl.value.trim()
  if (trimmed) {
    try {
      // Must be an absolute origin-like URL
      const u = new URL(trimmed.includes('://') ? trimmed : `http://${trimmed}`)
      if (!u.host) throw new Error('invalid')
      baseUrl.value = trimmed.replace(/\/+$/, '') || trimmed
    } catch {
      error.value = 'Enter a valid base URL (e.g. http://example.com)'
      return
    }
  }
  setBugBaseUrlOverride(baseUrl.value)
  saved.value = true
  window.setTimeout(() => {
    saved.value = false
  }, 2000)
}

function clear() {
  baseUrl.value = ''
  setBugBaseUrlOverride('')
  saved.value = true
  window.setTimeout(() => {
    saved.value = false
  }, 2000)
}
</script>

<template>
  <div class="w-full px-4 py-8">
    <h1 class="mb-6 inline-flex items-center gap-2 text-2xl font-semibold">
      <Settings class="h-6 w-6 opacity-80" :stroke-width="1.75" />
      Settings
    </h1>

    <section class="field-panel-stack max-w-2xl space-y-4">
      <div>
        <h2 class="field-panel-label">Bug base URL (personal)</h2>
        <p class="mt-1 text-sm text-muted-foreground">
          Override the host for bug links for you only on this browser. Path stays the same.
        </p>
      </div>
      <div class="space-y-2">
        <label class="text-sm text-muted-foreground">Base URL</label>
        <Input
          v-model="baseUrl"
          placeholder="http://example.com"
          autocomplete="off"
        />
      </div>
      <div class="rounded-md border border-border bg-muted/30 p-3 text-xs space-y-1">
        <p class="text-muted-foreground">Example rewrite</p>
        <p class="break-all font-mono">{{ exampleOriginal }}</p>
        <p class="text-muted-foreground">→</p>
        <p class="break-all font-mono text-foreground">{{ preview }}</p>
      </div>
      <p v-if="error" class="text-sm text-rose-400">{{ error }}</p>
      <p v-if="saved" class="text-sm text-emerald-600 dark:text-emerald-400">Saved</p>
      <div class="flex flex-wrap gap-2">
        <Button class="h-8" @click="save">Save</Button>
        <Button variant="outline" class="h-8" @click="clear">Clear</Button>
      </div>
    </section>
  </div>
</template>
