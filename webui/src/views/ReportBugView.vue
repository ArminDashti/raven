<script setup lang="ts">
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { api } from '@/lib/api'
import { getUser } from '@/lib/auth'
import { canReport, isReporterRole } from '@/lib/permissions'
import { autoDir } from '@/lib/format'
import { displayPersonName } from '@/lib/status'
import { Paperclip, Send, X } from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const MAX_FILES = 10
const MAX_FILE_BYTES = 10 * 1024 * 1024
const PAGE_PARAMS_KEY = 'bug_report_page_parameters'

const router = useRouter()
const url = ref('')
const description = ref('')
const pageParameters = ref('')
const priority = ref('normal')
const reporterUserId = ref('')
const selectedFiles = ref<File[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const reporters = ref<
  Array<{ id: number; username: string; role: string; first_name?: string; last_name?: string }>
>([])
const error = ref('')
const loading = ref(false)
const user = getUser()

const descriptionDir = computed(() => autoDir(description.value))
const paramsDir = computed(() => autoDir(pageParameters.value))

function savePageParameters() {
  if (typeof localStorage === 'undefined') return
  localStorage.setItem(PAGE_PARAMS_KEY, pageParameters.value.trim())
}

onMounted(async () => {
  if (!canReport(user?.role)) {
    await router.replace({ name: 'bugs' })
    return
  }
  if (typeof localStorage !== 'undefined') {
    pageParameters.value = localStorage.getItem(PAGE_PARAMS_KEY) || ''
  }
  const users = await api.users()
  reporters.value = users.filter((u) => isReporterRole(u.role))
  if (canReport(user?.role) && user) {
    reporterUserId.value = String(user.id)
  }
})

function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
}

function onFileChange(e: Event) {
  error.value = ''
  const input = e.target as HTMLInputElement
  const incoming = input.files ? Array.from(input.files) : []
  const next = [...selectedFiles.value]
  for (const f of incoming) {
    if (next.length >= MAX_FILES) {
      error.value = `At most ${MAX_FILES} files allowed`
      break
    }
    if (f.size > MAX_FILE_BYTES) {
      error.value = `"${f.name}" exceeds ${formatSize(MAX_FILE_BYTES)} limit`
      continue
    }
    const dup = next.some((x) => x.name === f.name && x.size === f.size && x.lastModified === f.lastModified)
    if (!dup) next.push(f)
  }
  selectedFiles.value = next
  if (fileInput.value) fileInput.value.value = ''
}

function removeFile(index: number) {
  selectedFiles.value = selectedFiles.value.filter((_, i) => i !== index)
}

function clearFiles() {
  selectedFiles.value = []
  if (fileInput.value) fileInput.value.value = ''
}

async function onSubmit() {
  error.value = ''
  if (
    !url.value.trim() ||
    !description.value.trim() ||
    !pageParameters.value.trim() ||
    !reporterUserId.value
  ) {
    error.value = 'All fields except attachments are required'
    return
  }
  if (selectedFiles.value.length > MAX_FILES) {
    error.value = `At most ${MAX_FILES} files allowed`
    return
  }
  for (const f of selectedFiles.value) {
    if (f.size > MAX_FILE_BYTES) {
      error.value = `"${f.name}" exceeds ${formatSize(MAX_FILE_BYTES)} limit`
      return
    }
  }
  loading.value = true
  try {
    const form = new FormData()
    form.append('url', url.value.trim())
    form.append('description', description.value.trim())
    form.append('page_parameters', pageParameters.value.trim())
    form.append('priority', priority.value)
    form.append('reporter_user_id', reporterUserId.value)
    selectedFiles.value.forEach((f) => form.append('attachments', f))
    await api.createBug(form)
    savePageParameters()
    await router.push({ name: 'bugs' })
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Submit failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="w-full px-4 py-8">
    <h1 class="text-2xl font-semibold mb-6">Report Bug</h1>
    <form class="max-w-2xl space-y-3" @submit.prevent="onSubmit">
      <div class="field-panel-stack">
        <label class="field-panel-label">URL *</label>
        <Input v-model="url" required placeholder="https://…" />
      </div>
      <div class="field-panel-stack">
        <label class="field-panel-label">Bugs *</label>
        <Textarea
          v-model="description"
          required
          :dir="descriptionDir"
          :rows="8"
          class="min-h-[160px]"
        />
      </div>
      <div class="field-panel-stack">
        <label class="field-panel-label">Page parameters *</label>
        <Textarea
          v-model="pageParameters"
          required
          :dir="paramsDir"
          @blur="savePageParameters"
        />
      </div>
      <div class="field-panel-row">
        <label class="field-panel-label">Priority *</label>
        <select
          v-model="priority"
          required
          class="flex h-9 max-w-[14rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
        >
          <option value="low">Low</option>
          <option value="normal">Medium</option>
          <option value="high">High</option>
        </select>
      </div>
      <div class="field-panel-row">
        <label class="field-panel-label">Reporter *</label>
        <select
          v-model="reporterUserId"
          required
          class="flex h-9 max-w-[14rem] rounded-md border border-input bg-background px-3 text-sm text-foreground"
        >
          <option disabled value="">Select reporter</option>
          <option v-for="r in reporters" :key="r.id" :value="String(r.id)">
            {{ displayPersonName(r.username) }} ({{ r.role }})
          </option>
        </select>
      </div>
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-2">
          <label class="text-sm inline-flex items-center gap-1.5">
            <Paperclip class="h-3.5 w-3.5 opacity-70" :stroke-width="1.75" />
            Attachments (optional, up to {{ MAX_FILES }} files, {{ formatSize(MAX_FILE_BYTES) }} each)
          </label>
          <button
            v-if="selectedFiles.length"
            type="button"
            class="text-xs text-muted-foreground hover:text-foreground underline"
            @click="clearFiles"
          >
            Clear all
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          multiple
          class="block w-full text-sm text-muted-foreground file:mr-4 file:rounded-md file:border-0 file:bg-secondary file:px-3 file:py-1.5 file:text-sm"
          @change="onFileChange"
        />
        <ul v-if="selectedFiles.length" class="space-y-1 rounded-md border border-border p-2 text-sm">
          <li
            v-for="(f, i) in selectedFiles"
            :key="`${f.name}-${f.size}-${f.lastModified}`"
            class="flex items-center justify-between gap-2"
          >
            <span class="truncate">{{ f.name }} <span class="text-muted-foreground">({{ formatSize(f.size) }})</span></span>
            <button
              type="button"
              class="shrink-0 inline-flex items-center gap-1 text-xs text-rose-400 hover:underline"
              @click="removeFile(i)"
            >
              <X class="h-3 w-3" :stroke-width="1.75" />
              Remove
            </button>
          </li>
        </ul>
      </div>
      <p v-if="error" class="text-sm text-rose-400">{{ error }}</p>
      <Button type="submit" class="inline-flex items-center gap-1.5" :disabled="loading">
        <Send class="h-3.5 w-3.5" :stroke-width="1.75" />
        {{ loading ? 'Submitting…' : 'Submit' }}
      </Button>
    </form>
  </div>
</template>
